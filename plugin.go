package redis

import (
	"sync"

	"github.com/roadrunner-server/api/v3/plugins/v1/kv"
	"github.com/roadrunner-server/errors"
	rkv "github.com/roadrunner-server/redis/v3/kv"
	"go.uber.org/zap"
)

const PluginName = "redis"

type Configurer interface {
	// UnmarshalKey takes a single key and unmarshal it into a Struct.
	UnmarshalKey(name string, out any) error
	// Has checks if config section exists.
	Has(name string) bool
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
}

func (p *Plugin) Init(cfg Configurer, log Logger) error {
	p.log = log.NamedLogger(PluginName)
	p.cfgPlugin = cfg

	return nil
}

func (p *Plugin) Name() string {
	return PluginName
}

// KvFromConfig provides KV storage implementation over the redis plugin
func (p *Plugin) KvFromConfig(key string) (kv.Storage, error) {
	const op = errors.Op("redis_plugin_provide")
	st, err := rkv.NewRedisDriver(p.log, key, p.cfgPlugin)
	if err != nil {
		return nil, errors.E(op, err)
	}

	return st, nil
}
