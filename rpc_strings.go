package redis

import (
	stderr "errors"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	redisV1 "github.com/roadrunner-server/api-go/v6/redis/v1"
	"github.com/roadrunner-server/errors"
)

var (
	errNxXx   = stderr.New("nx and xx are mutually exclusive")
	errExPx   = stderr.New("ex and px are mutually exclusive")
	errExpire = stderr.New("invalid expire time")
)

func (r *rpc) Get(in *redisV1.GetRequest, out *redisV1.BytesReply) error {
	cl := r.begin("get", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.bytesReply(cl.c.Get(cl.ctx, in.GetKey()), out)
}

func (r *rpc) Set(in *redisV1.SetRequest, out *redisV1.BoolReply) error {
	cl := r.begin("set", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	if in.GetNx() && in.GetXx() {
		return cl.fail(errNxXx)
	}
	if in.Ex != nil && in.Px != nil {
		return cl.fail(errExPx)
	}
	if (in.Ex != nil && in.GetEx() <= 0) || (in.Px != nil && in.GetPx() <= 0) {
		return cl.fail(errExpire)
	}

	args := redis.SetArgs{KeepTTL: in.GetKeepTtl()}
	switch {
	case in.GetNx():
		args.Mode = "NX"
	case in.GetXx():
		args.Mode = "XX"
	}
	switch {
	case in.Ex != nil:
		args.TTL = time.Duration(in.GetEx()) * time.Second
	case in.Px != nil:
		args.TTL = time.Duration(in.GetPx()) * time.Millisecond
	}

	err := cl.c.SetArgs(cl.ctx, in.GetKey(), in.GetValue(), args).Err()
	if stderr.Is(err, redis.Nil) {
		out.Value = false
		return nil
	}
	if err != nil {
		return cl.fail(err)
	}
	out.Value = true
	return nil
}

func (r *rpc) MGet(in *redisV1.MGetRequest, out *redisV1.BytesListReply) error {
	cl := r.begin("mget", in.GetConnection())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	if err := requireKeys(in.GetKeys()); err != nil {
		return cl.fail(err)
	}
	return cl.sliceReply(cl.c.MGet(cl.ctx, in.GetKeys()...), out)
}

func (r *rpc) MSet(in *redisV1.MSetRequest, _ *redisV1.Empty) error {
	cl := r.begin("mset", in.GetConnection())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	if len(in.GetValues()) == 0 {
		return cl.fail(errors.E(errors.NoKeys))
	}
	args := make([]any, 0, 2*len(in.GetValues()))
	for k, v := range in.GetValues() {
		if strings.TrimSpace(k) == "" {
			return cl.fail(errors.E(errors.EmptyKey))
		}
		args = append(args, k, v)
	}
	if err := cl.c.MSet(cl.ctx, args...).Err(); err != nil {
		return cl.fail(err)
	}
	return nil
}

func (r *rpc) GetDel(in *redisV1.GetDelRequest, out *redisV1.BytesReply) error {
	cl := r.begin("getdel", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.bytesReply(cl.c.GetDel(cl.ctx, in.GetKey()), out)
}

func (r *rpc) Append(in *redisV1.AppendRequest, out *redisV1.IntReply) error {
	cl := r.begin("append", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.intReply(cl.c.Append(cl.ctx, in.GetKey(), string(in.GetValue())), out)
}

func (r *rpc) StrLen(in *redisV1.StrLenRequest, out *redisV1.IntReply) error {
	cl := r.begin("strlen", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.intReply(cl.c.StrLen(cl.ctx, in.GetKey()), out)
}

func (r *rpc) Incr(in *redisV1.IncrRequest, out *redisV1.IntReply) error {
	cl := r.begin("incr", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.intReply(cl.c.Incr(cl.ctx, in.GetKey()), out)
}

func (r *rpc) IncrBy(in *redisV1.IncrByRequest, out *redisV1.IntReply) error {
	cl := r.begin("incrby", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.intReply(cl.c.IncrBy(cl.ctx, in.GetKey(), in.GetIncrement()), out)
}

func (r *rpc) IncrByFloat(in *redisV1.IncrByFloatRequest, out *redisV1.FloatReply) error {
	cl := r.begin("incrbyfloat", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.floatReply(cl.c.IncrByFloat(cl.ctx, in.GetKey(), in.GetIncrement()), out)
}

func (r *rpc) Decr(in *redisV1.DecrRequest, out *redisV1.IntReply) error {
	cl := r.begin("decr", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.intReply(cl.c.Decr(cl.ctx, in.GetKey()), out)
}

func (r *rpc) DecrBy(in *redisV1.DecrByRequest, out *redisV1.IntReply) error {
	cl := r.begin("decrby", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.intReply(cl.c.DecrBy(cl.ctx, in.GetKey(), in.GetDecrement()), out)
}
