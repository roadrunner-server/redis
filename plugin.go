package redis

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/extra/redisprometheus/v9"
	"github.com/roadrunner-server/api/v4/plugins/v1/kv"
	"github.com/roadrunner-server/endure/v2/dep"
	"github.com/roadrunner-server/errors"
	rkv "github.com/roadrunner-server/redis/v5/kv"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
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
	NamedLogger(name string) *zap.Logger
}

type Plugin struct {
	sync.RWMutex
	// config for RR integration
	cfgPlugin Configurer
	// logger
	log *zap.Logger
	// otel tracer
	tracer *sdktrace.TracerProvider
	// prometheus metrics
	metricsCollector *redisprometheus.Collector
}

func (p *Plugin) Init(cfg Configurer, log Logger) error {
	p.log = log.NamedLogger(PluginName)
	p.cfgPlugin = cfg
	p.tracer = sdktrace.NewTracerProvider()

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
func (p *Plugin) KvFromConfig(key string) (kv.Storage, error) {
	const op = errors.Op("redis_plugin_provide")
	st, err := rkv.NewRedisDriver(p.log, key, p.cfgPlugin, p.tracer)
	if err != nil {
		return nil, errors.E(op, err)
	}

	p.metricsCollector = st.MetricsCollector()

	return st, nil
}

func (p *Plugin) MetricsCollector() []prometheus.Collector {
	return []prometheus.Collector{p.metricsCollector}
}
