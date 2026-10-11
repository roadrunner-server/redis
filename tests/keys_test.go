package kv

import (
	"testing"
	"time"

	redisV1 "github.com/roadrunner-server/api-go/v6/redis/v1"
)

func TestKeys(t *testing.T) {
	client := bootRedis(t)
	future := time.Now().Add(time.Hour).Unix()

	runCases(t, client, []rpcCase{
		{name: "del counts removed", setup: [][]any{{"mset", "a", "1", "b", "2"}}, method: "redis.Del", in: &redisV1.DelRequest{Keys: []string{"a", "b", "c"}}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: 2}},
		{name: "del no keys", method: "redis.Del", in: &redisV1.DelRequest{}, out: &redisV1.IntReply{}, wantErr: "should provide at least 1 key"},
		{name: "unlink counts removed", setup: [][]any{{"set", "a", "1"}}, method: "redis.Unlink", in: &redisV1.UnlinkRequest{Keys: []string{"a", "b"}}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: 1}},
		{name: "exists counts repeated keys", setup: [][]any{{"set", "a", "1"}}, method: "redis.Exists", in: &redisV1.ExistsRequest{Keys: []string{"a", "a", "b"}}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: 2}},

		{name: "expire existing", setup: [][]any{{"set", "a", "1"}}, method: "redis.Expire", in: &redisV1.ExpireRequest{Key: "a", Seconds: 100}, out: &redisV1.BoolReply{}, want: &redisV1.BoolReply{Value: true}, after: []any{"ttl", "a"}, wantAfter: int64(100)},
		{name: "expire missing", method: "redis.Expire", in: &redisV1.ExpireRequest{Key: "a", Seconds: 100}, out: &redisV1.BoolReply{}, want: &redisV1.BoolReply{Value: false}},
		{name: "pexpire", setup: [][]any{{"set", "a", "1"}}, method: "redis.PExpire", in: &redisV1.PExpireRequest{Key: "a", Milliseconds: 100000}, out: &redisV1.BoolReply{}, want: &redisV1.BoolReply{Value: true}, after: []any{"ttl", "a"}, wantAfter: int64(100)},
		{name: "expireat", setup: [][]any{{"set", "a", "1"}}, method: "redis.ExpireAt", in: &redisV1.ExpireAtRequest{Key: "a", UnixSeconds: future}, out: &redisV1.BoolReply{}, want: &redisV1.BoolReply{Value: true}, after: []any{"expiretime", "a"}, wantAfter: future},

		{name: "ttl missing is -2", method: "redis.TTL", in: &redisV1.TTLRequest{Key: "a"}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: -2}},
		{name: "ttl no expiry is -1", setup: [][]any{{"set", "a", "1"}}, method: "redis.TTL", in: &redisV1.TTLRequest{Key: "a"}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: -1}},
		{name: "ttl set", setup: [][]any{{"set", "a", "1", "ex", "100"}}, method: "redis.TTL", in: &redisV1.TTLRequest{Key: "a"}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: 100}},
		{name: "pttl missing is -2", method: "redis.PTTL", in: &redisV1.PTTLRequest{Key: "a"}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: -2}},

		{name: "persist removes ttl", setup: [][]any{{"set", "a", "1", "ex", "100"}}, method: "redis.Persist", in: &redisV1.PersistRequest{Key: "a"}, out: &redisV1.BoolReply{}, want: &redisV1.BoolReply{Value: true}, after: []any{"ttl", "a"}, wantAfter: int64(-1)},
		{name: "persist without ttl", setup: [][]any{{"set", "a", "1"}}, method: "redis.Persist", in: &redisV1.PersistRequest{Key: "a"}, out: &redisV1.BoolReply{}, want: &redisV1.BoolReply{Value: false}},

		{name: "type string", setup: [][]any{{"set", "a", "1"}}, method: "redis.Type", in: &redisV1.TypeRequest{Key: "a"}, out: &redisV1.StringReply{}, want: &redisV1.StringReply{Value: "string"}},
		{name: "type hash", setup: [][]any{{"hset", "a", "f", "1"}}, method: "redis.Type", in: &redisV1.TypeRequest{Key: "a"}, out: &redisV1.StringReply{}, want: &redisV1.StringReply{Value: "hash"}},
		{name: "type missing is none", method: "redis.Type", in: &redisV1.TypeRequest{Key: "a"}, out: &redisV1.StringReply{}, want: &redisV1.StringReply{Value: "none"}},

		{name: "rename", setup: [][]any{{"set", "a", "1"}}, method: "redis.Rename", in: &redisV1.RenameRequest{Key: "a", NewKey: "b"}, out: &redisV1.BoolReply{}, want: &redisV1.BoolReply{Value: true}, after: []any{"mget", "a", "b"}, wantAfter: []any{nil, "1"}},
		{name: "rename missing", method: "redis.Rename", in: &redisV1.RenameRequest{Key: "a", NewKey: "b"}, out: &redisV1.BoolReply{}, wantErr: "no such key"},
		{name: "rename empty new key", setup: [][]any{{"set", "a", "1"}}, method: "redis.Rename", in: &redisV1.RenameRequest{Key: "a", NewKey: ""}, out: &redisV1.BoolReply{}, wantErr: "key can't be empty string"},

		{name: "scan match", setup: [][]any{{"mset", "u:1", "a", "u:2", "b", "o:1", "c"}}, method: "redis.Scan", in: &redisV1.ScanRequest{Match: "u:*", Count: 100}, out: &redisV1.ScanReply{}, want: &redisV1.ScanReply{Cursor: 0, Keys: []string{"u:1", "u:2"}}, unordered: true},
		{name: "scan type", setup: [][]any{{"set", "s", "a"}, {"hset", "h", "f", "b"}}, method: "redis.Scan", in: &redisV1.ScanRequest{Type: "hash", Count: 100}, out: &redisV1.ScanReply{}, want: &redisV1.ScanReply{Cursor: 0, Keys: []string{"h"}}},
		{name: "scan empty db", method: "redis.Scan", in: &redisV1.ScanRequest{}, out: &redisV1.ScanReply{}, want: &redisV1.ScanReply{Cursor: 0}},
	})
}
