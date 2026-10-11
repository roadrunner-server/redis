package kv

import (
	"testing"

	redisV1 "github.com/roadrunner-server/api-go/v6/redis/v1"
	"google.golang.org/protobuf/proto"
)

func sm(member string, score float64) *redisV1.ScoredMember {
	return &redisV1.ScoredMember{Member: b(member), Score: score}
}

func TestSortedSets(t *testing.T) {
	client := bootRedis(t)

	runCases(t, client, []rpcCase{
		{name: "zadd counts new members", setup: [][]any{{"zadd", "z", "1", "a"}}, method: "redis.ZAdd", in: &redisV1.ZAddRequest{Key: "z", Members: []*redisV1.ScoredMember{sm("a", 5), sm("b", 2)}}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: 1}, after: []any{"zscore", "z", "a"}, wantAfter: float64(5)},
		{name: "zadd ch counts changed", setup: [][]any{{"zadd", "z", "1", "a"}}, method: "redis.ZAdd", in: &redisV1.ZAddRequest{Key: "z", Members: []*redisV1.ScoredMember{sm("a", 5), sm("b", 2)}, Ch: true}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: 2}},
		{name: "zadd nx keeps old score", setup: [][]any{{"zadd", "z", "1", "a"}}, method: "redis.ZAdd", in: &redisV1.ZAddRequest{Key: "z", Members: []*redisV1.ScoredMember{sm("a", 5)}, Nx: true}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: 0}, after: []any{"zscore", "z", "a"}, wantAfter: float64(1)},
		{name: "zadd xx skips new", method: "redis.ZAdd", in: &redisV1.ZAddRequest{Key: "z", Members: []*redisV1.ScoredMember{sm("a", 5)}, Xx: true}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: 0}, after: []any{"zcard", "z"}, wantAfter: int64(0)},
		{name: "zadd gt keeps higher", setup: [][]any{{"zadd", "z", "9", "a"}}, method: "redis.ZAdd", in: &redisV1.ZAddRequest{Key: "z", Members: []*redisV1.ScoredMember{sm("a", 5)}, Gt: true, Ch: true}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: 0}, after: []any{"zscore", "z", "a"}, wantAfter: float64(9)},
		{name: "zadd no members", method: "redis.ZAdd", in: &redisV1.ZAddRequest{Key: "z"}, out: &redisV1.IntReply{}, wantErr: "should provide at least 1 key"},
		{name: "zadd empty member", method: "redis.ZAdd", in: &redisV1.ZAddRequest{Key: "z", Members: []*redisV1.ScoredMember{sm("", 1)}}, out: &redisV1.IntReply{}, wantErr: "key can't be empty string"},
		{name: "zadd wrong type", setup: [][]any{{"set", "z", "x"}}, method: "redis.ZAdd", in: &redisV1.ZAddRequest{Key: "z", Members: []*redisV1.ScoredMember{sm("a", 1)}}, out: &redisV1.IntReply{}, wantErr: "WRONGTYPE"},

		{name: "zrem counts removed", setup: [][]any{{"zadd", "z", "1", "a", "2", "b"}}, method: "redis.ZRem", in: &redisV1.ZRemRequest{Key: "z", Members: bs("a", "x")}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: 1}},
		{name: "zscore", setup: [][]any{{"zadd", "z", "1.5", "a"}}, method: "redis.ZScore", in: &redisV1.ZScoreRequest{Key: "z", Member: b("a")}, out: &redisV1.OptionalFloatReply{}, want: &redisV1.OptionalFloatReply{Value: proto.Float64(1.5)}},
		{name: "zscore missing member is nil", setup: [][]any{{"zadd", "z", "1", "a"}}, method: "redis.ZScore", in: &redisV1.ZScoreRequest{Key: "z", Member: b("b")}, out: &redisV1.OptionalFloatReply{}, want: &redisV1.OptionalFloatReply{}},
		{name: "zscore empty member", method: "redis.ZScore", in: &redisV1.ZScoreRequest{Key: "z", Member: b(" ")}, out: &redisV1.OptionalFloatReply{}, wantErr: "key can't be empty string"},
		{name: "zincrby new member", method: "redis.ZIncrBy", in: &redisV1.ZIncrByRequest{Key: "z", Increment: 2.5, Member: b("a")}, out: &redisV1.FloatReply{}, want: &redisV1.FloatReply{Value: 2.5}},
		{name: "zincrby existing", setup: [][]any{{"zadd", "z", "1", "a"}}, method: "redis.ZIncrBy", in: &redisV1.ZIncrByRequest{Key: "z", Increment: -3, Member: b("a")}, out: &redisV1.FloatReply{}, want: &redisV1.FloatReply{Value: -2}},
		{name: "zcard", setup: [][]any{{"zadd", "z", "1", "a", "2", "b"}}, method: "redis.ZCard", in: &redisV1.ZCardRequest{Key: "z"}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: 2}},
		{name: "zcount exclusive min", setup: [][]any{{"zadd", "z", "1", "a", "2", "b", "3", "c"}}, method: "redis.ZCount", in: &redisV1.ZCountRequest{Key: "z", Min: "(1", Max: "+inf"}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: 2}},
		{name: "zrank", setup: [][]any{{"zadd", "z", "1", "a", "2", "b"}}, method: "redis.ZRank", in: &redisV1.ZRankRequest{Key: "z", Member: b("b")}, out: &redisV1.OptionalIntReply{}, want: &redisV1.OptionalIntReply{Value: proto.Int64(1)}},
		{name: "zrank missing is nil", setup: [][]any{{"zadd", "z", "1", "a"}}, method: "redis.ZRank", in: &redisV1.ZRankRequest{Key: "z", Member: b("x")}, out: &redisV1.OptionalIntReply{}, want: &redisV1.OptionalIntReply{}},
		{name: "zrevrank", setup: [][]any{{"zadd", "z", "1", "a", "2", "b"}}, method: "redis.ZRevRank", in: &redisV1.ZRevRankRequest{Key: "z", Member: b("b")}, out: &redisV1.OptionalIntReply{}, want: &redisV1.OptionalIntReply{Value: proto.Int64(0)}},

		{name: "zrange by index", setup: [][]any{{"zadd", "z", "1", "a", "2", "b", "3", "c"}}, method: "redis.ZRange", in: &redisV1.ZRangeRequest{Key: "z", Start: "0", Stop: "1"}, out: &redisV1.ScoredMembersReply{}, want: &redisV1.ScoredMembersReply{Members: []*redisV1.ScoredMember{sm("a", 0), sm("b", 0)}}},
		{name: "zrange with scores", setup: [][]any{{"zadd", "z", "1", "a", "2", "b"}}, method: "redis.ZRange", in: &redisV1.ZRangeRequest{Key: "z", Start: "0", Stop: "-1", WithScores: true}, out: &redisV1.ScoredMembersReply{}, want: &redisV1.ScoredMembersReply{Members: []*redisV1.ScoredMember{sm("a", 1), sm("b", 2)}}},
		{name: "zrange by score rev limit", setup: [][]any{{"zadd", "z", "1", "a", "2", "b", "3", "c"}}, method: "redis.ZRange", in: &redisV1.ZRangeRequest{Key: "z", Start: "+inf", Stop: "(1", ByScore: true, Rev: true, Offset: 0, Count: 1, WithScores: true}, out: &redisV1.ScoredMembersReply{}, want: &redisV1.ScoredMembersReply{Members: []*redisV1.ScoredMember{sm("c", 3)}}},
		{name: "zrange by lex", setup: [][]any{{"zadd", "z", "0", "a", "0", "b", "0", "c"}}, method: "redis.ZRange", in: &redisV1.ZRangeRequest{Key: "z", Start: "[b", Stop: "+", ByLex: true}, out: &redisV1.ScoredMembersReply{}, want: &redisV1.ScoredMembersReply{Members: []*redisV1.ScoredMember{sm("b", 0), sm("c", 0)}}},
		{name: "zrange by score and by lex", method: "redis.ZRange", in: &redisV1.ZRangeRequest{Key: "z", Start: "0", Stop: "1", ByScore: true, ByLex: true}, out: &redisV1.ScoredMembersReply{}, wantErr: "mutually exclusive"},
		{name: "zrange missing is empty", method: "redis.ZRange", in: &redisV1.ZRangeRequest{Key: "z", Start: "0", Stop: "-1"}, out: &redisV1.ScoredMembersReply{}, want: &redisV1.ScoredMembersReply{}},

		{name: "zpopmin count 0 pops one", setup: [][]any{{"zadd", "z", "1", "a", "2", "b"}}, method: "redis.ZPopMin", in: &redisV1.ZPopMinRequest{Key: "z"}, out: &redisV1.ScoredMembersReply{}, want: &redisV1.ScoredMembersReply{Members: []*redisV1.ScoredMember{sm("a", 1)}}, after: []any{"zcard", "z"}, wantAfter: int64(1)},
		{name: "zpopmax count 2", setup: [][]any{{"zadd", "z", "1", "a", "2", "b", "3", "c"}}, method: "redis.ZPopMax", in: &redisV1.ZPopMaxRequest{Key: "z", Count: 2}, out: &redisV1.ScoredMembersReply{}, want: &redisV1.ScoredMembersReply{Members: []*redisV1.ScoredMember{sm("c", 3), sm("b", 2)}}},
		{name: "zpopmin missing is empty", method: "redis.ZPopMin", in: &redisV1.ZPopMinRequest{Key: "z"}, out: &redisV1.ScoredMembersReply{}, want: &redisV1.ScoredMembersReply{}},
		{name: "zremrangebyrank", setup: [][]any{{"zadd", "z", "1", "a", "2", "b", "3", "c"}}, method: "redis.ZRemRangeByRank", in: &redisV1.ZRemRangeByRankRequest{Key: "z", Start: 0, Stop: 1}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: 2}, after: []any{"zrange", "z", "0", "-1"}, wantAfter: []any{"c"}},
		{name: "zremrangebyscore", setup: [][]any{{"zadd", "z", "1", "a", "2", "b", "3", "c"}}, method: "redis.ZRemRangeByScore", in: &redisV1.ZRemRangeByScoreRequest{Key: "z", Min: "-inf", Max: "(3"}, out: &redisV1.IntReply{}, want: &redisV1.IntReply{Value: 2}},
	})
}
