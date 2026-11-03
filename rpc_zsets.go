package redis

import (
	stderr "errors"

	"github.com/redis/go-redis/v9"
	redisV1 "github.com/roadrunner-server/api-go/v6/redis/v1"
	"github.com/roadrunner-server/errors"
)

var (
	errByScoreByLex = stderr.New("by_score and by_lex are mutually exclusive")
	errGtLtNx       = stderr.New("gt, lt and nx are mutually exclusive")
)

func (r *rpc) ZAdd(in *redisV1.ZAddRequest, out *redisV1.IntReply) error {
	cl := r.begin("zadd", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	if in.GetNx() && in.GetXx() {
		return cl.fail(errNxXx)
	}
	if (in.GetGt() && in.GetLt()) || (in.GetNx() && (in.GetGt() || in.GetLt())) {
		return cl.fail(errGtLtNx)
	}
	if len(in.GetMembers()) == 0 {
		return cl.fail(errors.E(errors.NoKeys))
	}
	members := make([]redis.Z, 0, len(in.GetMembers()))
	for _, m := range in.GetMembers() {
		if err := requireMember(m.GetMember()); err != nil {
			return cl.fail(err)
		}
		members = append(members, redis.Z{Score: m.GetScore(), Member: m.GetMember()})
	}
	args := redis.ZAddArgs{NX: in.GetNx(), XX: in.GetXx(), GT: in.GetGt(), LT: in.GetLt(), Ch: in.GetCh(), Members: members}
	return cl.intReply(cl.c.ZAddArgs(cl.ctx, in.GetKey(), args), out)
}

func (r *rpc) ZRem(in *redisV1.ZRemRequest, out *redisV1.IntReply) error {
	cl := r.begin("zrem", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	if err := requireMembers(in.GetMembers()); err != nil {
		return cl.fail(err)
	}
	return cl.intReply(cl.c.ZRem(cl.ctx, in.GetKey(), toArgs(in.GetMembers())...), out)
}

func (r *rpc) ZScore(in *redisV1.ZScoreRequest, out *redisV1.OptionalFloatReply) error {
	cl := r.begin("zscore", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	if err := requireMember(in.GetMember()); err != nil {
		return cl.fail(err)
	}
	return cl.optFloatReply(cl.c.ZScore(cl.ctx, in.GetKey(), string(in.GetMember())), out)
}

func (r *rpc) ZIncrBy(in *redisV1.ZIncrByRequest, out *redisV1.FloatReply) error {
	cl := r.begin("zincrby", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	if err := requireMember(in.GetMember()); err != nil {
		return cl.fail(err)
	}
	return cl.floatReply(cl.c.ZIncrBy(cl.ctx, in.GetKey(), in.GetIncrement(), string(in.GetMember())), out)
}

func (r *rpc) ZCard(in *redisV1.ZCardRequest, out *redisV1.IntReply) error {
	cl := r.begin("zcard", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.intReply(cl.c.ZCard(cl.ctx, in.GetKey()), out)
}

func (r *rpc) ZCount(in *redisV1.ZCountRequest, out *redisV1.IntReply) error {
	cl := r.begin("zcount", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.intReply(cl.c.ZCount(cl.ctx, in.GetKey(), in.GetMin(), in.GetMax()), out)
}

func (r *rpc) ZRank(in *redisV1.ZRankRequest, out *redisV1.OptionalIntReply) error {
	cl := r.begin("zrank", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	if err := requireMember(in.GetMember()); err != nil {
		return cl.fail(err)
	}
	return cl.optIntReply(cl.c.ZRank(cl.ctx, in.GetKey(), string(in.GetMember())), out)
}

func (r *rpc) ZRevRank(in *redisV1.ZRevRankRequest, out *redisV1.OptionalIntReply) error {
	cl := r.begin("zrevrank", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	if err := requireMember(in.GetMember()); err != nil {
		return cl.fail(err)
	}
	return cl.optIntReply(cl.c.ZRevRank(cl.ctx, in.GetKey(), string(in.GetMember())), out)
}

func (r *rpc) ZRange(in *redisV1.ZRangeRequest, out *redisV1.ScoredMembersReply) error {
	cl := r.begin("zrange", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	if in.GetByScore() && in.GetByLex() {
		return cl.fail(errByScoreByLex)
	}
	args := redis.ZRangeArgs{
		Key:     in.GetKey(),
		Start:   in.GetStart(),
		Stop:    in.GetStop(),
		ByScore: in.GetByScore(),
		ByLex:   in.GetByLex(),
		Rev:     in.GetRev(),
		Offset:  in.GetOffset(),
		Count:   in.GetCount(),
	}
	if in.GetWithScores() {
		return cl.scoredReply(cl.c.ZRangeArgsWithScores(cl.ctx, args), out)
	}
	members, err := cl.c.ZRangeArgs(cl.ctx, args).Result()
	if err != nil {
		return cl.fail(err)
	}
	out.Members = make([]*redisV1.ScoredMember, 0, len(members))
	for _, m := range members {
		out.Members = append(out.Members, &redisV1.ScoredMember{Member: []byte(m)})
	}
	return nil
}

func (r *rpc) ZPopMin(in *redisV1.ZPopMinRequest, out *redisV1.ScoredMembersReply) error {
	cl := r.begin("zpopmin", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.scoredReply(cl.c.ZPopMin(cl.ctx, in.GetKey(), int64(popCount(in.GetCount()))), out)
}

func (r *rpc) ZPopMax(in *redisV1.ZPopMaxRequest, out *redisV1.ScoredMembersReply) error {
	cl := r.begin("zpopmax", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.scoredReply(cl.c.ZPopMax(cl.ctx, in.GetKey(), int64(popCount(in.GetCount()))), out)
}

func (r *rpc) ZRemRangeByRank(in *redisV1.ZRemRangeByRankRequest, out *redisV1.IntReply) error {
	cl := r.begin("zremrangebyrank", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.intReply(cl.c.ZRemRangeByRank(cl.ctx, in.GetKey(), in.GetStart(), in.GetStop()), out)
}

func (r *rpc) ZRemRangeByScore(in *redisV1.ZRemRangeByScoreRequest, out *redisV1.IntReply) error {
	cl := r.begin("zremrangebyscore", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.intReply(cl.c.ZRemRangeByScore(cl.ctx, in.GetKey(), in.GetMin(), in.GetMax()), out)
}
