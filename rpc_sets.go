package redis

import (
	redisV1 "github.com/roadrunner-server/api-go/v6/redis/v1"
)

func (r *rpc) SAdd(in *redisV1.SAddRequest, out *redisV1.IntReply) error {
	cl := r.begin("sadd", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	if err := requireMembers(in.GetMembers()); err != nil {
		return cl.fail(err)
	}
	return cl.intReply(cl.c.SAdd(cl.ctx, in.GetKey(), toArgs(in.GetMembers())...), out)
}

func (r *rpc) SRem(in *redisV1.SRemRequest, out *redisV1.IntReply) error {
	cl := r.begin("srem", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	if err := requireMembers(in.GetMembers()); err != nil {
		return cl.fail(err)
	}
	return cl.intReply(cl.c.SRem(cl.ctx, in.GetKey(), toArgs(in.GetMembers())...), out)
}

func (r *rpc) SMembers(in *redisV1.SMembersRequest, out *redisV1.BytesListReply) error {
	cl := r.begin("smembers", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.bytesListReply(cl.c.SMembers(cl.ctx, in.GetKey()), out)
}

func (r *rpc) SIsMember(in *redisV1.SIsMemberRequest, out *redisV1.BoolReply) error {
	cl := r.begin("sismember", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.boolReply(cl.c.SIsMember(cl.ctx, in.GetKey(), in.GetMember()), out)
}

func (r *rpc) SMIsMember(in *redisV1.SMIsMemberRequest, out *redisV1.BoolListReply) error {
	cl := r.begin("smismember", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	if err := requireMembers(in.GetMembers()); err != nil {
		return cl.fail(err)
	}
	return cl.boolListReply(cl.c.SMIsMember(cl.ctx, in.GetKey(), toArgs(in.GetMembers())...), out)
}

func (r *rpc) SCard(in *redisV1.SCardRequest, out *redisV1.IntReply) error {
	cl := r.begin("scard", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.intReply(cl.c.SCard(cl.ctx, in.GetKey()), out)
}

func (r *rpc) SPop(in *redisV1.SPopRequest, out *redisV1.BytesReply) error {
	cl := r.begin("spop", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.bytesReply(cl.c.SPop(cl.ctx, in.GetKey()), out)
}

func (r *rpc) SPopN(in *redisV1.SPopNRequest, out *redisV1.BytesListReply) error {
	cl := r.begin("spopn", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.bytesListReply(cl.c.SPopN(cl.ctx, in.GetKey(), in.GetCount()), out)
}

func (r *rpc) SRandMember(in *redisV1.SRandMemberRequest, out *redisV1.BytesReply) error {
	cl := r.begin("srandmember", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.bytesReply(cl.c.SRandMember(cl.ctx, in.GetKey()), out)
}

func (r *rpc) SRandMemberN(in *redisV1.SRandMemberNRequest, out *redisV1.BytesListReply) error {
	cl := r.begin("srandmembern", in.GetConnection(), in.GetKey())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	return cl.bytesListReply(cl.c.SRandMemberN(cl.ctx, in.GetKey(), in.GetCount()), out)
}

func (r *rpc) SDiff(in *redisV1.SDiffRequest, out *redisV1.BytesListReply) error {
	cl := r.begin("sdiff", in.GetConnection())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	if err := requireKeys(in.GetKeys()); err != nil {
		return cl.fail(err)
	}
	return cl.bytesListReply(cl.c.SDiff(cl.ctx, in.GetKeys()...), out)
}

func (r *rpc) SInter(in *redisV1.SInterRequest, out *redisV1.BytesListReply) error {
	cl := r.begin("sinter", in.GetConnection())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	if err := requireKeys(in.GetKeys()); err != nil {
		return cl.fail(err)
	}
	return cl.bytesListReply(cl.c.SInter(cl.ctx, in.GetKeys()...), out)
}

func (r *rpc) SUnion(in *redisV1.SUnionRequest, out *redisV1.BytesListReply) error {
	cl := r.begin("sunion", in.GetConnection())
	defer cl.end()
	if cl.err != nil {
		return cl.err
	}
	if err := requireKeys(in.GetKeys()); err != nil {
		return cl.fail(err)
	}
	return cl.bytesListReply(cl.c.SUnion(cl.ctx, in.GetKeys()...), out)
}
