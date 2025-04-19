package kv

import (
	"context"
	stderr "errors"
	"strings"
	"time"
	"unsafe"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/extra/redisprometheus/v9"
	"github.com/redis/go-redis/v9"
	"github.com/roadrunner-server/api/v4/plugins/v1/kv"
	"github.com/roadrunner-server/errors"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
)

const (
	tracerName = "redis"
)

type Configurer interface {
	// UnmarshalKey takes a single key and unmarshal it into a Struct.
	UnmarshalKey(name string, out any) error
	// Has checks if config section exists.
	Has(name string) bool
}

type Driver struct {
	universalClient  redis.UniversalClient
	tracer           *sdktrace.TracerProvider
	log              *zap.Logger
	cfg              *Config
	metricsCollector *redisprometheus.Collector
}

func NewRedisDriver(log *zap.Logger, key string, cfgPlugin Configurer, tracer *sdktrace.TracerProvider) (*Driver, error) {
	const op = errors.Op("new_redis_driver")

	d := &Driver{
		log:    log,
		tracer: tracer,
	}

	// will be different for every connected Driver
	err := cfgPlugin.UnmarshalKey(key, &d.cfg)
	if err != nil {
		return nil, errors.E(op, err)
	}

	if d.cfg == nil {
		return nil, errors.E(op, errors.Errorf("config not found by provided key: %s", key))
	}

	d.cfg.InitDefaults()

	tlsConfig, err := tlsConfig(d.cfg.TLSConfig)
	if err != nil {
		return nil, errors.E(op, err)
	}

	d.universalClient = redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:            d.cfg.Addrs,
		DB:               d.cfg.DB,
		Username:         d.cfg.Username,
		Password:         d.cfg.Password,
		SentinelPassword: d.cfg.SentinelPassword,
		MaxRetries:       d.cfg.MaxRetries,
		MinRetryBackoff:  d.cfg.MaxRetryBackoff,
		MaxRetryBackoff:  d.cfg.MaxRetryBackoff,
		DialTimeout:      d.cfg.DialTimeout,
		ReadTimeout:      d.cfg.ReadTimeout,
		WriteTimeout:     d.cfg.WriteTimeout,
		PoolSize:         d.cfg.PoolSize,
		MinIdleConns:     d.cfg.MinIdleConns,
		ConnMaxLifetime:  d.cfg.MaxConnAge,
		PoolTimeout:      d.cfg.PoolTimeout,
		ConnMaxIdleTime:  d.cfg.IdleTimeout,
		ReadOnly:         d.cfg.ReadOnly,
		RouteByLatency:   d.cfg.RouteByLatency,
		RouteRandomly:    d.cfg.RouteRandomly,
		MasterName:       d.cfg.MasterName,
		TLSConfig:        tlsConfig,
	})

	err = redisotel.InstrumentMetrics(d.universalClient)
	if err != nil {
		d.log.Warn("failed to instrument redis metrics, driver will work without metrics", zap.Error(err))
	}

	err = redisotel.InstrumentTracing(d.universalClient)
	if err != nil {
		d.log.Warn("failed to instrument redis tracing, driver will work without tracing", zap.Error(err))
	}

	d.metricsCollector = redisprometheus.NewCollector("rr", "redis", d.universalClient)

	return d, nil
}

// Has checks if value exists.
func (d *Driver) Has(keys ...string) (map[string]bool, error) {
	const op = errors.Op("redis_driver_has")
	ctx, span := d.tracer.Tracer(tracerName).Start(context.Background(), "redis:has")
	defer span.End()

	if keys == nil {
		span.RecordError(errors.E(op, errors.Str("no keys")))
		return nil, errors.E(op, errors.NoKeys)
	}

	m := make(map[string]bool, len(keys))
	for _, key := range keys {
		keyTrimmed := strings.TrimSpace(key)
		if keyTrimmed == "" {
			span.RecordError(errors.E(op, errors.Str("empty key")))
			return nil, errors.E(op, errors.EmptyKey)
		}

		exist, err := d.universalClient.Exists(ctx, key).Result()
		if err != nil {
			return nil, err
		}
		if exist == 1 {
			m[key] = true
		}
	}

	return m, nil
}

// Get loads key content into slice.
func (d *Driver) Get(key string) ([]byte, error) {
	const op = errors.Op("redis_driver_get")
	ctx, span := d.tracer.Tracer(tracerName).Start(context.Background(), "redis:get")
	defer span.End()

	// to get cases like "  "
	keyTrimmed := strings.TrimSpace(key)
	if keyTrimmed == "" {
		span.RecordError(errors.E(op, errors.EmptyKey))
		return nil, errors.E(op, errors.EmptyKey)
	}

	return d.universalClient.Get(ctx, key).Bytes()
}

// MGet loads content of multiple values (some values might be skipped).
// https://redis.io/commands/mget
// Returns slice with the interfaces with values
func (d *Driver) MGet(keys ...string) (map[string][]byte, error) {
	const op = errors.Op("redis_driver_mget")
	ctx, span := d.tracer.Tracer(tracerName).Start(context.Background(), "redis:mget")
	defer span.End()
	if keys == nil {
		span.RecordError(errors.E(op, errors.NoKeys))
		return nil, errors.E(op, errors.NoKeys)
	}

	// should not be empty keys
	for _, key := range keys {
		keyTrimmed := strings.TrimSpace(key)
		if keyTrimmed == "" {
			span.RecordError(errors.E(op, errors.EmptyKey))
			return nil, errors.E(op, errors.EmptyKey)
		}
	}

	m := make(map[string][]byte, len(keys))

	for _, k := range keys {
		cmd := d.universalClient.Get(ctx, k)
		if cmd.Err() != nil {
			if stderr.Is(cmd.Err(), redis.Nil) {
				continue
			}

			span.RecordError(cmd.Err())
			return nil, errors.E(op, cmd.Err())
		}

		m[k] = strToBytes(cmd.Val())
	}

	return m, nil
}

