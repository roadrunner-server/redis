package kv

import (
	"context"
	"crypto/rand"
	"errors"
	"log/slog"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
	api "github.com/roadrunner-server/api-plugins/v6/jobs"
	"github.com/roadrunner-server/jobs/v6"
	priorityqueue "github.com/roadrunner-server/priority_queue"
	"github.com/roadrunner-server/redis/v6/redisjobs"
	"github.com/stretchr/testify/require"
)

// replyFault fails one Redis operation before execution or after Redis replies.
type replyFault struct {
	armed atomic.Bool
	after bool
}

func TestRedisJobsDelayedDelivery(t *testing.T) {
	cfg := redisjobs.Config{Queue: rand.Text(), Prefetch: 1}
	producer, _ := jobsDriver(t, cfg, nil)
	ctx := t.Context()
	start := time.Now()
	require.NoError(t, producer.Push(ctx, &jobs.Job{
		Ident: "delayed", Job: "send", Options: &jobs.Options{Pipeline: "mail", Delay: 1},
	}))
	state, err := producer.State(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 1, state.Delayed)
	require.Zero(t, state.Reserved)
	require.NoError(t, producer.Stop(ctx))

	reader, q := jobsDriver(t, cfg, nil)
	require.NoError(t, reader.Resume(ctx, "mail"))
	first := takeJob(t, q)
	require.GreaterOrEqual(t, time.Since(start), time.Second-time.Millisecond)
	start = time.Now()
	require.NoError(t, first.Requeue(map[string][]string{"attempt": {"second"}}, 1))
	state, err = reader.State(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 1, state.Delayed)
	require.Zero(t, state.Active)
	second := takeJob(t, q)
	require.GreaterOrEqual(t, time.Since(start), time.Second-time.Millisecond)
	require.Equal(t, "delayed", second.ID())
	require.Equal(t, []string{"second"}, second.Headers()["attempt"])
	require.NoError(t, second.Ack())
}

func (f *replyFault) DialHook(next goredis.DialHook) goredis.DialHook { return next }

func (f *replyFault) ProcessPipelineHook(next goredis.ProcessPipelineHook) goredis.ProcessPipelineHook {
	return next
}

func (f *replyFault) ProcessHook(next goredis.ProcessHook) goredis.ProcessHook {
	return func(ctx context.Context, cmd goredis.Cmder) error {
		if !f.armed.Swap(false) {
			return next(ctx, cmd)
		}
		if f.after {
			if err := next(ctx, cmd); err != nil {
				return err
			}
		}
		return errors.New("connection interrupted")
	}
}

func jobsDriver(t *testing.T, cfg redisjobs.Config, hook goredis.Hook) (api.Driver, *priorityqueue.BinHeap[api.Job]) {
	t.Helper()
	addr := os.Getenv("RR_REDIS_JOBS_ADDR")
	if addr == "" {
		addr = redisAddr
	}
	client := goredis.NewClient(&goredis.Options{Addr: addr, DB: 2})
	if hook != nil {
		client.AddHook(hook)
	}
	q := priorityqueue.NewBinHeap[api.Job](10)
	d, err := redisjobs.New(t.Context(), client, slog.Default(), q, "mail", cfg)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, d.Stop(context.Background())) })
	return d, q
}

func TestRedisJobsClusterDelayedDuplicates(t *testing.T) {
	addrs := os.Getenv("RR_REDIS_TEST_CLUSTER")
	if addrs == "" {
		t.Skip("set RR_REDIS_TEST_CLUSTER to the Redis Cluster addresses")
	}
	cfg := redisjobs.Config{Queue: rand.Text(), Prefetch: 1}
	q := priorityqueue.NewBinHeap[api.Job](2)
	reader := func() api.Driver {
		t.Helper()
		client := goredis.NewClusterClient(&goredis.ClusterOptions{Addrs: strings.Split(addrs, ",")})
		d, err := redisjobs.New(t.Context(), client, slog.Default(), q, "mail", cfg)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, d.Stop(context.Background())) })
		return d
	}
	a, b := reader(), reader()
	ctx := t.Context()
	job := &jobs.Job{Ident: "same", Job: "mail", Options: &jobs.Options{Delay: 1}}
	require.NoError(t, a.Push(ctx, job))
	require.NoError(t, a.Push(ctx, job))
	require.NoError(t, a.Resume(ctx, "mail"))
	require.NoError(t, b.Resume(ctx, "mail"))
	require.Eventually(t, func() bool { return q.Len() == 2 }, 5*time.Second, 10*time.Millisecond)
	for range 2 {
		i := q.ExtractMin()
		require.Equal(t, "same", i.ID())
		require.NoError(t, i.Ack())
	}
	s, err := b.State(ctx)
	require.NoError(t, err)
	require.Zero(t, s.Active)
	require.Zero(t, s.Reserved)
	require.Zero(t, s.Delayed)
}

