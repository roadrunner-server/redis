package redis

import (
	redisV1 "github.com/roadrunner-server/api-go/v6/redis/v1"
	"github.com/roadrunner-server/errors"
)

func (r *rpc) LPush(in *redisV1.LPushRequest, out *redisV1.IntReply) error {
	cl := r.begin("lpush", in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	if len(in.GetElements()) == 0 {
		return cl.fail(errors.E(errors.NoKeys))
	}
	return cl.intReply(cl.c.LPush(cl.ctx, in.GetKey(), toArgs(in.GetElements())...), out)
}

func (r *rpc) RPush(in *redisV1.RPushRequest, out *redisV1.IntReply) error {
	cl := r.begin("rpush", in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	if len(in.GetElements()) == 0 {
		return cl.fail(errors.E(errors.NoKeys))
	}
	return cl.intReply(cl.c.RPush(cl.ctx, in.GetKey(), toArgs(in.GetElements())...), out)
}

func (r *rpc) LPop(in *redisV1.LPopRequest, out *redisV1.BytesListReply) error {
	cl := r.begin("lpop", in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.bytesListReply(cl.c.LPopCount(cl.ctx, in.GetKey(), popCount(in.GetCount())), out)
}

func (r *rpc) RPop(in *redisV1.RPopRequest, out *redisV1.BytesListReply) error {
	cl := r.begin("rpop", in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.bytesListReply(cl.c.RPopCount(cl.ctx, in.GetKey(), popCount(in.GetCount())), out)
}

// popCount maps the request count to the Redis COUNT argument: 0 pops one element.
func popCount(n int64) int {
	if n <= 0 {
		return 1
	}
	return int(n)
}

func (r *rpc) LRange(in *redisV1.LRangeRequest, out *redisV1.BytesListReply) error {
	cl := r.begin("lrange", in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.bytesListReply(cl.c.LRange(cl.ctx, in.GetKey(), in.GetStart(), in.GetStop()), out)
}

func (r *rpc) LLen(in *redisV1.LLenRequest, out *redisV1.IntReply) error {
	cl := r.begin("llen", in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.intReply(cl.c.LLen(cl.ctx, in.GetKey()), out)
}

func (r *rpc) LIndex(in *redisV1.LIndexRequest, out *redisV1.BytesReply) error {
	cl := r.begin("lindex", in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.bytesReply(cl.c.LIndex(cl.ctx, in.GetKey(), in.GetIndex()), out)
}

func (r *rpc) LSet(in *redisV1.LSetRequest, out *redisV1.BoolReply) error {
	cl := r.begin("lset", in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.statusReply(cl.c.LSet(cl.ctx, in.GetKey(), in.GetIndex(), in.GetElement()), out)
}

func (r *rpc) LRem(in *redisV1.LRemRequest, out *redisV1.IntReply) error {
	cl := r.begin("lrem", in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.intReply(cl.c.LRem(cl.ctx, in.GetKey(), in.GetCount(), in.GetElement()), out)
}

func (r *rpc) LTrim(in *redisV1.LTrimRequest, out *redisV1.BoolReply) error {
	cl := r.begin("ltrim", in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.statusReply(cl.c.LTrim(cl.ctx, in.GetKey(), in.GetStart(), in.GetStop()), out)
}

func (r *rpc) LMove(in *redisV1.LMoveRequest, out *redisV1.BytesReply) error {
	cl := r.begin("lmove", in.GetSource(), in.GetDestination())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.bytesReply(cl.c.LMove(cl.ctx, in.GetSource(), in.GetDestination(), in.GetWhereFrom(), in.GetWhereTo()), out)
}
