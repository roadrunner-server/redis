package kv

import (
	"context"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/roadrunner-server/jobs/v6"
	"github.com/roadrunner-server/redis/v6/redisjobs"
	"github.com/stretchr/testify/require"
)

// Tests that clear databases or change server settings use a private Unix socket.
func privateRedis(t *testing.T) *redis.Client {
	t.Helper()
	dir := t.TempDir()
	socket := filepath.Join(dir, "r.sock")
	ctx, stop := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, "redis-server", "--port", "0", "--unixsocket", socket,
		"--dir", dir, "--save", "", "--appendonly", "no")
	t.Cleanup(func() {
		stop()
		_ = cmd.Wait()
	})
	require.NoError(t, cmd.Start())
	c := redis.NewClient(&redis.Options{Addr: socket, DB: 2, MaxRetries: -1})
	t.Cleanup(func() { _ = c.Close() })
	require.Eventually(t, func() bool { return c.Ping(t.Context()).Err() == nil }, 5*time.Second, 10*time.Millisecond)
	return c
}

func TestRedisJobsMissingConsumerGroup(t *testing.T) {
	cases := []struct {
		name    string
		running bool
	}{
		{name: "reader recovers jobs accepted after group loss", running: true},
		{name: "producer reports the retained backlog"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := privateRedis(t)
			t.Setenv("RR_REDIS_JOBS_ADDR", c.Options().Addr)
			ctx := t.Context()
			d, q := jobsDriver(t, redisjobs.Config{Queue: "mail"}, nil)
			if tc.running {
				require.NoError(t, d.Resume(ctx, "mail"))
				require.NoError(t, d.Push(ctx, &jobs.Job{Ident: "before", Job: "mail"}))
				require.NoError(t, takeJob(t, q).Ack())
			}
			require.EqualValues(t, 1, c.XGroupDestroy(ctx, "rr:jobs:{mail}:ready", "roadrunner").Val())
			require.NoError(t, d.Push(ctx, &jobs.Job{Ident: "after", Job: "mail"}))
			if !tc.running {
				s, err := d.State(ctx)
				require.NoError(t, err)
				require.EqualValues(t, 1, s.Reserved)
				require.Zero(t, s.Active)
				require.NoError(t, d.Resume(ctx, "mail"))
			}
			i := takeJob(t, q)
			require.Equal(t, "after", i.ID())
			require.NoError(t, i.Ack())
		})
	}
}

func TestRedisJobsDrainReadyAtMemoryLimit(t *testing.T) {
	c := privateRedis(t)
	t.Setenv("RR_REDIS_JOBS_ADDR", c.Options().Addr)
	d, q := jobsDriver(t, redisjobs.Config{Queue: "mail", Prefetch: 1}, nil)
	ctx := t.Context()
	require.NoError(t, d.Push(ctx, &jobs.Job{Ident: "ready", Job: "mail"}))
	require.NoError(t, d.Push(ctx, &jobs.Job{Ident: "due", Job: "mail", Options: &jobs.Options{Delay: 1}}))
	require.Eventually(t, func() bool {
		now, err := c.Time(ctx).Result()
		return err == nil && c.ZCount(ctx, "rr:jobs:{mail}:delayed", "-inf", strconv.FormatInt(now.UnixMilli(), 10)).Val() == 1
	}, 3*time.Second, 10*time.Millisecond)
	require.NoError(t, c.ConfigSet(ctx, "maxmemory", "1").Err())
	require.NoError(t, d.Resume(ctx, "mail"))
	ready := takeJob(t, q)
	require.Equal(t, "ready", ready.ID())
	require.NoError(t, ready.Ack())
	require.NoError(t, c.ConfigSet(ctx, "maxmemory", "0").Err())
	due := takeJob(t, q)
	require.Equal(t, "due", due.ID())
	require.NoError(t, due.Ack())
}
