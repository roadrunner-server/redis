package kv

import (
	"net/rpc"
	"testing"
	"time"

	"tests/helpers"

	kvProto "github.com/roadrunner-server/api-go/v6/kv/v1"
	"github.com/roadrunner-server/kv/v6"
	"github.com/roadrunner-server/redis/v6"
	rpcPlugin "github.com/roadrunner-server/rpc/v6"
	"github.com/stretchr/testify/require"
)

const (
	rpcAddr    = "127.0.0.1:6001"
	rpcTLSAddr = "127.0.0.1:6002"
	storage    = "redis-rr"

	// shortTTL is the lifetime given to keys expected to expire during a test.
	shortTTL   = time.Second * 2
	expiryWait = time.Second * 30
	expiryTick = time.Millisecond * 250
)

func redisPlugins() []any {
	return []any{&kv.Plugin{}, &redis.Plugin{}, &rpcPlugin.Plugin{}}
}

// bootKV starts the container against cfgPath and hands back a connected rpc
// client with the storage emptied, so each test begins from a known state.
func bootKV(t *testing.T, cfgPath, addr string) *rpc.Client {
	t.Helper()

	helpers.Start(t, cfgPath, redisPlugins(), helpers.WithTCPProbe(addr))

	client := helpers.NewRPCClient(t, addr)
	require.NoError(t, client.Call("kv.Clear", &kvProto.Request{Storage: storage}, &kvProto.Response{}))

	return client
}

func items(pairs map[string]string) *kvProto.Request {
	req := &kvProto.Request{Storage: storage}
	for k, v := range pairs {
		req.Items = append(req.Items, &kvProto.Item{Key: k, Value: []byte(v)})
	}
	return req
}

func keys(names ...string) *kvProto.Request {
	req := &kvProto.Request{Storage: storage}
	for _, n := range names {
		req.Items = append(req.Items, &kvProto.Item{Key: n})
	}
	return req
}

// has returns how many of the given keys the storage currently holds.
func has(t *testing.T, client *rpc.Client, names ...string) int {
	t.Helper()

	resp := &kvProto.Response{}
	require.NoError(t, client.Call("kv.Has", keys(names...), resp))

	return len(resp.GetItems())
}

func TestSetAndHas(t *testing.T) {
	client := bootKV(t, "configs/.rr-redis.yaml", rpcAddr)

	require.NoError(t, client.Call("kv.Set", items(map[string]string{"a": "aa", "b": "bb"}), &kvProto.Response{}))

	require.Equal(t, 2, has(t, client, "a", "b"))
	require.Equal(t, 0, has(t, client, "missing"))
}

func TestMGetReturnsStoredValues(t *testing.T) {
	client := bootKV(t, "configs/.rr-redis.yaml", rpcAddr)

	require.NoError(t, client.Call("kv.Set", items(map[string]string{"a": "aa", "b": "bb"}), &kvProto.Response{}))

	resp := &kvProto.Response{}
	require.NoError(t, client.Call("kv.MGet", keys("a", "b", "absent"), resp))

	got := make(map[string]string, len(resp.GetItems()))
	for _, it := range resp.GetItems() {
		got[it.GetKey()] = string(it.GetValue())
	}

	require.Equal(t, map[string]string{"a": "aa", "b": "bb"}, got)
}

func TestDeleteRemovesOnlyTheNamedKey(t *testing.T) {
	client := bootKV(t, "configs/.rr-redis.yaml", rpcAddr)

	require.NoError(t, client.Call("kv.Set", items(map[string]string{"a": "aa", "b": "bb"}), &kvProto.Response{}))
	require.NoError(t, client.Call("kv.Delete", keys("a"), &kvProto.Response{}))

	require.Equal(t, 0, has(t, client, "a"))
	require.Equal(t, 1, has(t, client, "b"))
}

