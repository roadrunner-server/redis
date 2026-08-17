package redis

import (
	"io"
	"log/slog"
	"testing"

	"github.com/roadrunner-server/errors"
	rkv "github.com/roadrunner-server/redis/v6/kv"
	"github.com/stretchr/testify/require"
)

// stubConfigurer hands the driver a pre-built config instead of decoding YAML.
type stubConfigurer struct {
	sections map[string]bool
	cfg      *rkv.Config
	err      error
}

func (s *stubConfigurer) Has(name string) bool { return s.sections[name] }

func (s *stubConfigurer) UnmarshalKey(_ string, out any) error {
	if s.err != nil {
		return s.err
	}

	p, ok := out.(**rkv.Config)
	if !ok {
		return errors.Str("unexpected target type")
	}
	*p = s.cfg
	return nil
}

type discardLogger struct{}

func (discardLogger) NamedLogger(string) *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newPlugin(t *testing.T, c *stubConfigurer) *Plugin {
	t.Helper()

	p := &Plugin{}
	require.NoError(t, p.Init(c, discardLogger{}))

	return p
}

func TestInitBuildsTracerAndLogger(t *testing.T) {
	p := newPlugin(t, &stubConfigurer{})

	require.NotNil(t, p.log)
	require.NotNil(t, p.tracer)
	require.NotNil(t, p.cfgPlugin)
}

func TestName(t *testing.T) {
	require.Equal(t, PluginName, (&Plugin{}).Name())
}

func TestCollectsDeclaresTracerDependency(t *testing.T) {
	require.Len(t, (&Plugin{}).Collects(), 1)
}

// TestMetricsCollectorEmptyBeforeAnyStorage covers the nil guard: asking for
// collectors before a storage exists must yield nothing rather than a slice
// holding a nil collector, which would panic the prometheus registry.
func TestMetricsCollectorEmptyBeforeAnyStorage(t *testing.T) {
	require.Nil(t, newPlugin(t, &stubConfigurer{}).MetricsCollector())
}

func TestKvFromConfigPropagatesDecodeError(t *testing.T) {
	p := newPlugin(t, &stubConfigurer{err: errors.Str("broken config")})

	_, err := p.KvFromConfig(t.Context(), "kv.broken")

	require.ErrorContains(t, err, "broken config")
}

func TestKvFromConfigBuildsDriverAndRegistersCollector(t *testing.T) {
	c := &stubConfigurer{
		sections: map[string]bool{"kv.redis-rr": true},
		cfg:      &rkv.Config{Addrs: []string{"127.0.0.1:16379"}},
	}
	p := newPlugin(t, c)

	st, err := p.KvFromConfig(t.Context(), "kv.redis-rr")

	require.NoError(t, err)
	require.NotNil(t, st)
	require.Len(t, p.MetricsCollector(), 1, "the driver's collector should be exposed once a storage exists")
}
