package redis

import (
	"bytes"
	"context"
	stderr "errors"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
	redisV1 "github.com/roadrunner-server/api-go/v6/redis/v1"
	"github.com/roadrunner-server/errors"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "redis"

var (
	errEmptyConnection  = stderr.New("no connection provided")
	errNoSuchConnection = stderr.New("no such connection")
)

type rpc struct {
	pl     *Plugin
	tracer trace.Tracer
}

// call carries the span, the resolved client and the error of one RPC call.
type call struct {
	ctx  context.Context
	span trace.Span
	op   errors.Op
	c    redis.UniversalClient
	err  error
}

// begin opens the span, validates the keys and resolves the connection.
// The caller must defer end() and return err when it is set.
func (r *rpc) begin(command, connection string, keys ...string) *call {
	ctx, span := r.tracer.Start(context.Background(), "redis:"+command)
	cl := &call{ctx: ctx, span: span, op: errors.Op("redis_rpc_" + command)}

	for _, k := range keys {
		if err := requireKey(k); err != nil {
			cl.err = cl.fail(err)
			return cl
		}
	}

	if connection == "" {
		cl.err = cl.fail(errEmptyConnection)
		return cl
	}

	c, ok := r.pl.connections[connection]
	if !ok {
		cl.err = cl.fail(fmt.Errorf("%w: %s", errNoSuchConnection, connection))
		return cl
	}

	cl.c = c
	return cl
}

func (c *call) end() { c.span.End() }

func (c *call) fail(err error) error {
	c.span.RecordError(err)
	return errors.E(c.op, err)
}

// requireKeys rejects an empty key list and empty keys.
func requireKeys(keys []string) error {
	if len(keys) == 0 {
		return errors.E(errors.NoKeys)
	}
	for _, k := range keys {
		if err := requireKey(k); err != nil {
			return err
		}
	}
	return nil
}

// requireKey rejects an empty key.
func requireKey(key string) error {
	if strings.TrimSpace(key) == "" {
		return errors.E(errors.EmptyKey)
	}
	return nil
}

// requireMembers rejects an empty member list and empty members.
func requireMembers(members [][]byte) error {
	if len(members) == 0 {
		return errors.E(errors.NoKeys)
	}
	for _, m := range members {
		if err := requireMember(m); err != nil {
			return err
		}
	}
	return nil
}

// requireMember rejects an empty member.
func requireMember(member []byte) error {
	if len(bytes.TrimSpace(member)) == 0 {
		return errors.E(errors.EmptyKey)
	}
	return nil
}

func toArgs(members [][]byte) []any {
	out := make([]any, 0, len(members))
	for _, m := range members {
		out = append(out, m)
	}
	return out
}

func toBytes(values []string) [][]byte {
	out := make([][]byte, 0, len(values))
	for _, v := range values {
		out = append(out, []byte(v))
	}
	return out
}

// Reply mapping. Every helper records the error on the span and wraps it.

func (c *call) intReply(cmd *redis.IntCmd, out *redisV1.IntReply) error {
	v, err := cmd.Result()
	if err != nil {
		return c.fail(err)
	}
	out.Value = v
	return nil
}

func (c *call) floatReply(cmd *redis.FloatCmd, out *redisV1.FloatReply) error {
	v, err := cmd.Result()
	if err != nil {
		return c.fail(err)
	}
	out.Value = v
	return nil
}

func (c *call) boolReply(cmd *redis.BoolCmd, out *redisV1.BoolReply) error {
	v, err := cmd.Result()
	if err != nil {
		return c.fail(err)
	}
	out.Value = v
	return nil
}

// statusReply maps an OK status to true.
func (c *call) statusReply(cmd *redis.StatusCmd, out *redisV1.BoolReply) error {
	if err := cmd.Err(); err != nil {
		return c.fail(err)
	}
	out.Value = true
	return nil
}

func (c *call) stringReply(cmd *redis.StatusCmd, out *redisV1.StringReply) error {
	v, err := cmd.Result()
	if err != nil {
		return c.fail(err)
	}
	out.Value = v
	return nil
}

// bytesReply maps a nil reply to an absent value.
func (c *call) bytesReply(cmd *redis.StringCmd, out *redisV1.BytesReply) error {
	v, err := cmd.Bytes()
	if stderr.Is(err, redis.Nil) {
		out.Value = nil
		return nil
	}
	if err != nil {
		return c.fail(err)
	}
	if v == nil {
		v = []byte{}
	}
	out.Value = v
	return nil
}

func (c *call) bytesListReply(cmd *redis.StringSliceCmd, out *redisV1.BytesListReply) error {
	v, err := cmd.Result()
	if stderr.Is(err, redis.Nil) {
		out.Values = nil
		return nil
	}
	if err != nil {
		return c.fail(err)
	}
	out.Values = toBytes(v)
	return nil
}

// sliceReply maps MGET and HMGET replies: a nil entry sets found[i] to false.
func (c *call) sliceReply(cmd *redis.SliceCmd, out *redisV1.BytesListReply) error {
	v, err := cmd.Result()
	if err != nil {
		return c.fail(err)
	}
	out.Values = make([][]byte, len(v))
	out.Found = make([]bool, len(v))
	for i, e := range v {
		s, ok := e.(string)
		if !ok {
			continue
		}
		out.Values[i] = []byte(s)
		out.Found[i] = true
	}
	return nil
}

func (c *call) boolListReply(cmd *redis.BoolSliceCmd, out *redisV1.BoolListReply) error {
	v, err := cmd.Result()
	if err != nil {
		return c.fail(err)
	}
	out.Values = v
	return nil
}

func (c *call) mapReply(cmd *redis.MapStringStringCmd, out *redisV1.MapReply) error {
	v, err := cmd.Result()
	if err != nil {
		return c.fail(err)
	}
	out.Values = make(map[string][]byte, len(v))
	for k, s := range v {
		out.Values[k] = []byte(s)
	}
	return nil
}

func (c *call) scoredReply(cmd *redis.ZSliceCmd, out *redisV1.ScoredMembersReply) error {
	v, err := cmd.Result()
	if err != nil {
		return c.fail(err)
	}
	out.Members = make([]*redisV1.ScoredMember, 0, len(v))
	for _, z := range v {
		m, _ := z.Member.(string)
		out.Members = append(out.Members, &redisV1.ScoredMember{Member: []byte(m), Score: z.Score})
	}
	return nil
}

func (c *call) optIntReply(cmd *redis.IntCmd, out *redisV1.OptionalIntReply) error {
	v, err := cmd.Result()
	if stderr.Is(err, redis.Nil) {
		out.Value = nil
		return nil
	}
	if err != nil {
		return c.fail(err)
	}
	out.Value = &v
	return nil
}

func (c *call) optFloatReply(cmd *redis.FloatCmd, out *redisV1.OptionalFloatReply) error {
	v, err := cmd.Result()
	if stderr.Is(err, redis.Nil) {
		out.Value = nil
		return nil
	}
	if err != nil {
		return c.fail(err)
	}
	out.Value = &v
	return nil
}
