package kv

import (
	"testing"

	kvProto "github.com/roadrunner-server/api-go/v6/kv/v1"
	"github.com/stretchr/testify/require"
)

// TestTLSConnection drives the same round trip over the TLS port, so the
// certificate wiring in the driver config is exercised end to end rather than
// only parsed.
func TestTLSConnection(t *testing.T) {
	client := bootKV(t, "configs/.rr-redis-tls.yaml", rpcTLSAddr)

	require.NoError(t, client.Call("kv.Set", items(map[string]string{"a": "aa", "b": "bb"}), &kvProto.Response{}))

	require.Equal(t, 2, has(t, client, "a", "b"))

	resp := &kvProto.Response{}
	require.NoError(t, client.Call("kv.MGet", keys("a"), resp))
	require.Len(t, resp.GetItems(), 1)
	require.Equal(t, "aa", string(resp.GetItems()[0].GetValue()))
}