func TestClearEmptiesTheStorage(t *testing.T) {
	client := bootKV(t, "configs/.rr-redis.yaml", rpcAddr)

	require.NoError(t, client.Call("kv.Set", items(map[string]string{"a": "aa", "b": "bb"}), &kvProto.Response{}))
	require.NoError(t, client.Call("kv.Clear", &kvProto.Request{Storage: storage}, &kvProto.Response{}))

	require.Equal(t, 0, has(t, client, "a", "b"))
}

// TestTTLReportsRemainingLifetime covers the call memcached cannot serve: redis
// exposes a key's remaining lifetime, so kv.TTL must answer for a key that has
// one and stay silent about a key that does not.
func TestTTLReportsRemainingLifetime(t *testing.T) {
	client := bootKV(t, "configs/.rr-redis.yaml", rpcAddr)

	req := &kvProto.Request{
		Storage: storage,
		Items: []*kvProto.Item{
			{Key: "permanent", Value: []byte("v")},
			{Key: "ephemeral", Value: []byte("v"), Timeout: time.Now().UTC().Add(time.Minute).Format(time.RFC3339)},
		},
	}
	require.NoError(t, client.Call("kv.Set", req, &kvProto.Response{}))

	resp := &kvProto.Response{}
	require.NoError(t, client.Call("kv.TTL", keys("permanent", "ephemeral"), resp))

	// the driver skips keys with no expiry, so only the ephemeral one comes back
	require.Len(t, resp.GetItems(), 1)
	require.Equal(t, "ephemeral", resp.GetItems()[0].GetKey())
	expiry, err := time.Parse(time.RFC3339, resp.GetItems()[0].GetTimeout())
	require.NoError(t, err)
	require.True(t, expiry.After(time.Now()), "a key with a TTL must report a future expiry")
}

// TestKeyExpiresAfterTTL polls for the expiry rather than sleeping out the
// worst case, and checks the key without a TTL survives it.
func TestKeyExpiresAfterTTL(t *testing.T) {
	client := bootKV(t, "configs/.rr-redis.yaml", rpcAddr)

	req := &kvProto.Request{
		Storage: storage,
		Items: []*kvProto.Item{
			{Key: "permanent", Value: []byte("v")},
			{Key: "ephemeral", Value: []byte("v"), Timeout: time.Now().UTC().Add(shortTTL).Format(time.RFC3339)},
		},
	}
	require.NoError(t, client.Call("kv.Set", req, &kvProto.Response{}))
	require.Equal(t, 2, has(t, client, "permanent", "ephemeral"))

	require.Eventually(t, func() bool {
		return has(t, client, "ephemeral") == 0
	}, expiryWait, expiryTick, "the key with a TTL never expired")

	require.Equal(t, 1, has(t, client, "permanent"), "the key without a TTL must survive")
}

func TestMExpireAppliesTTLToExistingKeys(t *testing.T) {
	client := bootKV(t, "configs/.rr-redis.yaml", rpcAddr)

	require.NoError(t, client.Call("kv.Set", items(map[string]string{"a": "aa", "b": "bb"}), &kvProto.Response{}))

	expire := &kvProto.Request{
		Storage: storage,
		Items: []*kvProto.Item{
			{Key: "a", Timeout: time.Now().UTC().Add(shortTTL).Format(time.RFC3339)},
			{Key: "b", Timeout: time.Now().UTC().Add(shortTTL).Format(time.RFC3339)},
		},
	}
	require.NoError(t, client.Call("kv.MExpire", expire, &kvProto.Response{}))

	require.Eventually(t, func() bool {
		return has(t, client, "a", "b") == 0
	}, expiryWait, expiryTick, "keys did not expire after MExpire")
}

func TestUnknownStorageIsRejected(t *testing.T) {
	client := bootKV(t, "configs/.rr-redis.yaml", rpcAddr)

	err := client.Call("kv.Has", &kvProto.Request{
		Storage: "not-configured",
		Items:   []*kvProto.Item{{Key: "a"}},
	}, &kvProto.Response{})

	require.Error(t, err)
}
