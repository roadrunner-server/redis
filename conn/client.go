package conn

import (
	"log/slog"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"github.com/roadrunner-server/errors"
)

// NewClient applies the defaults, builds the universal client and attaches
// the redisotel metrics and tracing hooks.
func NewClient(cfg *Config, log *slog.Logger) (redis.UniversalClient, error) {
	const op = errors.Op("redis_new_client")

	cfg.InitDefaults()

	tlsCfg, err := tlsConfig(cfg.TLSConfig)
	if err != nil {
		return nil, errors.E(op, err)
	}

	client := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:            cfg.Addrs,
		DB:               cfg.DB,
		Username:         cfg.Username,
		Password:         cfg.Password,
		SentinelUsername: cfg.SentinelUsername,
		SentinelPassword: cfg.SentinelPassword,
		MaxRetries:       cfg.MaxRetries,
		MinRetryBackoff:  cfg.MinRetryBackoff,
		MaxRetryBackoff:  cfg.MaxRetryBackoff,
		DialTimeout:      cfg.DialTimeout,
		ReadTimeout:      cfg.ReadTimeout,
		WriteTimeout:     cfg.WriteTimeout,
		PoolSize:         cfg.PoolSize,
		MinIdleConns:     cfg.MinIdleConns,
		ConnMaxLifetime:  cfg.MaxConnAge,
		PoolTimeout:      cfg.PoolTimeout,
		ConnMaxIdleTime:  cfg.IdleTimeout,
		ReadOnly:         cfg.ReadOnly,
		RouteByLatency:   cfg.RouteByLatency,
		RouteRandomly:    cfg.RouteRandomly,
		MasterName:       cfg.MasterName,
		TLSConfig:        tlsCfg,
	})

	if err := redisotel.InstrumentMetrics(client); err != nil {
		log.Warn("failed to instrument redis metrics, client will work without metrics", "error", err)
	}

	if err := redisotel.InstrumentTracing(client); err != nil {
		log.Warn("failed to instrument redis tracing, client will work without tracing", "error", err)
	}

	return client, nil
}
