package redis

import (
	"context"
	"log/slog"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/extra/redisprometheus/v9"
	"github.com/redis/go-redis/v9"
	"github.com/roadrunner-server/api-plugins/v6/kv"
	"github.com/roadrunner-server/endure/v2/dep"
	"github.com/roadrunner-server/errors"
	"github.com/roadrunner-server/redis/v6/conn"
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
	// prometheus metrics of the KV driver
	metricsCollector *redisprometheus.Collector
	// named clients from the redis config section
	connections map[string]redis.UniversalClient
	// one prometheus collector per named connection
	collectors []prometheus.Collector
}

func (p *Plugin) Init(cfg Configurer, log Logger) error {
	const op = errors.Op("redis_plugin_init")

	p.log = log.NamedLogger(PluginName)
	p.cfgPlugin = cfg
	p.tracer = sdktrace.NewTracerProvider()
	p.connections = make(map[string]redis.UniversalClient)

	if !cfg.Has(PluginName) {
		return nil
	}

	var conns map[string]*conn.Config
	if err := cfg.UnmarshalKey(PluginName, &conns); err != nil {
		return errors.E(op, err)
	}

	for name, c := range conns {
		if c == nil {
			c = &conn.Config{}
		}

		client, err := conn.NewClient(c, p.log)
		if err != nil {
			return errors.E(op, err)
		}

		p.connections[name] = client
		p.collectors = append(p.collectors, redisprometheus.NewCollector("rr", "redis_"+name, client))
	}

	return nil
}

func (p *Plugin) Serve() chan error {
	return make(chan error, 1)
}

func (p *Plugin) Stop(context.Context) error {
	for _, c := range p.connections {
		_ = c.Close()
	}

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

	// should not be needed, but to double check
	p.rwm.Lock()
	defer p.rwm.Unlock()

	p.metricsCollector = st.MetricsCollector()

	return st, nil
}

func (p *Plugin) MetricsCollector() []prometheus.Collector {
	p.rwm.RLock()
	defer p.rwm.RUnlock()

	out := make([]prometheus.Collector, 0, len(p.collectors)+1)
	out = append(out, p.collectors...)
	if p.metricsCollector != nil {
		out = append(out, p.metricsCollector)
	}

	if len(out) == 0 {
		return nil
	}

	return out
}

func (p *Plugin) RPC() any {
	return &rpc{
		pl:     p,
		tracer: p.tracer.Tracer(tracerName),
	}
}
