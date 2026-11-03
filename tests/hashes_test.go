package kv

import (
	"testing"

	redisV1 "github.com/roadrunner-server/api-go/v6/redis/v1"
)

func TestHashes(t *testing.T) {
	client := bootRedis(t)

	runCases(t, client, []rpcCase{
		{name: "hset counts new fields", setup: [][]any{{"hset", "h", "a", "0"}}, method: "redis.HSet", in: &redisV1.HSetRequest{Connection: connDefault, Key: "h", Fields: map[string][]byte{"a": b("1"), "b": b("2")}}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: 1}, after: []any{"hget", "h", "a"}, wantAfter: "1"},
		{name: "hset no fields", method: "redis.HSet", in: &redisV1.HSetRequest{Connection: connDefault, Key: "h"}, out: &redisV1.IntReply{}, wantErr: "should provide at least 1 key"},
		{name: "hset empty field name", method: "redis.HSet", in: &redisV1.HSetRequest{Connection: connDefault, Key: "h", Fields: map[string][]byte{" ": b("1")}}, out: &redisV1.IntReply{}, wantErr: "key can't be empty string"},
		{name: "hset wrong type", setup: [][]any{{"set", "h", "x"}}, method: "redis.HSet", in: &redisV1.HSetRequest{Connection: connDefault, Key: "h", Fields: map[string][]byte{"a": b("1")}}, out: &redisV1.IntReply{}, wantErr: "WRONGTYPE"},
		{name: "hsetnx new field", method: "redis.HSetNX", in: &redisV1.HSetNXRequest{Connection: connDefault, Key: "h", Field: "a", Value: b("1")}, out: &redisV1.BoolReply{}, want: &redisV1.BoolReply{Value: true}},
		{name: "hsetnx existing field keeps old", setup: [][]any{{"hset", "h", "a", "old"}}, method: "redis.HSetNX", in: &redisV1.HSetNXRequest{Connection: connDefault, Key: "h", Field: "a", Value: b("1")}, out: &redisV1.BoolReply{}, want: &redisV1.BoolReply{Value: false}, after: []any{"hget", "h", "a"}, wantAfter: "old"},

		{name: "hget", setup: [][]any{{"hset", "h", "a", "1"}}, method: "redis.HGet", in: &redisV1.HGetRequest{Connection: connDefault, Key: "h", Field: "a"}, out: &redisV1.BytesReply{}, want: &redisV1.BytesReply{Value: b("1")}},
		{name: "hget missing field is nil", setup: [][]any{{"hset", "h", "a", "1"}}, method: "redis.HGet", in: &redisV1.HGetRequest{Connection: connDefault, Key: "h", Field: "b"}, out: &redisV1.BytesReply{}, want: &redisV1.BytesReply{}},
		{name: "hget empty field", method: "redis.HGet", in: &redisV1.HGetRequest{Connection: connDefault, Key: "h", Field: ""}, out: &redisV1.BytesReply{}, wantErr: "key can't be empty string"},
		{name: "hmget keeps positions", setup: [][]any{{"hset", "h", "a", "1", "c", "3"}}, method: "redis.HMGet", in: &redisV1.HMGetRequest{Connection: connDefault, Key: "h", Fields: []string{"a", "b", "c"}}, out: &redisV1.BytesListReply{}, want: &redisV1.BytesListReply{Values: [][]byte{b("1"), nil, b("3")}, Found: []bool{true, false, true}}},
		{name: "hgetall", setup: [][]any{{"hset", "h", "a", "1", "b", "2"}}, method: "redis.HGetAll", in: &redisV1.HGetAllRequest{Connection: connDefault, Key: "h"}, out: &redisV1.MapReply{}, want: &redisV1.MapReply{Values: map[string][]byte{"a": b("1"), "b": b("2")}}},
		{name: "hgetall missing is empty", method: "redis.HGetAll", in: &redisV1.HGetAllRequest{Connection: connDefault, Key: "h"}, out: &redisV1.MapReply{}, want: &redisV1.MapReply{}},

		{name: "hdel counts removed", setup: [][]any{{"hset", "h", "a", "1", "b", "2"}}, method: "redis.HDel", in: &redisV1.HDelRequest{Connection: connDefault, Key: "h", Fields: []string{"a", "z"}}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: 1}},
		{name: "hexists", setup: [][]any{{"hset", "h", "a", "1"}}, method: "redis.HExists", in: &redisV1.HExistsRequest{Connection: connDefault, Key: "h", Field: "a"}, out: &redisV1.BoolReply{}, want: &redisV1.BoolReply{Value: true}},
		{name: "hexists missing", method: "redis.HExists", in: &redisV1.HExistsRequest{Connection: connDefault, Key: "h", Field: "a"}, out: &redisV1.BoolReply{}, want: &redisV1.BoolReply{Value: false}},
		{name: "hlen", setup: [][]any{{"hset", "h", "a", "1", "b", "2"}}, method: "redis.HLen", in: &redisV1.HLenRequest{Connection: connDefault, Key: "h"}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: 2}},
		{name: "hkeys", setup: [][]any{{"hset", "h", "a", "1", "b", "2"}}, method: "redis.HKeys", in: &redisV1.HKeysRequest{Connection: connDefault, Key: "h"}, out: &redisV1.BytesListReply{}, want: &redisV1.BytesListReply{Values: bs("a", "b")}, unordered: true},
		{name: "hvals", setup: [][]any{{"hset", "h", "a", "1", "b", "2"}}, method: "redis.HVals", in: &redisV1.HValsRequest{Connection: connDefault, Key: "h"}, out: &redisV1.BytesListReply{}, want: &redisV1.BytesListReply{Values: bs("1", "2")}, unordered: true},

		{name: "hincrby missing field", method: "redis.HIncrBy", in: &redisV1.HIncrByRequest{Connection: connDefault, Key: "h", Field: "n", Increment: 5}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: 5}},
		{name: "hincrby existing", setup: [][]any{{"hset", "h", "n", "10"}}, method: "redis.HIncrBy", in: &redisV1.HIncrByRequest{Connection: connDefault, Key: "h", Field: "n", Increment: -3}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: 7}},
		{name: "hincrby non integer", setup: [][]any{{"hset", "h", "n", "x"}}, method: "redis.HIncrBy", in: &redisV1.HIncrByRequest{Connection: connDefault, Key: "h", Field: "n", Increment: 1}, out: &redisV1.IntReply{}, wantErr: "not an integer"},
		{name: "hincrbyfloat", setup: [][]any{{"hset", "h", "n", "10.5"}}, method: "redis.HIncrByFloat", in: &redisV1.HIncrByFloatRequest{Connection: connDefault, Key: "h", Field: "n", Increment: 0.25}, out: &redisV1.FloatReply{}, want: &redisV1.FloatReply{Value: 10.75}},
	})
}
