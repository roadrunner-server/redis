package kv

import (
	"testing"

	redisV1 "github.com/roadrunner-server/api-go/v6/redis/v1"
	"google.golang.org/protobuf/proto"
)

func TestStrings(t *testing.T) {
	client := bootRedis(t)

	runCases(t, client, []rpcCase{
		{name: "get missing is nil", method: "redis.Get", in: &redisV1.GetRequest{Connection: connDefault, Key: "k"}, out: &redisV1.BytesReply{}, want: &redisV1.BytesReply{}},
		{name: "get existing", setup: [][]any{{"set", "k", "v"}}, method: "redis.Get", in: &redisV1.GetRequest{Connection: connDefault, Key: "k"}, out: &redisV1.BytesReply{}, want: &redisV1.BytesReply{Value: b("v")}},
		{name: "get empty string is present", setup: [][]any{{"set", "k", ""}}, method: "redis.Get", in: &redisV1.GetRequest{Connection: connDefault, Key: "k"}, out: &redisV1.BytesReply{}, want: &redisV1.BytesReply{Value: []byte{}}},
		{name: "get empty key", method: "redis.Get", in: &redisV1.GetRequest{Connection: connDefault, Key: " "}, out: &redisV1.BytesReply{}, wantErr: "key can't be empty string"},
		{name: "get wrong type", setup: [][]any{{"rpush", "k", "a"}}, method: "redis.Get", in: &redisV1.GetRequest{Connection: connDefault, Key: "k"}, out: &redisV1.BytesReply{}, wantErr: "WRONGTYPE"},

		{name: "set plain", method: "redis.Set", in: &redisV1.SetRequest{Connection: connDefault, Key: "k", Value: b("v")}, out: &redisV1.BoolReply{}, want: &redisV1.BoolReply{Value: true}, after: []any{"get", "k"}, wantAfter: "v"},
		{name: "set ex sets ttl", method: "redis.Set", in: &redisV1.SetRequest{Connection: connDefault, Key: "k", Value: b("v"), Ex: proto.Int64(100)}, out: &redisV1.BoolReply{}, want: &redisV1.BoolReply{Value: true}, after: []any{"ttl", "k"}, wantAfter: int64(100)},
		{name: "set px sets ttl", method: "redis.Set", in: &redisV1.SetRequest{Connection: connDefault, Key: "k", Value: b("v"), Px: proto.Int64(100250)}, out: &redisV1.BoolReply{}, want: &redisV1.BoolReply{Value: true}, after: []any{"ttl", "k"}, wantAfter: int64(100)},
		{name: "set nx on missing", method: "redis.Set", in: &redisV1.SetRequest{Connection: connDefault, Key: "k", Value: b("v"), Nx: true}, out: &redisV1.BoolReply{}, want: &redisV1.BoolReply{Value: true}},
		{name: "set nx on existing keeps old", setup: [][]any{{"set", "k", "old"}}, method: "redis.Set", in: &redisV1.SetRequest{Connection: connDefault, Key: "k", Value: b("v"), Nx: true}, out: &redisV1.BoolReply{}, want: &redisV1.BoolReply{Value: false}, after: []any{"get", "k"}, wantAfter: "old"},
		{name: "set xx on missing", method: "redis.Set", in: &redisV1.SetRequest{Connection: connDefault, Key: "k", Value: b("v"), Xx: true}, out: &redisV1.BoolReply{}, want: &redisV1.BoolReply{Value: false}},
		{name: "set xx on existing", setup: [][]any{{"set", "k", "old"}}, method: "redis.Set", in: &redisV1.SetRequest{Connection: connDefault, Key: "k", Value: b("v"), Xx: true}, out: &redisV1.BoolReply{}, want: &redisV1.BoolReply{Value: true}, after: []any{"get", "k"}, wantAfter: "v"},
		{name: "set keepttl keeps ttl", setup: [][]any{{"set", "k", "old", "ex", "100"}}, method: "redis.Set", in: &redisV1.SetRequest{Connection: connDefault, Key: "k", Value: b("v"), KeepTtl: true}, out: &redisV1.BoolReply{}, want: &redisV1.BoolReply{Value: true}, after: []any{"ttl", "k"}, wantAfter: int64(100)},
		{name: "set without keepttl drops ttl", setup: [][]any{{"set", "k", "old", "ex", "100"}}, method: "redis.Set", in: &redisV1.SetRequest{Connection: connDefault, Key: "k", Value: b("v")}, out: &redisV1.BoolReply{}, want: &redisV1.BoolReply{Value: true}, after: []any{"ttl", "k"}, wantAfter: int64(-1)},
		{name: "set nx and xx together", method: "redis.Set", in: &redisV1.SetRequest{Connection: connDefault, Key: "k", Value: b("v"), Nx: true, Xx: true}, out: &redisV1.BoolReply{}, wantErr: "nx and xx are mutually exclusive"},
		{name: "set ex and px together", method: "redis.Set", in: &redisV1.SetRequest{Connection: connDefault, Key: "k", Value: b("v"), Ex: proto.Int64(1), Px: proto.Int64(1000)}, out: &redisV1.BoolReply{}, wantErr: "ex and px are mutually exclusive"},
		{name: "set ex zero", method: "redis.Set", in: &redisV1.SetRequest{Connection: connDefault, Key: "k", Value: b("v"), Ex: proto.Int64(0)}, out: &redisV1.BoolReply{}, wantErr: "invalid expire time"},
		{name: "set px negative", method: "redis.Set", in: &redisV1.SetRequest{Connection: connDefault, Key: "k", Value: b("v"), Px: proto.Int64(-5)}, out: &redisV1.BoolReply{}, wantErr: "invalid expire time"},

		{name: "mget keeps positions", setup: [][]any{{"mset", "a", "1", "c", "3"}}, method: "redis.MGet", in: &redisV1.MGetRequest{Connection: connDefault, Keys: []string{"a", "b", "c"}}, out: &redisV1.BytesListReply{}, want: &redisV1.BytesListReply{Values: [][]byte{b("1"), nil, b("3")}, Found: []bool{true, false, true}}},
		{name: "mget no keys", method: "redis.MGet", in: &redisV1.MGetRequest{Connection: connDefault}, out: &redisV1.BytesListReply{}, wantErr: "should provide at least 1 key"},
		{name: "mset", method: "redis.MSet", in: &redisV1.MSetRequest{Connection: connDefault, Values: map[string][]byte{"a": b("1"), "b": b("2")}}, out: &redisV1.Empty{}, want: &redisV1.Empty{}, after: []any{"mget", "a", "b"}, wantAfter: []any{"1", "2"}},
		{name: "mset no values", method: "redis.MSet", in: &redisV1.MSetRequest{Connection: connDefault}, out: &redisV1.Empty{}, wantErr: "should provide at least 1 key"},

		{name: "getdel returns and removes", setup: [][]any{{"set", "k", "v"}}, method: "redis.GetDel", in: &redisV1.GetDelRequest{Connection: connDefault, Key: "k"}, out: &redisV1.BytesReply{}, want: &redisV1.BytesReply{Value: b("v")}, after: []any{"exists", "k"}, wantAfter: int64(0)},
		{name: "getdel missing is nil", method: "redis.GetDel", in: &redisV1.GetDelRequest{Connection: connDefault, Key: "k"}, out: &redisV1.BytesReply{}, want: &redisV1.BytesReply{}},

		{name: "append to missing", method: "redis.Append", in: &redisV1.AppendRequest{Connection: connDefault, Key: "k", Value: b("ab")}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: 2}},
		{name: "append to existing", setup: [][]any{{"set", "k", "ab"}}, method: "redis.Append", in: &redisV1.AppendRequest{Connection: connDefault, Key: "k", Value: b("cd")}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: 4}, after: []any{"get", "k"}, wantAfter: "abcd"},
		{name: "strlen", setup: [][]any{{"set", "k", "abcd"}}, method: "redis.StrLen", in: &redisV1.StrLenRequest{Connection: connDefault, Key: "k"}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: 4}},
		{name: "strlen missing", method: "redis.StrLen", in: &redisV1.StrLenRequest{Connection: connDefault, Key: "k"}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: 0}},

		{name: "incr missing starts at 1", method: "redis.Incr", in: &redisV1.IncrRequest{Connection: connDefault, Key: "n"}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: 1}},
		{name: "incr existing", setup: [][]any{{"set", "n", "10"}}, method: "redis.Incr", in: &redisV1.IncrRequest{Connection: connDefault, Key: "n"}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: 11}},
		{name: "incr non integer", setup: [][]any{{"set", "n", "abc"}}, method: "redis.Incr", in: &redisV1.IncrRequest{Connection: connDefault, Key: "n"}, out: &redisV1.IntReply{}, wantErr: "not an integer"},
		{name: "incrby missing returns increment", method: "redis.IncrBy", in: &redisV1.IncrByRequest{Connection: connDefault, Key: "n", Increment: 5}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: 5}},
		{name: "incrby negative", setup: [][]any{{"set", "n", "10"}}, method: "redis.IncrBy", in: &redisV1.IncrByRequest{Connection: connDefault, Key: "n", Increment: -3}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: 7}},
		{name: "incrby overflow", setup: [][]any{{"set", "n", "9223372036854775807"}}, method: "redis.IncrBy", in: &redisV1.IncrByRequest{Connection: connDefault, Key: "n", Increment: 1}, out: &redisV1.IntReply{}, wantErr: "overflow"},
		{name: "incrbyfloat", setup: [][]any{{"set", "n", "10.5"}}, method: "redis.IncrByFloat", in: &redisV1.IncrByFloatRequest{Connection: connDefault, Key: "n", Increment: 0.25}, out: &redisV1.FloatReply{}, want: &redisV1.FloatReply{Value: 10.75}},
		{name: "decr missing is -1", method: "redis.Decr", in: &redisV1.DecrRequest{Connection: connDefault, Key: "n"}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: -1}},
		{name: "decrby", setup: [][]any{{"set", "n", "10"}}, method: "redis.DecrBy", in: &redisV1.DecrByRequest{Connection: connDefault, Key: "n", Decrement: 4}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: 6}},
	})
}
