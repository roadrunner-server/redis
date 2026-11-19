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

func (d *Driver) listen(ctx context.Context, done chan struct{}) {
	defer close(done)
	cursor := "0-0"
	var nextScan time.Time
	for ctx.Err() == nil {
		select {
		case d.slots <- struct{}{}:
		case <-ctx.Done():
			return
		}
		i, err := d.read(ctx, &cursor, &nextScan)
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

func (d *Driver) read(ctx context.Context, cursor *string, nextScan *time.Time) (*item, error) {
	if *cursor != "0-0" || !time.Now().Before(*nextScan) {
		const promote = `
local now = redis.call('TIME')
local due = redis.call('ZRANGEBYSCORE', KEYS[2], '-inf', now[1] * 1000 + math.floor(now[2] / 1000), 'LIMIT', 0, 100)
for _, job in ipairs(due) do
  redis.call('XADD', KEYS[1], '*', 'job', job)
  redis.call('ZREM', KEYS[2], job)
end
return #due`
		if err := d.client.Eval(ctx, promote, d.keys()).Err(); err != nil {
			return nil, err
		}
		messages, next, err := d.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{
			Stream: d.stream, Group: group, Consumer: d.consumer, Start: *cursor, MinIdle: d.cfg.VisibilityTimeout, Count: 1,
		}).Result()
		if err != nil {
			return nil, err
		}
		*cursor = next
		*nextScan = time.Now().Add(readInterval)
		if len(messages) > 0 {
			m := messages[0]
			p, err := d.client.XPendingExt(ctx, &redis.XPendingExtArgs{Stream: d.stream, Group: group, Start: m.ID, End: m.ID, Count: 1}).Result()
			if err != nil || len(p) == 0 || p[0].Consumer != d.consumer {
				return nil, err
			}
			return d.unpack(m, p[0].RetryCount)
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
	return d.unpack(streams[0].Messages[0], 1)
}

func (d *Driver) unpack(m redis.XMessage, delivery int64) (*item, error) {
	i := &item{entry: m.ID, delivery: delivery, driver: d}
	raw, ok := m.Values["job"].(string)
	if !ok {
		return nil, fmt.Errorf("redis stream entry %s has no job", m.ID)
	}
	if err := json.Unmarshal([]byte(raw), &i.record); err != nil {
		return nil, err
	}
	return i, nil
}
