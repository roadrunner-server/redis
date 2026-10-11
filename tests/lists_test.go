package kv

import (
	"testing"

	redisV1 "github.com/roadrunner-server/api-go/v6/redis/v1"
)

func TestLists(t *testing.T) {
	client := bootRedis(t)

	runCases(t, client, []rpcCase{
		{name: "lpush prepends in order", setup: [][]any{{"rpush", "l", "c"}}, method: "redis.LPush", in: &redisV1.LPushRequest{Key: "l", Elements: bs("b", "a")}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: 3}, after: []any{"lrange", "l", "0", "-1"}, wantAfter: []any{"a", "b", "c"}},
		{name: "lpush no elements", method: "redis.LPush", in: &redisV1.LPushRequest{Key: "l"}, out: &redisV1.IntReply{}, wantErr: "should provide at least 1 key"},
		{name: "rpush appends", setup: [][]any{{"rpush", "l", "a"}}, method: "redis.RPush", in: &redisV1.RPushRequest{Key: "l", Elements: bs("b", "c")}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: 3}, after: []any{"lrange", "l", "0", "-1"}, wantAfter: []any{"a", "b", "c"}},
		{name: "rpush wrong type", setup: [][]any{{"set", "l", "x"}}, method: "redis.RPush", in: &redisV1.RPushRequest{Key: "l", Elements: bs("a")}, out: &redisV1.IntReply{}, wantErr: "WRONGTYPE"},

		{name: "lpop count 0 pops one", setup: [][]any{{"rpush", "l", "a", "b", "c"}}, method: "redis.LPop", in: &redisV1.LPopRequest{Key: "l"}, out: &redisV1.BytesListReply{}, want: &redisV1.BytesListReply{Values: bs("a")}, after: []any{"llen", "l"}, wantAfter: int64(2)},
		{name: "lpop count 2", setup: [][]any{{"rpush", "l", "a", "b", "c"}}, method: "redis.LPop", in: &redisV1.LPopRequest{Key: "l", Count: 2}, out: &redisV1.BytesListReply{}, want: &redisV1.BytesListReply{Values: bs("a", "b")}},
		{name: "lpop missing is empty", method: "redis.LPop", in: &redisV1.LPopRequest{Key: "l"}, out: &redisV1.BytesListReply{}, want: &redisV1.BytesListReply{}},
		{name: "rpop count 2", setup: [][]any{{"rpush", "l", "a", "b", "c"}}, method: "redis.RPop", in: &redisV1.RPopRequest{Key: "l", Count: 2}, out: &redisV1.BytesListReply{}, want: &redisV1.BytesListReply{Values: bs("c", "b")}},

		{name: "lrange", setup: [][]any{{"rpush", "l", "a", "b", "c"}}, method: "redis.LRange", in: &redisV1.LRangeRequest{Key: "l", Start: 0, Stop: -1}, out: &redisV1.BytesListReply{}, want: &redisV1.BytesListReply{Values: bs("a", "b", "c")}},
		{name: "lrange out of range is empty", setup: [][]any{{"rpush", "l", "a"}}, method: "redis.LRange", in: &redisV1.LRangeRequest{Key: "l", Start: 5, Stop: 10}, out: &redisV1.BytesListReply{}, want: &redisV1.BytesListReply{}},
		{name: "llen", setup: [][]any{{"rpush", "l", "a", "b"}}, method: "redis.LLen", in: &redisV1.LLenRequest{Key: "l"}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: 2}},
		{name: "lindex negative", setup: [][]any{{"rpush", "l", "a", "b"}}, method: "redis.LIndex", in: &redisV1.LIndexRequest{Key: "l", Index: -1}, out: &redisV1.BytesReply{}, want: &redisV1.BytesReply{Value: b("b")}},
		{name: "lindex out of range is nil", setup: [][]any{{"rpush", "l", "a"}}, method: "redis.LIndex", in: &redisV1.LIndexRequest{Key: "l", Index: 5}, out: &redisV1.BytesReply{}, want: &redisV1.BytesReply{}},

		{name: "lset", setup: [][]any{{"rpush", "l", "a", "b"}}, method: "redis.LSet", in: &redisV1.LSetRequest{Key: "l", Index: 1, Element: b("z")}, out: &redisV1.BoolReply{}, want: &redisV1.BoolReply{Value: true}, after: []any{"lrange", "l", "0", "-1"}, wantAfter: []any{"a", "z"}},
		{name: "lset out of range", setup: [][]any{{"rpush", "l", "a"}}, method: "redis.LSet", in: &redisV1.LSetRequest{Key: "l", Index: 5, Element: b("z")}, out: &redisV1.BoolReply{}, wantErr: "out of range"},
		{name: "lrem all", setup: [][]any{{"rpush", "l", "a", "b", "a"}}, method: "redis.LRem", in: &redisV1.LRemRequest{Key: "l", Count: 0, Element: b("a")}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: 2}},
		{name: "lrem from tail", setup: [][]any{{"rpush", "l", "a", "b", "a"}}, method: "redis.LRem", in: &redisV1.LRemRequest{Key: "l", Count: -1, Element: b("a")}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: 1}, after: []any{"lrange", "l", "0", "-1"}, wantAfter: []any{"a", "b"}},
		{name: "ltrim", setup: [][]any{{"rpush", "l", "a", "b", "c"}}, method: "redis.LTrim", in: &redisV1.LTrimRequest{Key: "l", Start: 1, Stop: -1}, out: &redisV1.BoolReply{}, want: &redisV1.BoolReply{Value: true}, after: []any{"lrange", "l", "0", "-1"}, wantAfter: []any{"b", "c"}},

		{name: "lmove right to left", setup: [][]any{{"rpush", "src", "a", "b"}}, method: "redis.LMove", in: &redisV1.LMoveRequest{Source: "src", Destination: "dst", WhereFrom: "RIGHT", WhereTo: "LEFT"}, out: &redisV1.BytesReply{}, want: &redisV1.BytesReply{Value: b("b")}, after: []any{"lrange", "dst", "0", "-1"}, wantAfter: []any{"b"}},
		{name: "lmove missing source is nil", method: "redis.LMove", in: &redisV1.LMoveRequest{Source: "src", Destination: "dst", WhereFrom: "LEFT", WhereTo: "RIGHT"}, out: &redisV1.BytesReply{}, want: &redisV1.BytesReply{}},
		{name: "lmove bad direction", setup: [][]any{{"rpush", "src", "a"}}, method: "redis.LMove", in: &redisV1.LMoveRequest{Source: "src", Destination: "dst", WhereFrom: "UP", WhereTo: "LEFT"}, out: &redisV1.BytesReply{}, wantErr: "syntax error"},
		{name: "lmove empty destination", method: "redis.LMove", in: &redisV1.LMoveRequest{Source: "src", WhereFrom: "LEFT", WhereTo: "LEFT"}, out: &redisV1.BytesReply{}, wantErr: "key can't be empty string"},
	})
}
