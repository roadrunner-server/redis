package kv

import (
	"context"
	"crypto/rand"
	"log/slog"
	"testing"

	mocklogger "tests/mock"

	api "github.com/roadrunner-server/api-plugins/v6/jobs"
	"github.com/roadrunner-server/config/v6"
	"github.com/roadrunner-server/jobs/v6"
	priorityqueue "github.com/roadrunner-server/priority_queue"
	"github.com/roadrunner-server/redis/v6"
	"github.com/stretchr/testify/require"
)

func TestRedisJobsPipelineDeclarations(t *testing.T) {
	queueName := rand.Text()
	t.Setenv("RR_TEST_QUEUE", queueName)
	cfg := &config.Plugin{Path: "configs/.rr-redis-jobs.yaml"}
	require.NoError(t, cfg.Init())
	p := &redis.Plugin{}
	require.NoError(t, p.Init(cfg, mocklogger.NewLogger(slog.Default())))
	c, ok := any(p).(api.Constructor)
	require.True(t, ok, "Jobs must discover the Redis driver")
	ctx := t.Context()
	q := priorityqueue.NewBinHeap[api.Job](1)
	producer, err := c.DriverFromConfig(ctx, "jobs.pipelines.mail.config", q, jobs.Pipeline{"name": "mail", "driver": "redis"})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, producer.Stop(context.Background())) })
	state, err := producer.State(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 5, state.Priority)
	require.False(t, state.Ready)
	require.NoError(t, producer.Push(ctx, &jobs.Job{Ident: "from-yaml", Job: "send", Options: &jobs.Options{Pipeline: "mail"}}))

	cache, err := p.KvFromConfig(ctx, "kv.cache.config")
	require.NoError(t, err)
	t.Cleanup(func() { cache.Stop(context.Background()) })
	require.NoError(t, cache.Clear(ctx))
	pipe := jobs.Pipeline{"name": "from-rpc", "driver": "redis", "queue": queueName, "prefetch": "1", "visibility_timeout": "2s"}
	reader, err := c.DriverFromPipeline(ctx, pipe, q)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, reader.Stop(context.Background())) })
	require.NoError(t, reader.Resume(ctx, pipe.Name()))
	i := takeJob(t, q)
	require.Equal(t, "from-yaml", i.ID())
	require.Equal(t, "from-rpc", i.GroupID())
	require.EqualValues(t, 5, i.Priority())
	require.NoError(t, i.Ack())
}