// Set sets value with the TTL in seconds
// https://redis.io/commands/set
// Redis `SET key value [expiration]` command.
//
// Use expiration for `SETEX`-like behavior.
// Zero expiration means the key has no expiration time.
func (d *Driver) Set(items ...kv.Item) error {
	const op = errors.Op("redis_driver_set")
	ctx, span := d.tracer.Tracer(tracerName).Start(context.Background(), "redis:set")
	defer span.End()

	if items == nil {
		span.RecordError(errors.E(op, errors.NoKeys))
		return errors.E(op, errors.NoKeys)
	}
	now := time.Now()
	for _, item := range items {
		if item == nil {
			span.RecordError(errors.E(op, errors.EmptyKey))
			return errors.E(op, errors.EmptyKey)
		}

		if item.Timeout() == "" {
			err := d.universalClient.Set(ctx, item.Key(), item.Value(), 0).Err()
			if err != nil {
				span.RecordError(err)
				return err
			}
		} else {
			t, err := time.Parse(time.RFC3339, item.Timeout())
			if err != nil {
				span.RecordError(err)
				return err
			}
			err = d.universalClient.Set(ctx, item.Key(), item.Value(), t.Sub(now)).Err()
			if err != nil {
				span.RecordError(err)
				return err
			}
		}
	}

	return nil
}

// Delete one or multiple keys.
func (d *Driver) Delete(keys ...string) error {
	const op = errors.Op("redis_driver_delete")
	ctx, span := d.tracer.Tracer(tracerName).Start(context.Background(), "redis:delete")
	defer span.End()

	if keys == nil {
		span.RecordError(errors.E(op, errors.NoKeys))
		return errors.E(op, errors.NoKeys)
	}

	// should not be empty keys
	for _, key := range keys {
		keyTrimmed := strings.TrimSpace(key)
		if keyTrimmed == "" {
			span.RecordError(errors.E(op, errors.EmptyKey))
			return errors.E(op, errors.EmptyKey)
		}
	}

	return d.universalClient.Del(ctx, keys...).Err()
}

// MExpire https://redis.io/commands/expire
// timeout in RFC3339
func (d *Driver) MExpire(items ...kv.Item) error {
	const op = errors.Op("redis_driver_mexpire")
	ctx, span := d.tracer.Tracer(tracerName).Start(context.Background(), "redis:mexpire")
	defer span.End()

	now := time.Now()
	for _, item := range items {
		if item == nil {
			continue
		}
		if item.Timeout() == "" || strings.TrimSpace(item.Key()) == "" {
			span.RecordError(errors.Str("should set timeout and at least one key"))
			return errors.E(op, errors.Str("should set timeout and at least one key"))
		}

		t, err := time.Parse(time.RFC3339, item.Timeout())
		if err != nil {
			span.RecordError(err)
			return err
		}

		// t guessed to be in future
		// for Redis we use t.Sub, it will result in seconds, like 4.2s
		d.universalClient.Expire(ctx, item.Key(), t.Sub(now))
	}

	return nil
}

// TTL https://redis.io/commands/ttl
// return time in seconds (float64) for a given keys
func (d *Driver) TTL(keys ...string) (map[string]string, error) {
	const op = errors.Op("redis_driver_ttl")
	ctx, span := d.tracer.Tracer(tracerName).Start(context.Background(), "redis:ttl")
	defer span.End()

	if keys == nil {
		span.RecordError(errors.E(op, errors.NoKeys))
		return nil, errors.E(op, errors.NoKeys)
	}

	// should not be empty keys
	for _, key := range keys {
		keyTrimmed := strings.TrimSpace(key)
		if keyTrimmed == "" {
			span.RecordError(errors.E(op, errors.EmptyKey))
			return nil, errors.E(op, errors.EmptyKey)
		}
	}

	m := make(map[string]string, len(keys))

	for _, key := range keys {
		duration, err := d.universalClient.TTL(ctx, key).Result()
		if err != nil {
			span.RecordError(err)
			return nil, err
		}

		// The command returns -2 if the key does not exist.
		// The command returns -1 if the key exists but has no associated expire.
		if duration == -1 || duration == -2 {
			continue
		}

		m[key] = time.Now().Add(duration).Format(time.RFC3339)
	}

	return m, nil
}

func (d *Driver) Clear() error {
	ctx, span := d.tracer.Tracer(tracerName).Start(context.Background(), "redis:clear")
	defer span.End()

	fdb := d.universalClient.FlushDB(ctx)
	if fdb.Err() != nil {
		span.RecordError(fdb.Err())
		return fdb.Err()
	}

	return nil
}

func (d *Driver) Stop() {
	// close the connection
	_ = d.universalClient.Close()
}

func (d *Driver) MetricsCollector() *redisprometheus.Collector {
	return d.metricsCollector
}

func strToBytes(data string) []byte {
	if data == "" {
		return nil
	}

	return unsafe.Slice(unsafe.StringData(data), len(data))
}
