package kv

import (
	"io"
	"net/http"
	"testing"

	"tests/helpers"

	kvProto "github.com/roadrunner-server/api-go/v6/kv/v2"
	"github.com/roadrunner-server/kv/v6"
	"github.com/roadrunner-server/metrics/v6"
	"github.com/roadrunner-server/redis/v6"
	rpcPlugin "github.com/roadrunner-server/rpc/v6"
	"github.com/stretchr/testify/require"
)

// TestGlobalSectionSuppliesAddrs covers the fallback lookup: the kv entry
// carries no config block, so the driver has to pick the top-level redis-rr
// section up instead.
func TestGlobalSectionSuppliesAddrs(t *testing.T) {
	client := bootKV(t, "configs/.rr-redis-global.yaml", rpcAddr)

	require.NoError(t, client.Call("kv.Set", items(map[string]string{"a": "aa"}), &kvProto.KvResponse{}))
	require.Equal(t, 1, has(t, client, "a"))
}

// TestMissingConfigFailsToServe pins the error path: with neither a per-storage
// config block nor a global section, the driver cannot be built and Serve must
// report it rather than starting a storage pointed nowhere.
func TestMissingConfigFailsToServe(t *testing.T) {
	err := helpers.StartExpectServeError(t, "configs/.rr-redis-no-config.yaml", redisPlugins())

	require.Error(t, err)
}

// TestMetricsAreExported checks the driver registers its pool gauges with the
// metrics plugin and that they reach the exporter.
func TestMetricsAreExported(t *testing.T) {
	helpers.Start(t,
		"configs/.rr-redis-metrics.yaml",
		[]any{&kv.Plugin{}, &redis.Plugin{}, &rpcPlugin.Plugin{}, &metrics.Plugin{}},
		helpers.WithTCPProbe(rpcAddr),
	)

	// touch the storage so the pool has stats worth reporting
	client := helpers.NewRPCClient(t, rpcAddr)
	require.NoError(t, client.Call("kv.Set", items(map[string]string{"a": "aa"}), &kvProto.KvResponse{}))

	body := scrape(t, "http://127.0.0.1:2112/metrics")

	for _, want := range []string{
		"rr_redis_pool_conn_idle_current",
		"rr_redis_pool_conn_stale_total",
		"rr_redis_pool_conn_total_current",
		"rr_redis_pool_hit_total",
	} {
		require.Contains(t, body, want)
	}
}

// scrape fetches the metrics endpoint and returns the body.
func scrape(t *testing.T, url string) string {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer func() { require.NoError(t, resp.Body.Close()) }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return string(body)
}