func TestRedisJobsNack(t *testing.T) {
	d, q := jobsDriver(t, redisjobs.Config{Queue: rand.Text(), Prefetch: 1, VisibilityTimeout: 100 * time.Millisecond}, nil)
	ctx := t.Context()
	require.NoError(t, d.Push(ctx, &jobs.Job{Ident: "recover", Job: "mail"}))
	require.NoError(t, d.Resume(ctx, "mail"))
	require.NoError(t, takeJob(t, q).Nack())
	next := takeJob(t, q)
	require.Equal(t, "recover", next.ID())
	require.NoError(t, next.NackWithOptions(false, 0))
	s, err := d.State(ctx)
	require.NoError(t, err)
	require.Zero(t, s.Active)
	require.Zero(t, s.Reserved)
	require.Zero(t, s.Delayed)
}

func TestRedisJobsPrefetchAndDelayedNack(t *testing.T) {
	d, q := jobsDriver(t, redisjobs.Config{Queue: rand.Text(), Prefetch: 1}, nil)
	ctx := t.Context()
	require.NoError(t, d.Push(ctx, &jobs.Job{Ident: "first", Job: "mail"}))
	require.NoError(t, d.Push(ctx, &jobs.Job{Ident: "second", Job: "mail"}))
	require.NoError(t, d.Resume(ctx, "mail"))
	first := takeJob(t, q)
	require.Equal(t, "first", first.ID())
	require.Never(t, func() bool { return q.Len() != 0 }, 250*time.Millisecond, 10*time.Millisecond)
	s, err := d.State(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 1, s.Active)
	require.EqualValues(t, 1, s.Reserved)
	start := time.Now()
	require.NoError(t, first.NackWithOptions(true, 1))
	second := takeJob(t, q)
	require.Equal(t, "second", second.ID())
	require.NoError(t, second.Ack())
	s, err = d.State(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 1, s.Delayed)
	retry := takeJob(t, q)
	require.GreaterOrEqual(t, time.Since(start), time.Second-time.Millisecond)
	require.Equal(t, "first", retry.ID())
	require.NoError(t, retry.Ack())
	s, err = d.State(ctx)
	require.NoError(t, err)
	require.Zero(t, s.Delayed)
	require.Zero(t, s.Active)
	require.Zero(t, s.Reserved)
}

func takeJob(t *testing.T, q *priorityqueue.BinHeap[api.Job]) api.Job {
	t.Helper()
	require.Eventually(t, func() bool { return q.Len() > 0 }, 5*time.Second, 5*time.Millisecond)
	return q.ExtractMin()
}

func TestRedisJobsStaleDelivery(t *testing.T) {
	cfg := redisjobs.Config{Queue: rand.Text(), Prefetch: 2, VisibilityTimeout: 150 * time.Millisecond}
	a, qa := jobsDriver(t, cfg, nil)
	b, qb := jobsDriver(t, cfg, nil)
	ctx := t.Context()
	require.NoError(t, a.Push(ctx, &jobs.Job{Ident: "123", Job: "mail"}))
	require.NoError(t, a.Resume(ctx, "mail"))
	oldA := takeJob(t, qa)
	require.NoError(t, a.Pause(ctx, "mail"))
	require.NoError(t, b.Resume(ctx, "mail"))
	oldB := takeJob(t, qb)
	require.NoError(t, b.Pause(ctx, "mail"))
	require.NoError(t, a.Resume(ctx, "mail"))
	current := takeJob(t, qa)
	require.Equal(t, "123", current.ID())
	require.NoError(t, oldA.Requeue(nil, 1))
	require.NoError(t, oldB.Ack())
	state, err := a.State(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 1, state.Active)
	require.Zero(t, state.Reserved)
	require.Zero(t, state.Delayed)
	require.NoError(t, current.Ack())
}

