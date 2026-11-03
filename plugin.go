package redis

import (
	"context"
	"log/slog"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/extra/redisprometheus/v9"
	"github.com/redis/go-redis/v9"
	"github.com/roadrunner-server/api-plugins/v6/kv"
	"github.com/roadrunner-server/endure/v2/dep"
	"github.com/roadrunner-server/errors"
	rkv "github.com/roadrunner-server/redis/v6/kv"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const PluginName = "redis"

type Configurer interface {
	// UnmarshalKey takes a single key and unmarshal it into a Struct.
	UnmarshalKey(name string, out any) error
	// Has checks if config section exists.
	Has(name string) bool
}

type Tracer interface {
	Tracer() *sdktrace.TracerProvider
}

type Logger interface {
	NamedLogger(name string) *slog.Logger
}

type Plugin struct {
	rwm sync.RWMutex
	// config for RR integration
	cfgPlugin Configurer
	// logger
	log *slog.Logger
	// otel tracer
	tracer *sdktrace.TracerProvider
	// prometheus metrics
	metricsCollector *redisprometheus.Collector
	// clients of the KV storages by storage name, used by the RPC service
	clients map[string]redis.UniversalClient
}

func (p *Plugin) Init(cfg Configurer, log Logger) error {
	p.log = log.NamedLogger(PluginName)
	p.cfgPlugin = cfg
	p.tracer = sdktrace.NewTracerProvider()
	p.clients = make(map[string]redis.UniversalClient)

	return nil
}

func (p *Plugin) Name() string {
	return PluginName
}

func (p *Plugin) Collects() []*dep.In {
	return []*dep.In{
		dep.Fits(func(pp any) {
			p.tracer = pp.(Tracer).Tracer()
		}, (*Tracer)(nil)),
	}
}

// KvFromConfig provides KV storage implementation over the redis plugin
func (p *Plugin) KvFromConfig(_ context.Context, key string) (kv.Storage, error) {
	const op = errors.Op("redis_plugin_provide")
	st, err := rkv.NewRedisDriver(p.log, key, p.cfgPlugin, p.tracer)
	if err != nil {
		return nil, errors.E(op, err)
	}

	p.rwm.Lock()
	defer p.rwm.Unlock()

	p.metricsCollector = st.MetricsCollector()
	p.clients[storageName(key)] = st.Client()

	return st, nil
}

// storageName returns the KV storage name for the config key the KV plugin
// passes: `kv.<name>.config` for a local config block, `<name>` for a global
// section.
func storageName(key string) string {
	const prefix, suffix = "kv.", ".config"
	if strings.HasPrefix(key, prefix) && strings.HasSuffix(key, suffix) {
		return key[len(prefix) : len(key)-len(suffix)]
	}

	return key
}

func (p *Plugin) MetricsCollector() []prometheus.Collector {
	p.rwm.RLock()
	defer p.rwm.RUnlock()

	if p.metricsCollector == nil {
		return nil
	}

	return []prometheus.Collector{p.metricsCollector}
}

func (p *Plugin) RPC() any {
	return &rpc{
		pl:     p,
		tracer: p.tracer.Tracer(tracerName),
	}
}
