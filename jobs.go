package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/roadrunner-server/api-plugins/v6/jobs"
	rkv "github.com/roadrunner-server/redis/v6/kv"
	"github.com/roadrunner-server/redis/v6/redisjobs"
)

var _ jobs.Constructor = (*Plugin)(nil)

func (p *Plugin) DriverFromConfig(ctx context.Context, key string, q jobs.Queue, pipe jobs.Pipeline) (jobs.Driver, error) {
	cfg := redisjobs.Config{Priority: pipe.Priority()}
	if p.cfgPlugin.Has(key) {
		if err := p.cfgPlugin.UnmarshalKey(key, &cfg); err != nil {
			return nil, err
		}
	}
	return p.jobsDriver(ctx, q, pipe.Name(), cfg)
}

func (p *Plugin) DriverFromPipeline(ctx context.Context, pipe jobs.Pipeline, q jobs.Queue) (jobs.Driver, error) {
	cfg := redisjobs.Config{Queue: pipe.String("queue", pipe.Name()), Priority: pipe.Priority()}
	if pipe.Has("prefetch") {
		cfg.Prefetch = pipe.Int("prefetch", -1)
	}
	if pipe.Has("visibility_timeout") {
		d, err := time.ParseDuration(pipe.String("visibility_timeout", ""))
		if err != nil {
			return nil, fmt.Errorf("redis jobs visibility_timeout: %w", err)
		}
		cfg.VisibilityTimeout = d
	}
	return p.jobsDriver(ctx, q, pipe.Name(), cfg)
}

func (p *Plugin) jobsDriver(ctx context.Context, q jobs.Queue, name string, cfg redisjobs.Config) (jobs.Driver, error) {
	if !p.cfgPlugin.Has(PluginName) {
		return nil, errors.New("redis jobs: missing redis connection settings")
	}
	var conn rkv.Config
	if err := p.cfgPlugin.UnmarshalKey(PluginName, &conn); err != nil {
		return nil, err
	}
	if conn.DB < 0 || (conn.DB != 0 && len(conn.Addrs) > 1 && conn.MasterName == "") {
		return nil, errors.New("redis jobs: db must be nonnegative and Redis Cluster requires database 0")
	}
	if conn.ReadOnly {
		return nil, errors.New("redis jobs requires a writable connection")
	}
	client, err := conn.NewClient(p.log)
	if err != nil {
		return nil, err
	}
	d, err := redisjobs.New(ctx, client, p.log, q, name, cfg)
	if err != nil {
		_ = client.Close()
		return nil, err
	}
	return d, nil
}
