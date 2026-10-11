package redis

import (
	"time"

	redisV1 "github.com/roadrunner-server/api-go/v6/redis/v1"
)

func (r *rpc) Del(in *redisV1.DelRequest, out *redisV1.IntReply) error {
	cl := r.begin("del")
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	if err := requireKeys(in.GetKeys()); err != nil {
		return cl.fail(err)
	}
	return cl.intReply(cl.c.Del(cl.ctx, in.GetKeys()...), out)
}

func (r *rpc) Unlink(in *redisV1.UnlinkRequest, out *redisV1.IntReply) error {
	cl := r.begin("unlink")
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	if err := requireKeys(in.GetKeys()); err != nil {
		return cl.fail(err)
	}
	return cl.intReply(cl.c.Unlink(cl.ctx, in.GetKeys()...), out)
}

func (r *rpc) Exists(in *redisV1.ExistsRequest, out *redisV1.IntReply) error {
	cl := r.begin("exists")
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	if err := requireKeys(in.GetKeys()); err != nil {
		return cl.fail(err)
	}
	return cl.intReply(cl.c.Exists(cl.ctx, in.GetKeys()...), out)
}

func (r *rpc) Expire(in *redisV1.ExpireRequest, out *redisV1.BoolReply) error {
	cl := r.begin("expire", in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.boolReply(cl.c.Expire(cl.ctx, in.GetKey(), time.Duration(in.GetSeconds())*time.Second), out)
}

func (r *rpc) PExpire(in *redisV1.PExpireRequest, out *redisV1.BoolReply) error {
	cl := r.begin("pexpire", in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.boolReply(cl.c.PExpire(cl.ctx, in.GetKey(), time.Duration(in.GetMilliseconds())*time.Millisecond), out)
}

func (r *rpc) ExpireAt(in *redisV1.ExpireAtRequest, out *redisV1.BoolReply) error {
	cl := r.begin("expireat", in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.boolReply(cl.c.ExpireAt(cl.ctx, in.GetKey(), time.Unix(in.GetUnixSeconds(), 0)), out)
}

func (r *rpc) TTL(in *redisV1.TTLRequest, out *redisV1.IntReply) error {
	cl := r.begin("ttl", in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	d, err := cl.c.TTL(cl.ctx, in.GetKey()).Result()
	if err != nil {
		return cl.fail(err)
	}
	out.Value = ttlSeconds(d, time.Second)
	return nil
}

func (r *rpc) PTTL(in *redisV1.PTTLRequest, out *redisV1.IntReply) error {
	cl := r.begin("pttl", in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	d, err := cl.c.PTTL(cl.ctx, in.GetKey()).Result()
	if err != nil {
		return cl.fail(err)
	}
	out.Value = ttlSeconds(d, time.Millisecond)
	return nil
}

// ttlSeconds converts the go-redis duration back to the Redis integer reply.
// go-redis encodes -1 and -2 as -1 and -2 nanoseconds.
func ttlSeconds(d, unit time.Duration) int64 {
	if d < 0 {
		return int64(d)
	}
	return int64(d / unit)
}

func (r *rpc) Persist(in *redisV1.PersistRequest, out *redisV1.BoolReply) error {
	cl := r.begin("persist", in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.boolReply(cl.c.Persist(cl.ctx, in.GetKey()), out)
}

func (r *rpc) Type(in *redisV1.TypeRequest, out *redisV1.StringReply) error {
	cl := r.begin("type", in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.stringReply(cl.c.Type(cl.ctx, in.GetKey()), out)
}

func (r *rpc) Rename(in *redisV1.RenameRequest, out *redisV1.BoolReply) error {
	cl := r.begin("rename", in.GetKey(), in.GetNewKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.statusReply(cl.c.Rename(cl.ctx, in.GetKey(), in.GetNewKey()), out)
}

func (r *rpc) Scan(in *redisV1.ScanRequest, out *redisV1.ScanReply) error {
	cl := r.begin("scan")
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	keys, cursor, err := cl.c.ScanType(cl.ctx, in.GetCursor(), in.GetMatch(), in.GetCount(), in.GetType()).Result()
	if err != nil {
		return cl.fail(err)
	}
	out.Cursor = cursor
	out.Keys = keys
	return nil
}
