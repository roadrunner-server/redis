package redisjobs

import (
	"context"
	"crypto/rand"
	"encoding/json/v2"
	"errors"
	"maps"
	"sync"
)

type record struct {
	Nonce    string              `json:"nonce,omitempty"`
	ID       string              `json:"id"`
	Name     string              `json:"job"`
	Payload  []byte              `json:"payload"`
	Headers  map[string][]string `json:"headers"`
	Priority int64               `json:"priority"`
	AutoAck  bool                `json:"auto_ack"`
}

func (r record) encode(delay int64) ([]byte, error) {
	if delay < 0 {
		return nil, errors.New("redis jobs: delay must not be negative")
	}
	if delay > 0 {
		// Each delayed push needs a distinct sorted set member, even for identical jobs.
		r.Nonce = rand.Text()
	}
	return json.Marshal(r)
}

type item struct {
	record   record
	driver   *Driver
	entry    string
	delivery int64
	settled  sync.Once
}

func (i *item) ID() string                   { return i.record.ID }
func (i *item) GroupID() string              { return i.driver.pipeline }
func (i *item) Priority() int64              { return i.record.Priority }
func (i *item) Body() []byte                 { return i.record.Payload }
func (i *item) Headers() map[string][]string { return i.record.Headers }

func (i *item) Context() ([]byte, error) {
	return json.Marshal(struct {
		ID       string              `json:"id"`
		Job      string              `json:"job"`
		Driver   string              `json:"driver"`
		Pipeline string              `json:"pipeline"`
		Queue    string              `json:"queue"`
		Headers  map[string][]string `json:"headers"`
	}{i.ID(), i.record.Name, "redis", i.GroupID(), i.driver.cfg.Queue, i.Headers()})
}

// A worker transport error leaves the delivery pending until the visibility timeout.
func (i *item) Nack() error {
	i.settled.Do(func() { <-i.driver.slots })
	return nil
}

func (i *item) Ack() error { return i.finish(false, nil, 0) }

func (i *item) NackWithOptions(requeue bool, delay int) error {
	return i.finish(requeue, nil, delay)
}

func (i *item) Requeue(headers map[string][]string, delay int) error {
	return i.finish(true, headers, delay)
}

func (i *item) finish(requeue bool, headers map[string][]string, delay int) error {
	var err error
	// Jobs can call Ack after a failed Requeue. Redis must retain that delivery.
	i.settled.Do(func() {
		defer func() { <-i.driver.slots }()
		if i.record.AutoAck {
			return
		}
		var replacement []byte
		if requeue {
			r := i.record
			if len(headers) != 0 {
				r.Headers = make(map[string][]string, len(r.Headers)+len(headers))
				maps.Copy(r.Headers, i.record.Headers)
				maps.Copy(r.Headers, headers)
			}
			replacement, err = r.encode(int64(delay))
			if err != nil {
				return
			}
		}
		_, err = i.settle(context.Background(), replacement, delay)
	})
	return err
}

func (i *item) settle(ctx context.Context, replacement []byte, delay int) (int64, error) {
	// XPENDING includes the owner and delivery count. https://redis.io/docs/latest/commands/xpending/
	const settle = enqueue + `
local p = redis.call('XPENDING', KEYS[1], ARGV[1], ARGV[2], ARGV[2], 1)[1]
if not p or p[2] ~= ARGV[3] or p[4] ~= tonumber(ARGV[4]) then return 0 end
if ARGV[5] ~= '' then enqueue(ARGV[5], ARGV[6]) end
redis.call('XACK', KEYS[1], ARGV[1], ARGV[2])
return redis.call('XDEL', KEYS[1], ARGV[2])`
	return i.driver.client.Eval(ctx, settle, i.driver.keys(), group, i.entry, i.driver.consumer, i.delivery, replacement, delay).Int64()
}