func TestRedisJobsAutoAck(t *testing.T) {
	name := rand.Text()
	d, q := jobsDriver(t, redisjobs.Config{Queue: name}, nil)
	ctx := t.Context()
	require.NoError(t, d.Push(ctx, &jobs.Job{Ident: "auto", Job: "mail", Hdr: map[string][]string{"tags": {"one", "two"}}, Options: &jobs.Options{AutoAck: true}}))
	require.NoError(t, d.Resume(ctx, "mail"))
	item := takeJob(t, q)
	info, err := item.Context()
	require.NoError(t, err)
	require.JSONEq(t, `{"id":"auto","job":"mail","driver":"redis","pipeline":"mail","queue":"`+name+`","headers":{"tags":["one","two"]}}`, string(info))
	state, err := d.State(ctx)
	require.NoError(t, err)
	require.Zero(t, state.Active)
	require.Zero(t, state.Reserved)
	require.ErrorContains(t, item.Requeue(nil, 1), "auto-ack")
	state, err = d.State(ctx)
	require.NoError(t, err)
	require.Zero(t, state.Delayed)
}

func TestRedisJobsPauseKeepsAutoAckDelivery(t *testing.T) {
	d, q := jobsDriver(t, redisjobs.Config{Queue: rand.Text(), Prefetch: 11, VisibilityTimeout: 300 * time.Millisecond}, nil)
	ctx := t.Context()
	for range 10 {
		require.NoError(t, d.Push(ctx, &jobs.Job{Ident: rand.Text(), Job: "fill"}))
	}
	require.NoError(t, d.Push(ctx, &jobs.Job{Ident: "blocked", Job: "mail", Options: &jobs.Options{AutoAck: true}}))
	require.NoError(t, d.Resume(ctx, "mail"))
	require.Eventually(t, func() bool {
		s, err := d.State(ctx)
		return err == nil && s.Reserved == 0 && q.Len() == 10
	}, 3*time.Second, 5*time.Millisecond)
	require.NoError(t, d.Pause(ctx, "mail"))
	s, err := d.State(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 11, s.Active, "the blocked job must remain recoverable")
	for range 10 {
		require.NoError(t, q.ExtractMin().Ack())
	}
	require.NoError(t, d.Resume(ctx, "mail"))
	i := takeJob(t, q)
	require.Equal(t, "blocked", i.ID())
	_, err = i.Context()
	require.NoError(t, err)
	require.NoError(t, i.Ack())
}

func TestRedisJobsStopWithWorker(t *testing.T) {
	cases := []struct {
		name   string
		expire bool
	}{
		{name: "worker acknowledges before stop deadline"},
		{name: "unfinished job is recovered after deadline", expire: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := redisjobs.Config{Queue: rand.Text(), Prefetch: 1, VisibilityTimeout: 150 * time.Millisecond}
			d, q := jobsDriver(t, cfg, nil)
			ctx := t.Context()
			require.NoError(t, d.Push(ctx, &jobs.Job{Ident: "active", Job: "mail"}))
			require.NoError(t, d.Resume(ctx, "mail"))
			i := takeJob(t, q)
			_, err := i.Context()
			require.NoError(t, err)
			grace, cancel := context.WithTimeout(ctx, 4*time.Second)
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- d.Stop(grace) }()
			require.Eventually(t, func() bool {
				s, err := d.State(ctx)
				return err != nil || !s.Ready
			}, time.Second, 5*time.Millisecond)
			if tc.expire {
				cancel()
				require.ErrorIs(t, <-done, context.Canceled)
			} else {
				require.NoError(t, i.Ack())
				require.NoError(t, <-done)
			}
			b, qb := jobsDriver(t, cfg, nil)
			if tc.expire {
				require.NoError(t, b.Resume(ctx, "mail"))
				recovered := takeJob(t, qb)
				require.Equal(t, "active", recovered.ID())
				require.NoError(t, recovered.Ack())
			}
			s, err := b.State(ctx)
			require.NoError(t, err)
			require.Zero(t, s.Active)
			require.Zero(t, s.Reserved)
		})
	}
}

