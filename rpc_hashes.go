package redis

import (
	redisV1 "github.com/roadrunner-server/api-go/v6/redis/v1"
	"github.com/roadrunner-server/errors"
)

func (r *rpc) HSet(in *redisV1.HSetRequest, out *redisV1.IntReply) error {
	cl := r.begin("hset", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	if len(in.GetFields()) == 0 {
		return cl.fail(errors.E(errors.NoKeys))
	}
	args := make([]any, 0, 2*len(in.GetFields()))
	for f, v := range in.GetFields() {
		if err := requireKey(f); err != nil {
			return cl.fail(err)
		}
		args = append(args, f, v)
	}
	return cl.intReply(cl.c.HSet(cl.ctx, in.GetKey(), args...), out)
}

func (r *rpc) HSetNX(in *redisV1.HSetNXRequest, out *redisV1.BoolReply) error {
	cl := r.begin("hsetnx", in.GetConnection(), in.GetKey(), in.GetField())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.boolReply(cl.c.HSetNX(cl.ctx, in.GetKey(), in.GetField(), in.GetValue()), out)
}

func (r *rpc) HGet(in *redisV1.HGetRequest, out *redisV1.BytesReply) error {
	cl := r.begin("hget", in.GetConnection(), in.GetKey(), in.GetField())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.bytesReply(cl.c.HGet(cl.ctx, in.GetKey(), in.GetField()), out)
}

func (r *rpc) HMGet(in *redisV1.HMGetRequest, out *redisV1.BytesListReply) error {
	cl := r.begin("hmget", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	if err := requireKeys(in.GetFields()); err != nil {
		return cl.fail(err)
	}
	return cl.sliceReply(cl.c.HMGet(cl.ctx, in.GetKey(), in.GetFields()...), out)
}

func (r *rpc) HGetAll(in *redisV1.HGetAllRequest, out *redisV1.MapReply) error {
	cl := r.begin("hgetall", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.mapReply(cl.c.HGetAll(cl.ctx, in.GetKey()), out)
}

func (r *rpc) HDel(in *redisV1.HDelRequest, out *redisV1.IntReply) error {
	cl := r.begin("hdel", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	if err := requireKeys(in.GetFields()); err != nil {
		return cl.fail(err)
	}
	return cl.intReply(cl.c.HDel(cl.ctx, in.GetKey(), in.GetFields()...), out)
}

func (r *rpc) HExists(in *redisV1.HExistsRequest, out *redisV1.BoolReply) error {
	cl := r.begin("hexists", in.GetConnection(), in.GetKey(), in.GetField())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.boolReply(cl.c.HExists(cl.ctx, in.GetKey(), in.GetField()), out)
}

func (r *rpc) HLen(in *redisV1.HLenRequest, out *redisV1.IntReply) error {
	cl := r.begin("hlen", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.intReply(cl.c.HLen(cl.ctx, in.GetKey()), out)
}

func (r *rpc) HKeys(in *redisV1.HKeysRequest, out *redisV1.BytesListReply) error {
	cl := r.begin("hkeys", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.bytesListReply(cl.c.HKeys(cl.ctx, in.GetKey()), out)
}

func (r *rpc) HVals(in *redisV1.HValsRequest, out *redisV1.BytesListReply) error {
	cl := r.begin("hvals", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.bytesListReply(cl.c.HVals(cl.ctx, in.GetKey()), out)
}

func (r *rpc) HIncrBy(in *redisV1.HIncrByRequest, out *redisV1.IntReply) error {
	cl := r.begin("hincrby", in.GetConnection(), in.GetKey(), in.GetField())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.intReply(cl.c.HIncrBy(cl.ctx, in.GetKey(), in.GetField(), in.GetIncrement()), out)
}

func (r *rpc) HIncrByFloat(in *redisV1.HIncrByFloatRequest, out *redisV1.FloatReply) error {
	cl := r.begin("hincrbyfloat", in.GetConnection(), in.GetKey(), in.GetField())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.floatReply(cl.c.HIncrByFloat(cl.ctx, in.GetKey(), in.GetField(), in.GetIncrement()), out)
}
