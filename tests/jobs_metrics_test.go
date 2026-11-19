package kv

import (
	"log/slog"
	"testing"
	"time"

	mocklogger "tests/mock"

	api "github.com/roadrunner-server/api-plugins/v6/jobs"
	"github.com/roadrunner-server/config/v6"
	"github.com/roadrunner-server/jobs/v6"
	priorityqueue "github.com/roadrunner-server/priority_queue"
	"github.com/roadrunner-server/redis/v6"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestRedisJobsReleasePoolMetrics(t *testing.T) {
	cases := []struct {
		name    string
		queue   string
		wantErr string
	}{
		{name: "stopped pipeline", queue: "mail"},
		{name: "failed declaration", queue: "mail{west}", wantErr: "braces"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			addr := privateRedis(t).Options().Addr
			t.Setenv("RR_TEST_PRIVATE_REDIS", addr)
			t.Setenv("RR_TEST_QUEUE", "mail")
			reader := metric.NewManualReader()
			provider := metric.NewMeterProvider(metric.WithReader(reader))
			original := otel.GetMeterProvider()
			otel.SetMeterProvider(provider)
			defer otel.SetMeterProvider(original)
			defer func() { require.NoError(t, provider.Shutdown(t.Context())) }()
			cfg := &config.Plugin{Path: "configs/.rr-redis-jobs.yaml"}
			require.NoError(t, cfg.Init())
			p := &redis.Plugin{}
			require.NoError(t, p.Init(cfg, mocklogger.NewLogger(slog.Default())))
			reportsPool := func() bool {
				var data metricdata.ResourceMetrics
				require.NoError(t, reader.Collect(t.Context(), &data))
				for _, scope := range data.ScopeMetrics {
					for _, m := range scope.Metrics {
						if m.Name != "db.client.connections.usage" {
							continue
						}
						for _, point := range m.Data.(metricdata.Sum[int64]).DataPoints {
							if pool, ok := point.Attributes.Value("pool.name"); ok && pool.AsString() == addr {
								return true
							}
						}
					}
				}
				return false
			}
			d, err := p.DriverFromPipeline(t.Context(), jobs.Pipeline{"name": "mail", "queue": tc.queue}, priorityqueue.NewBinHeap[api.Job](1))
			if tc.wantErr == "" {
				require.NoError(t, err)
				require.True(t, reportsPool())
				require.NoError(t, d.Stop(t.Context()))
			} else {
				require.ErrorContains(t, err, tc.wantErr)
			}
			require.Eventually(t, func() bool { return !reportsPool() }, time.Second, 5*time.Millisecond)
		})
	}
}
