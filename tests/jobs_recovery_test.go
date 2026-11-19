package kv

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	api "github.com/roadrunner-server/api-plugins/v6/jobs"
	"github.com/roadrunner-server/jobs/v6"
	priorityqueue "github.com/roadrunner-server/priority_queue"
	"github.com/roadrunner-server/redis/v6/redisjobs"
	"github.com/stretchr/testify/require"
)

func TestRedisJobsRecoveryWithMovingTail(t *testing.T) {
	c := privateRedis(t)
	t.Setenv("RR_REDIS_JOBS_ADDR", c.Options().Addr)
	ctx := t.Context()
	d, q := jobsDriver(t, redisjobs.Config{Queue: "mail", Prefetch: 1, VisibilityTimeout: time.Hour}, nil)
	const stream = "rr:jobs:{mail}:ready"
	const group = "roadrunner"
	require.NoError(t, d.Push(ctx, &jobs.Job{Ident: "abandoned", Job: "mail"}))
	abandoned, err := c.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group: group, Consumer: "old", Streams: []string{stream, ">"}, Count: 1, Block: -1,
	}).Result()
	require.NoError(t, err)
	id := abandoned[0].Messages[0].ID

	// Each peer takes a new job and finishes its previous one while the tested worker is busy.
	var previous []string
	advance := func() {
		t.Helper()
		const batch = `
local ids = {}
for n = 1, 12 do
  ids[n] = redis.call('XADD', KEYS[1], '*', 'job', '{"id":"peer","job":"mail"}')
  redis.call('XREADGROUP','GROUP',ARGV[1],'peer-'..n,'COUNT',1,'STREAMS',KEYS[1],'>')
end
return ids`
		ids, err := c.Eval(ctx, batch, []string{stream}, group).StringSlice()
		require.NoError(t, err)
		if len(previous) != 0 {
			require.NoError(t, c.XAck(ctx, stream, group, previous...).Err())
			require.NoError(t, c.XDel(ctx, stream, previous...).Err())
		}
		previous = ids
		require.NoError(t, d.Push(ctx, &jobs.Job{Ident: "foreground", Job: "mail"}))
	}
	advance()
	require.NoError(t, d.Resume(ctx, "mail"))
	current := takeJob(t, q)
	require.Equal(t, "foreground", current.ID())
	require.NoError(t, c.Do(ctx, "XCLAIM", stream, group, "old", 0, id, "IDLE", (2*time.Hour).Milliseconds(), "JUSTID").Err())
	deadline := time.Now().Add(2 * time.Second)
	for current.ID() == "foreground" && time.Now().Before(deadline) {
		advance()
		require.NoError(t, current.Ack())
		current = takeJob(t, q)
	}
	require.Equal(t, "abandoned", current.ID())
	require.NoError(t, current.Ack())
}

// replicaReads models a read-only command routed to a replica at an earlier replication offset.
type replicaReads struct {
	redis.UniversalClient
	replica *redis.Client
}

func (r replicaReads) XPendingExt(ctx context.Context, a *redis.XPendingExtArgs) *redis.XPendingExtCmd {
	return r.replica.XPendingExt(ctx, a)
}

func TestRedisJobsClaimWithReplicaLag(t *testing.T) {
	c := privateRedis(t)
	t.Setenv("RR_REDIS_JOBS_ADDR", c.Options().Addr)
	cfg := redisjobs.Config{Queue: "mail", Prefetch: 1, VisibilityTimeout: 50 * time.Millisecond}
	a, first := jobsDriver(t, cfg, nil)
	ctx := t.Context()
	require.NoError(t, a.Push(ctx, &jobs.Job{Ident: "lost", Job: "mail"}))
	require.NoError(t, a.Resume(ctx, "mail"))
	require.Equal(t, "lost", takeJob(t, first).ID())
	require.NoError(t, a.Stop(ctx))
	const stream = "rr:jobs:{mail}:ready"
	require.EqualValues(t, 1, c.Copy(ctx, stream, stream, 3, true).Val())
	replica := redis.NewClient(&redis.Options{Addr: c.Options().Addr, DB: 3})
	t.Cleanup(func() { _ = replica.Close() })
	client := redis.NewClient(&redis.Options{Addr: c.Options().Addr, DB: 2})
	q := priorityqueue.NewBinHeap[api.Job](1)
	b, err := redisjobs.New(ctx, replicaReads{client, replica}, slog.Default(), q, "mail", cfg)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, b.Stop(context.Background())) })
	require.NoError(t, b.Resume(ctx, "mail"))
	i := takeJob(t, q)
	require.Equal(t, "lost", i.ID())
	require.NoError(t, i.Ack())
	s, err := b.State(ctx)
	require.NoError(t, err)
	require.Zero(t, s.Active)
	require.Zero(t, s.Reserved)
}

func TestRedisJobsRetireConsumer(t *testing.T) {
	cases := []struct {
		name    string
		pending bool
	}{
		{name: "remove consumer after acknowledgment"},
		{name: "keep consumer until pending delivery can be claimed", pending: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := privateRedis(t)
			t.Setenv("RR_REDIS_JOBS_ADDR", c.Options().Addr)
			cfg := redisjobs.Config{Queue: "mail", Prefetch: 1, VisibilityTimeout: 100 * time.Millisecond}
			d, q := jobsDriver(t, cfg, nil)
			ctx := t.Context()
			require.NoError(t, d.Push(ctx, &jobs.Job{Ident: "work", Job: "mail"}))
			require.NoError(t, d.Resume(ctx, "mail"))
			i := takeJob(t, q)
			if !tc.pending {
				require.NoError(t, i.Ack())
			}
			require.NoError(t, d.Stop(ctx))
			consumers, err := c.XInfoConsumers(ctx, "rr:jobs:{mail}:ready", "roadrunner").Result()
			require.NoError(t, err)
			if !tc.pending {
				require.Empty(t, consumers)
				return
			}
			require.Len(t, consumers, 1)
			require.EqualValues(t, 1, consumers[0].Pending)
			b, next := jobsDriver(t, cfg, nil)
			require.NoError(t, b.Resume(ctx, "mail"))
			recovered := takeJob(t, next)
			require.Equal(t, "work", recovered.ID())
			require.NoError(t, recovered.Ack())
		})
	}
}
