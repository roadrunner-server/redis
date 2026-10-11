package kv

import (
	"testing"

	redisV1 "github.com/roadrunner-server/api-go/v6/redis/v1"
)

func TestSets(t *testing.T) {
	client := bootRedis(t)

	runCases(t, client, []rpcCase{
		{name: "sadd counts new members", setup: [][]any{{"sadd", "s", "a"}}, method: "redis.SAdd", in: &redisV1.SAddRequest{Connection: connDefault, Key: "s", Members: bs("a", "b")}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: 1}},
		{name: "sadd no members", method: "redis.SAdd", in: &redisV1.SAddRequest{Connection: connDefault, Key: "s"}, out: &redisV1.IntReply{}, wantErr: "should provide at least 1 key"},
		{name: "sadd empty member", method: "redis.SAdd", in: &redisV1.SAddRequest{Connection: connDefault, Key: "s", Members: bs(" ")}, out: &redisV1.IntReply{}, wantErr: "key can't be empty string"},
		{name: "sadd wrong type", setup: [][]any{{"set", "s", "x"}}, method: "redis.SAdd", in: &redisV1.SAddRequest{Connection: connDefault, Key: "s", Members: bs("a")}, out: &redisV1.IntReply{}, wantErr: "WRONGTYPE"},
		{name: "srem counts removed", setup: [][]any{{"sadd", "s", "a", "b"}}, method: "redis.SRem", in: &redisV1.SRemRequest{Connection: connDefault, Key: "s", Members: bs("a", "z")}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: 1}},
		{name: "smembers", setup: [][]any{{"sadd", "s", "a", "b"}}, method: "redis.SMembers", in: &redisV1.SMembersRequest{Connection: connDefault, Key: "s"}, out: &redisV1.BytesListReply{}, want: &redisV1.BytesListReply{Values: bs("a", "b")}, unordered: true},
		{name: "smembers missing is empty", method: "redis.SMembers", in: &redisV1.SMembersRequest{Connection: connDefault, Key: "s"}, out: &redisV1.BytesListReply{}, want: &redisV1.BytesListReply{}},
		{name: "sismember", setup: [][]any{{"sadd", "s", "a"}}, method: "redis.SIsMember", in: &redisV1.SIsMemberRequest{Connection: connDefault, Key: "s", Member: b("a")}, out: &redisV1.BoolReply{}, want: &redisV1.BoolReply{Value: true}},
		{name: "sismember missing", method: "redis.SIsMember", in: &redisV1.SIsMemberRequest{Connection: connDefault, Key: "s", Member: b("a")}, out: &redisV1.BoolReply{}, want: &redisV1.BoolReply{Value: false}},
		{name: "smismember keeps positions", setup: [][]any{{"sadd", "s", "a"}}, method: "redis.SMIsMember", in: &redisV1.SMIsMemberRequest{Connection: connDefault, Key: "s", Members: bs("a", "b")}, out: &redisV1.BoolListReply{}, want: &redisV1.BoolListReply{Values: []bool{true, false}}},
		{name: "scard", setup: [][]any{{"sadd", "s", "a", "b"}}, method: "redis.SCard", in: &redisV1.SCardRequest{Connection: connDefault, Key: "s"}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: 2}},

		{name: "spop single", setup: [][]any{{"sadd", "s", "a"}}, method: "redis.SPop", in: &redisV1.SPopRequest{Connection: connDefault, Key: "s"}, out: &redisV1.BytesReply{}, want: &redisV1.BytesReply{Value: b("a")}, after: []any{"scard", "s"}, wantAfter: int64(0)},
		{name: "spop missing is nil", method: "redis.SPop", in: &redisV1.SPopRequest{Connection: connDefault, Key: "s"}, out: &redisV1.BytesReply{}, want: &redisV1.BytesReply{}},
		{name: "spopn", setup: [][]any{{"sadd", "s", "a", "b"}}, method: "redis.SPopN", in: &redisV1.SPopNRequest{Connection: connDefault, Key: "s", Count: 5}, out: &redisV1.BytesListReply{}, want: &redisV1.BytesListReply{Values: bs("a", "b")}, unordered: true},
		{name: "srandmember single", setup: [][]any{{"sadd", "s", "a"}}, method: "redis.SRandMember", in: &redisV1.SRandMemberRequest{Connection: connDefault, Key: "s"}, out: &redisV1.BytesReply{}, want: &redisV1.BytesReply{Value: b("a")}, after: []any{"scard", "s"}, wantAfter: int64(1)},
		{name: "srandmembern negative allows repeats", setup: [][]any{{"sadd", "s", "a"}}, method: "redis.SRandMemberN", in: &redisV1.SRandMemberNRequest{Connection: connDefault, Key: "s", Count: -2}, out: &redisV1.BytesListReply{}, want: &redisV1.BytesListReply{Values: bs("a", "a")}},

		{name: "sdiff", setup: [][]any{{"sadd", "a", "1", "2", "3"}, {"sadd", "b", "2"}}, method: "redis.SDiff", in: &redisV1.SDiffRequest{Connection: connDefault, Keys: []string{"a", "b"}}, out: &redisV1.BytesListReply{}, want: &redisV1.BytesListReply{Values: bs("1", "3")}, unordered: true},
		{name: "sinter", setup: [][]any{{"sadd", "a", "1", "2"}, {"sadd", "b", "2", "3"}}, method: "redis.SInter", in: &redisV1.SInterRequest{Connection: connDefault, Keys: []string{"a", "b"}}, out: &redisV1.BytesListReply{}, want: &redisV1.BytesListReply{Values: bs("2")}},
		{name: "sunion", setup: [][]any{{"sadd", "a", "1"}, {"sadd", "b", "2"}}, method: "redis.SUnion", in: &redisV1.SUnionRequest{Connection: connDefault, Keys: []string{"a", "b"}}, out: &redisV1.BytesListReply{}, want: &redisV1.BytesListReply{Values: bs("1", "2")}, unordered: true},
		{name: "sunion no keys", method: "redis.SUnion", in: &redisV1.SUnionRequest{Connection: connDefault}, out: &redisV1.BytesListReply{}, wantErr: "should provide at least 1 key"},
	})
}