func TestRedisJobsPauseFullHeap(t *testing.T) {
	d, q := jobsDriver(t, redisjobs.Config{Queue: rand.Text(), Prefetch: 11}, nil)
	t.Cleanup(func() { q.Remove("mail") })
	ctx := t.Context()
	for range 11 {
		require.NoError(t, d.Push(ctx, &jobs.Job{Ident: rand.Text(), Job: "mail"}))
	}
	require.NoError(t, d.Resume(ctx, "mail"))
	require.Eventually(t, func() bool {
		s, err := d.State(ctx)
		return err == nil && s.Active == 11 && q.Len() == 10
	}, 5*time.Second, 10*time.Millisecond)
	deadline, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	require.NoError(t, d.Pause(deadline, "mail"))
}

type blockedRead struct {
	replyFault
	once    sync.Once
	entered chan struct{}
	release chan struct{}
}

func (b *blockedRead) ProcessHook(next goredis.ProcessHook) goredis.ProcessHook {
	return func(ctx context.Context, cmd goredis.Cmder) error {
		if cmd.Name() == "xreadgroup" {
			b.once.Do(func() {
				close(b.entered)
				<-b.release
			})
		}
		return next(ctx, cmd)
	}
}

func TestRedisJobsResumeAfterPauseDeadline(t *testing.T) {
	hook := &blockedRead{entered: make(chan struct{}), release: make(chan struct{})}
	d, q := jobsDriver(t, redisjobs.Config{Queue: rand.Text()}, hook)
	release := sync.OnceFunc(func() { close(hook.release) })
	t.Cleanup(release)
	ctx := t.Context()
	require.NoError(t, d.Resume(ctx, "mail"))
	select {
	case <-hook.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("consumer did not start reading")
	}
	expired, cancel := context.WithCancel(ctx)
	cancel()
	require.ErrorIs(t, d.Pause(expired, "mail"), context.Canceled)
	release()
	require.NoError(t, d.Resume(ctx, "mail"))
	require.NoError(t, d.Push(ctx, &jobs.Job{Ident: "after-pause", Job: "mail"}))
	i := takeJob(t, q)
	require.Equal(t, "after-pause", i.ID())
	require.NoError(t, i.Ack())
}

func TestRedisJobsInterruptedRequeue(t *testing.T) {
	cases := []struct {
		name       string
		afterWrite bool
		wantHeader string
	}{
		{name: "request lost before commit", wantHeader: "original"},
		{name: "reply lost after commit", afterWrite: true, wantHeader: "retry"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := redisjobs.Config{Queue: rand.Text(), Prefetch: 1, VisibilityTimeout: 100 * time.Millisecond}
			fault := &replyFault{after: tc.afterWrite}
			a, first := jobsDriver(t, cfg, fault)
			ctx := t.Context()
			payload := []byte{0, 255, 42}
			require.NoError(t, a.Push(ctx, &jobs.Job{
				Ident: "123", Job: "mail", Pld: payload, Hdr: map[string][]string{"attempt": {"original"}, "pool": {"fast"}},
				Options: &jobs.Options{Pipeline: "mail", Priority: 7},
			}))
			require.NoError(t, a.Resume(ctx, "mail"))
			item := takeJob(t, first)
			require.NoError(t, a.Pause(ctx, "mail"))
			fault.armed.Store(true)
			require.ErrorContains(t, item.Requeue(map[string][]string{"attempt": {"retry"}}, 0), "connection interrupted")
			require.NoError(t, item.Ack()) // Jobs calls Ack after a failed Requeue.

			b, second := jobsDriver(t, cfg, nil)
			require.NoError(t, b.Resume(ctx, "mail"))
			recovered := takeJob(t, second)
			require.Equal(t, "123", recovered.ID())
			require.Equal(t, payload, recovered.Body())
			require.Equal(t, int64(7), recovered.Priority())
			require.Equal(t, map[string][]string{"attempt": {tc.wantHeader}, "pool": {"fast"}}, recovered.Headers())
			require.NoError(t, recovered.Ack())
			state, err := b.State(ctx)
			require.NoError(t, err)
			require.Zero(t, state.Active)
			require.Zero(t, state.Reserved)
			require.Zero(t, state.Delayed)
		})
	}
}
