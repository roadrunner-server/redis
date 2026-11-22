package kv

import (
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

type Config struct {
	Addrs            []string      `mapstructure:"addrs"`
	DB               int           `mapstructure:"db"`
	Username         string        `mapstructure:"username"`
	Password         string        `mapstructure:"password"`
	MasterName       string        `mapstructure:"master_name"`
	SentinelUsername string        `mapstructure:"sentinel_username"`
	SentinelPassword string        `mapstructure:"sentinel_password"`
	RouteByLatency   bool          `mapstructure:"route_by_latency"`
	RouteRandomly    bool          `mapstructure:"route_randomly"`
	MaxRetries       int           `mapstructure:"max_retries"`
	DialTimeout      time.Duration `mapstructure:"dial_timeout"`
	MinRetryBackoff  time.Duration `mapstructure:"min_retry_backoff"`
	MaxRetryBackoff  time.Duration `mapstructure:"max_retry_backoff"`
	PoolSize         int           `mapstructure:"pool_size"`
	MinIdleConns     int           `mapstructure:"min_idle_conns"`
	MaxConnAge       time.Duration `mapstructure:"max_conn_age"`
	ReadTimeout      time.Duration `mapstructure:"read_timeout"`
	WriteTimeout     time.Duration `mapstructure:"write_timeout"`
	PoolTimeout      time.Duration `mapstructure:"pool_timeout"`
	IdleTimeout      time.Duration `mapstructure:"idle_timeout"`
	IdleCheckFreq    time.Duration `mapstructure:"idle_check_freq"`
	ReadOnly         bool          `mapstructure:"read_only"`
	TLSConfig        *TLSConfig    `mapstructure:"tls"`
}

// InitDefaults initializing fill config with default values
func (s *Config) InitDefaults() {
	if s.Addrs == nil {
		s.Addrs = []string{"127.0.0.1:6379"} // default addr is pointing to local storage
	}
}

// Client owns a Redis connection pool and its metrics registration.
type Client struct {
	redis.UniversalClient
	close func() error
}

func (c *Client) Close() error { return c.close() }

// NewClient opens a client for one KV storage or Jobs pipeline.
func (s *Config) NewClient(log *slog.Logger) (*Client, error) {
	s.InitDefaults()
	tls, err := tlsConfig(s.TLSConfig)
	if err != nil {
		return nil, err
	}
	client := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs: s.Addrs, DB: s.DB, Username: s.Username, Password: s.Password,
		SentinelUsername: s.SentinelUsername, SentinelPassword: s.SentinelPassword, MasterName: s.MasterName,
		MaxRetries: s.MaxRetries, MinRetryBackoff: s.MinRetryBackoff, MaxRetryBackoff: s.MaxRetryBackoff,
		DialTimeout: s.DialTimeout, ReadTimeout: s.ReadTimeout, WriteTimeout: s.WriteTimeout,
		PoolSize: s.PoolSize, MinIdleConns: s.MinIdleConns, ConnMaxLifetime: s.MaxConnAge,
		PoolTimeout: s.PoolTimeout, ConnMaxIdleTime: s.IdleTimeout, ReadOnly: s.ReadOnly,
		RouteByLatency: s.RouteByLatency, RouteRandomly: s.RouteRandomly, TLSConfig: tls,
	})
	done := make(chan struct{})
	if err := redisotel.InstrumentMetrics(client, redisotel.WithCloseChan(done)); err != nil {
		log.Warn("failed to instrument redis metrics, driver will work without metrics", "error", err)
	}
	if err := redisotel.InstrumentTracing(client); err != nil {
		log.Warn("failed to instrument redis tracing, driver will work without tracing", "error", err)
	}
	return &Client{UniversalClient: client, close: sync.OnceValue(func() error {
		close(done)
		return client.Close()
	})}, nil
}
