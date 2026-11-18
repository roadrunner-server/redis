package redisjobs

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/roadrunner-server/api-plugins/v6/jobs"
	priorityqueue "github.com/roadrunner-server/priority_queue"
)

const group = "roadrunner"

type Config struct {
	Queue             string        `mapstructure:"queue"`
	Priority          int64         `mapstructure:"priority"`
	Prefetch          int           `mapstructure:"prefetch"`
	VisibilityTimeout time.Duration `mapstructure:"visibility_timeout"`
}

type deliveryQueue interface {
	jobs.Queue
	InsertContext(context.Context, jobs.Job) error
}

var _ deliveryQueue = (*priorityqueue.BinHeap[jobs.Job])(nil)

type Driver struct {
	client   redis.UniversalClient
	log      *slog.Logger
	queue    deliveryQueue
	pipeline string
	cfg      Config
	stream   string
	delayed  string
	consumer string
	slots    chan struct{}

	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
	closed bool
}

// New takes ownership of the client after it creates the pipeline.
func New(ctx context.Context, client redis.UniversalClient, log *slog.Logger, q jobs.Queue, pipeline string, cfg Config) (*Driver, error) {
	heap, ok := q.(deliveryQueue)
	if !ok {
		return nil, errors.New("redis jobs: the Jobs queue must support InsertContext")
	}
	if cfg.Queue == "" {
		cfg.Queue = pipeline
	}
	if cfg.Queue == "" || strings.ContainsAny(cfg.Queue, "{}") {
		return nil, errors.New("redis jobs: queue must be nonempty and cannot contain braces")
	}
	if cfg.Priority == 0 {
		cfg.Priority = 10
	}
	if cfg.Prefetch == 0 {
		cfg.Prefetch = 10
	}
	if cfg.VisibilityTimeout == 0 {
		cfg.VisibilityTimeout = 5 * time.Minute
	}
	if cfg.Prefetch < 1 || cfg.VisibilityTimeout < time.Millisecond {
		return nil, errors.New("redis jobs: prefetch must be positive and visibility_timeout must be at least 1ms")
	}
	if cfg.Priority < 0 {
		return nil, errors.New("redis jobs: pipeline priority must not be negative")
	}
	info, err := client.InfoMap(ctx, "server").Result()
	if err != nil {
		return nil, err
	}
	version := info["Server"]["redis_version"]
	major, _, _ := strings.Cut(version, ".")
	if n, err := strconv.Atoi(major); err != nil || n < 8 {
		return nil, fmt.Errorf("redis jobs requires Redis 8.0 or later, got %q", version)
	}
	prefix := "rr:jobs:{" + cfg.Queue + "}:"
	d := &Driver{
		client: client, log: log, queue: heap, pipeline: pipeline, cfg: cfg,
		stream: prefix + "ready", delayed: prefix + "delayed", consumer: rand.Text(), slots: make(chan struct{}, cfg.Prefetch),
	}
	if err := client.XGroupCreateMkStream(ctx, d.stream, group, "0").Err(); err != nil && !redis.HasErrorPrefix(err, "BUSYGROUP") {
		return nil, err
	}
	return d, nil
}

func (d *Driver) Push(ctx context.Context, m jobs.Message) error {
	r := record{ID: m.ID(), Name: m.Name(), Payload: m.Payload(), Headers: m.Headers(), Priority: m.Priority(), AutoAck: m.AutoAck()}
	if r.Priority == 0 {
		r.Priority = d.cfg.Priority
	}
	data, err := r.encode(m.Delay())
	if err != nil {
		return err
	}
	if m.Delay() > 0 {
		return d.client.Eval(ctx, enqueue+`return enqueue(ARGV[1], ARGV[2])`, d.keys(), data, m.Delay()).Err()
	}
	return d.client.XAdd(ctx, &redis.XAddArgs{Stream: d.stream, Values: []any{"job", data}}).Err()
}

func (d *Driver) keys() []string { return []string{d.stream, d.delayed} }

const enqueue = `
local function enqueue(job, delay)
  if tonumber(delay) == 0 then return redis.call('XADD', KEYS[1], '*', 'job', job) end
  local now = redis.call('TIME')
  local due = (now[1] + delay) * 1000 + math.floor(now[2] / 1000)
  return redis.call('ZADD', KEYS[2], due, job)
end
`

func (d *Driver) Run(ctx context.Context, p jobs.Pipeline) error {
	return d.Resume(ctx, p.Name())
}

func (d *Driver) Resume(ctx context.Context, _ string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return errors.New("redis jobs: pipeline stopped")
	}
	if err := ctx.Err(); err != nil || d.cancel != nil {
		return err
	}
	if err := d.wait(ctx); err != nil {
		return err
	}
	run, cancel := context.WithCancel(context.Background())
	d.cancel, d.done = cancel, make(chan struct{})
	go d.listen(run, d.done)
	return nil
}

func (d *Driver) Pause(ctx context.Context, _ string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.pause(ctx)
}

func (d *Driver) pause(ctx context.Context) error {
	if d.cancel != nil {
		d.cancel()
		d.cancel = nil
	}
	return d.wait(ctx)
}

func (d *Driver) wait(ctx context.Context) error {
	if d.done == nil {
		return nil
	}
	select {
	case <-d.done:
		d.done = nil
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (d *Driver) Stop(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return nil
	}
	d.closed = true
	err := d.pause(ctx)
	for _, buffered := range d.queue.Remove(d.pipeline) {
		_ = buffered.Nack()
	}
	return errors.Join(err, d.client.Close())
}

func (d *Driver) State(ctx context.Context) (*jobs.State, error) {
	const state = `
local pending = redis.call('XPENDING', KEYS[1], ARGV[1])[1]
return {pending, redis.call('XLEN', KEYS[1]) - pending, redis.call('ZCARD', KEYS[2])}`
	counts, err := d.client.Eval(ctx, state, d.keys(), group).Int64Slice()
	if err != nil {
		return nil, err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	return &jobs.State{Pipeline: d.pipeline, Driver: "redis", Queue: d.cfg.Queue, Ready: d.cancel != nil,
		Priority: uint64(d.cfg.Priority), Active: counts[0], Reserved: counts[1], Delayed: counts[2]}, nil //nolint:gosec
}
