package redisjobs

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const readInterval = 200 * time.Millisecond

type pendingScan struct {
	after string
	tail  string
	next  time.Time
}

func (d *Driver) listen(ctx context.Context, done chan struct{}) {
	defer close(done)
	var scan pendingScan
	for ctx.Err() == nil {
		select {
		case d.slots <- struct{}{}:
		case <-ctx.Done():
			return
		}
		i, err := d.read(ctx, &scan)
		if redis.HasErrorPrefix(err, "NOGROUP") {
			scan = pendingScan{}
			err = d.ensureGroup(ctx)
		}
		if err == nil && i != nil && ctx.Err() == nil {
			err = d.queue.InsertContext(ctx, i)
			if err == nil {
				continue
			}
		}
		<-d.slots
		if err != nil && !errors.Is(err, redis.Nil) && ctx.Err() == nil {
			d.log.Error("read Redis job", "pipeline", d.pipeline, "error", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(readInterval):
			}
		}
	}
}

func (d *Driver) read(ctx context.Context, scan *pendingScan) (*item, error) {
	if scan.tail != "" || !time.Now().Before(scan.next) {
		const promote = redisClock + `
local due = redis.call('ZRANGE', KEYS[2], '-inf', now, 'BYSCORE', 'LIMIT', 0, 100)
for _, job in ipairs(due) do
  redis.call('XADD', KEYS[1], '*', 'job', job)
  redis.call('ZREM', KEYS[2], job)
end
return #due`
		// When promotion needs more memory, consuming ready jobs can free it.
		if err := d.client.Eval(ctx, promote, d.keys()).Err(); err != nil && !redis.HasErrorPrefix(err, "OOM") {
			return nil, err
		}
		i, err := d.claim(ctx, scan)
		if err != nil || i != nil {
			return i, err
		}
	}
	streams, err := d.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group: group, Consumer: d.consumer, Streams: []string{d.stream, ">"}, Count: 1, Block: readInterval,
	}).Result()
	if err != nil {
		return nil, err
	}
	if len(streams) == 0 || len(streams[0].Messages) == 0 {
		return nil, nil
	}
	m := streams[0].Messages[0]
	raw, ok := m.Values["job"].(string)
	if !ok {
		return nil, fmt.Errorf("redis stream entry %s has no job", m.ID)
	}
	return d.unpack(m.ID, "1", raw)
}

func (d *Driver) claim(ctx context.Context, scan *pendingScan) (*item, error) {
	// Use the pending tail at the start of each scan. https://redis.io/docs/latest/commands/xpending/
	// XCLAIM increments the delivery count on the primary. https://redis.io/docs/latest/commands/xclaim/
	const claim = `
local start, tail = '-', ARGV[5]
if tail == '' then
  tail = redis.call('XPENDING', KEYS[1], ARGV[1])[3]
  if not tail then return {} end
else
  start = '(' .. ARGV[4]
end
local page = redis.call('XPENDING', KEYS[1], ARGV[1], start, tail, 10)
for _, p in ipairs(page) do
  if p[3] >= tonumber(ARGV[3]) then
    local m = redis.call('XCLAIM', KEYS[1], ARGV[1], ARGV[2], ARGV[3], p[1])[1]
    if m then return {p[1], tail, tostring(p[4] + 1), m[2][2]} end
  end
end
if #page == 10 then return {page[10][1], tail} end
return {}`
	reply, err := d.client.Eval(ctx, claim, []string{d.stream}, group, d.consumer, d.cfg.VisibilityTimeout.Milliseconds(), scan.after, scan.tail).StringSlice()
	if err != nil {
		return nil, err
	}
	*scan = pendingScan{next: time.Now().Add(readInterval)}
	if len(reply) == 0 {
		return nil, nil
	}
	if reply[0] != reply[1] {
		scan.after, scan.tail = reply[0], reply[1]
	}
	if len(reply) == 2 {
		return nil, nil
	}
	return d.unpack(reply[0], reply[2], reply[3])
}

func (d *Driver) unpack(entry, delivery, raw string) (*item, error) {
	i := &item{entry: entry, delivery: delivery, driver: d}
	if err := json.Unmarshal([]byte(raw), &i.record); err != nil {
		return nil, err
	}
	return i, nil
}
