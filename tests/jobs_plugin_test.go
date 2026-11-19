package kv

import (
	"context"
	"crypto/rand"
	"log/slog"
	"testing"

	"tests/helpers"
	mocklogger "tests/mock"

	pb "github.com/roadrunner-server/api-go/v6/jobs/v1"
	api "github.com/roadrunner-server/api-plugins/v6/jobs"
	"github.com/roadrunner-server/config/v6"
	"github.com/roadrunner-server/jobs/v6"
	priorityqueue "github.com/roadrunner-server/priority_queue"
	"github.com/roadrunner-server/redis/v6"
	"github.com/roadrunner-server/rpc/v6"
	"github.com/roadrunner-server/server/v6"
	"github.com/stretchr/testify/require"
)

func TestRedisJobsPipelineDeclarations(t *testing.T) {
	t.Setenv("RR_TEST_PRIVATE_REDIS", privateRedis(t).Options().Addr)
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

func TestRedisJobsDeclareRPC(t *testing.T) {
	t.Setenv("RR_TEST_PRIVATE_REDIS", privateRedis(t).Options().Addr)
	t.Setenv("RR_TEST_QUEUE", rand.Text())
	const address = "127.0.0.1:16092"
	helpers.Start(t, "configs/.rr-redis-jobs.yaml", []any{&server.Plugin{}, &jobs.Plugin{}, &redis.Plugin{}, &rpc.Plugin{}}, helpers.WithTCPProbe(address))
	client := helpers.NewRPCClient(t, address)
	name := rand.Text()
	req := &pb.DeclareRequest{Pipeline: map[string]string{
		"name": "dynamic", "driver": "redis", "queue": name, "priority": "3", "prefetch": "1", "visibility_timeout": "2s",
	}}
	require.NoError(t, client.Call("jobs.Declare", req, &pb.Empty{}))
	require.NoError(t, client.Call("jobs.Push", &pb.PushRequest{Job: &pb.Job{
		Id: "from-rpc", Job: "send", Payload: []byte{0xff}, Options: &pb.Options{Pipeline: "dynamic"},
	}}, &pb.Empty{}))
	stats := &pb.Stats{}
	require.NoError(t, client.Call("jobs.Stat", &pb.Empty{}, stats))
	for _, s := range stats.GetStats() {
		if s.GetPipeline() == "dynamic" {
			require.Equal(t, name, s.GetQueue())
			require.EqualValues(t, 3, s.GetPriority())
			require.EqualValues(t, 1, s.GetReserved())
			require.False(t, s.GetReady())
			return
		}
	}
	t.Fatal("jobs.Stat omitted the declared pipeline")
}

func TestRedisJobsRejectInvalidDeclarations(t *testing.T) {
	cases := []struct {
		name string
		yaml string
		pipe jobs.Pipeline
		want string
	}{
		{name: "missing server", yaml: "version: '3'", want: "missing redis connection"},
		{name: "invalid RPC visibility", yaml: "redis: {db: 0}", pipe: jobs.Pipeline{"visibility_timeout": "tomorrow"}, want: "visibility_timeout"},
		{name: "invalid YAML visibility", yaml: "redis: {db: 0}\npipeline: {visibility_timeout: tomorrow}", want: "visibility_timeout"},
		{name: "invalid RPC prefetch", yaml: "redis: {db: 0}", pipe: jobs.Pipeline{"prefetch": "many"}, want: "prefetch"},
		{name: "negative database", yaml: "redis: {db: -1}", want: "nonnegative"},
		{name: "nonzero cluster database", yaml: "redis: {db: 1, addrs: ['node-1:6379', 'node-2:6379']}", want: "database 0"},
		{name: "read-only server", yaml: "redis: {read_only: true}", want: "writable connection"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Plugin{Type: "yaml", ReadInCfg: []byte(tc.yaml)}
			require.NoError(t, cfg.Init())
			p := &redis.Plugin{}
			require.NoError(t, p.Init(cfg, mocklogger.NewLogger(slog.Default())))
			q := priorityqueue.NewBinHeap[api.Job](1)
			var err error
			if tc.pipe == nil {
				_, err = p.DriverFromConfig(t.Context(), "pipeline", q, jobs.Pipeline{"name": "mail"})
			} else {
				tc.pipe.With("name", "mail")
				_, err = p.DriverFromPipeline(t.Context(), tc.pipe, q)
			}
			require.ErrorContains(t, err, tc.want)
		})
	}
}
