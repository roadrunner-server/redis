package redis

import (
	"io"
	"log/slog"
	"testing"

	"github.com/roadrunner-server/errors"
	"github.com/roadrunner-server/redis/v6/conn"
	"github.com/stretchr/testify/require"
)

// stubConfigurer hands the plugin pre-built configs instead of decoding YAML.
type stubConfigurer struct {
	sections map[string]bool
	cfg      *conn.Config
	conns    map[string]*conn.Config
	err      error
}

func (s *stubConfigurer) Has(name string) bool { return s.sections[name] }

func (s *stubConfigurer) UnmarshalKey(_ string, out any) error {
	if s.err != nil {
		return s.err
	}

	switch p := out.(type) {
	case **conn.Config:
		*p = s.cfg
	case *map[string]*conn.Config:
		*p = s.conns
	default:
		return errors.Str("unexpected target type")
	}
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
		cfg:      &conn.Config{Addrs: []string{"127.0.0.1:16379"}},
	}
	p := newPlugin(t, c)

	st, err := p.KvFromConfig(t.Context(), "kv.redis-rr")

	require.NoError(t, err)
	require.NotNil(t, st)
	require.Len(t, p.MetricsCollector(), 1, "the driver's collector should be exposed once a storage exists")
}

func TestInitOpensOneClientPerConnection(t *testing.T) {
	cases := []struct {
		name      string
		conns     map[string]*conn.Config
		wantNames []string
	}{
		{
			name:      "two connections",
			conns:     map[string]*conn.Config{"default": {Addrs: []string{"127.0.0.1:16379"}}, "second": {Addrs: []string{"127.0.0.1:16379"}, DB: 1}},
			wantNames: []string{"default", "second"},
		},
		{
			name:      "null connection body gets defaults",
			conns:     map[string]*conn.Config{"default": nil},
			wantNames: []string{"default"},
		},
		{
			name:      "empty section",
			conns:     map[string]*conn.Config{},
			wantNames: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := newPlugin(t, &stubConfigurer{sections: map[string]bool{PluginName: true}, conns: tc.conns})
			t.Cleanup(func() { require.NoError(t, p.Stop(t.Context())) })

			require.Len(t, p.connections, len(tc.wantNames))
			for _, n := range tc.wantNames {
				require.Contains(t, p.connections, n)
			}
			require.Len(t, p.MetricsCollector(), len(tc.wantNames))
		})
	}
}

func TestInitWithoutSectionKeepsNoConnections(t *testing.T) {
	p := newPlugin(t, &stubConfigurer{})

	require.Empty(t, p.connections)
	require.Nil(t, p.MetricsCollector())
}

func TestInitFailsOnUnreadableRootCA(t *testing.T) {
	c := &stubConfigurer{
		sections: map[string]bool{PluginName: true},
		conns:    map[string]*conn.Config{"default": {TLSConfig: &conn.TLSConfig{RootCa: "/nonexistent/ca.crt"}}},
	}

	err := (&Plugin{}).Init(c, discardLogger{})

	require.ErrorContains(t, err, "no such file")
}
