package kv

import (
	"bytes"
	"net/rpc"
	"sort"
	"testing"

	"tests/helpers"

	goredis "github.com/redis/go-redis/v9"
	redisV1 "github.com/roadrunner-server/api-go/v6/redis/v1"
	"github.com/roadrunner-server/redis/v6"
	rpcPlugin "github.com/roadrunner-server/rpc/v6"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

const (
	connDefault = "default"
	connSecond  = "second"
	redisAddr   = "127.0.0.1:16379"
)

// rdb talks to the test server directly for setup and verification, so the
// expectations do not depend on the RPC under test.
func rdb(t *testing.T) *goredis.Client {
	t.Helper()

	c := goredis.NewClient(&goredis.Options{Addr: redisAddr})
	t.Cleanup(func() { _ = c.Close() })

	return c
}

func bootRedis(t *testing.T) *rpc.Client {
	t.Helper()

	helpers.Start(t, "configs/.rr-redis-rpc.yaml", []any{&redis.Plugin{}, &rpcPlugin.Plugin{}}, helpers.WithTCPProbe(rpcAddr))

	return helpers.NewRPCClient(t, rpcAddr)
}

type rpcCase struct {
	name string
	// setup runs raw commands on db 0 before the call.
	setup  [][]any
	method string
	in     proto.Message
	out    proto.Message
	want   proto.Message
	// unordered sorts BytesListReply.values before the comparison.
	unordered bool
	wantErr   string
	// after is a raw command run on db 0 after the call; its result must equal wantAfter.
	after     []any
	wantAfter any
}

func runCases(t *testing.T, client *rpc.Client, cases []rpcCase) {
	t.Helper()

	r := rdb(t)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, r.FlushAll(t.Context()).Err())
			for _, s := range tc.setup {
				require.NoError(t, r.Do(t.Context(), s...).Err())
			}

			err := client.Call(tc.method, tc.in, tc.out)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)

			if tc.unordered {
				sortValues(tc.out)
				sortValues(tc.want)
			}
			require.True(t, proto.Equal(tc.want, tc.out), "want %v, got %v", tc.want, tc.out)

			if tc.after != nil {
				got, err := r.Do(t.Context(), tc.after...).Result()
				require.NoError(t, err)
				require.Equal(t, tc.wantAfter, got)
			}
		})
	}
}

func sortValues(m proto.Message) {
	l, ok := m.(*redisV1.BytesListReply)
	if !ok {
		return
	}
	sort.Slice(l.Values, func(i, j int) bool { return bytes.Compare(l.Values[i], l.Values[j]) < 0 })
}

func b(s string) []byte { return []byte(s) }

func bs(s ...string) [][]byte {
	out := make([][]byte, 0, len(s))
	for _, v := range s {
		out = append(out, []byte(v))
	}
	return out
}

func TestConnectionRouting(t *testing.T) {
	client := bootRedis(t)
	require.NoError(t, rdb(t).FlushAll(t.Context()).Err())

	require.NoError(t, client.Call("redis.Set", &redisV1.SetRequest{Connection: connDefault, Key: "k", Value: b("v")}, &redisV1.BoolReply{}))

	out := &redisV1.BytesReply{}
	require.NoError(t, client.Call("redis.Get", &redisV1.GetRequest{Connection: connSecond, Key: "k"}, out))
	require.Nil(t, out.Value, "db 1 must not see a key written to db 0")

	require.ErrorContains(t, client.Call("redis.Get", &redisV1.GetRequest{Connection: "nope", Key: "k"}, out), "no such connection: nope")
	require.ErrorContains(t, client.Call("redis.Get", &redisV1.GetRequest{Key: "k"}, out), "no connection provided")
}
