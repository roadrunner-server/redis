package kv

import (
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// Tests that clear databases or change server settings use a private Unix socket.
func privateRedis(t *testing.T) *redis.Client {
	t.Helper()
	dir := t.TempDir()
	socket := filepath.Join(dir, "r.sock")
	cmd := exec.CommandContext(t.Context(), "redis-server", "--port", "0", "--unixsocket", socket,
		"--dir", dir, "--save", "", "--appendonly", "no")
	require.NoError(t, cmd.Start())
	t.Cleanup(func() { _ = cmd.Wait() })
	c := redis.NewClient(&redis.Options{Addr: socket, MaxRetries: -1})
	t.Cleanup(func() { _ = c.Close() })
	require.Eventually(t, func() bool { return c.Ping(t.Context()).Err() == nil }, 5*time.Second, 10*time.Millisecond)
	return c
}
