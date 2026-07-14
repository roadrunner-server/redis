package kv

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"testing"
	"time"

	kvProto "github.com/roadrunner-server/api-go/v6/kv/v2"
	"github.com/roadrunner-server/config/v6"
	"github.com/roadrunner-server/endure/v2"
	goridgeRpc "github.com/roadrunner-server/goridge/v4/pkg/rpc"
	"github.com/roadrunner-server/kv/v6"
	"github.com/roadrunner-server/logger/v6"
	"github.com/roadrunner-server/metrics/v6"
	"github.com/roadrunner-server/redis/v6"
	rpcPlugin "github.com/roadrunner-server/rpc/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
)

func newKVClient(t *testing.T, address string) *rpc.Client {
	t.Helper()
	conn, err := (&net.Dialer{}).DialContext(t.Context(), "tcp", address)
	require.NoError(t, err)
	client := rpc.NewClientWithCodec(goridgeRpc.NewClientCodec(conn))
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestRedis(t *testing.T) {
	cont := endure.New(slog.LevelDebug)

	cfg := &config.Plugin{
		Version: "2024.2.0",
		Path:    "configs/.rr-redis.yaml",
	}

	err := cont.RegisterAll(
		cfg,
		&kv.Plugin{},
		&redis.Plugin{},
		&rpcPlugin.Plugin{},
		&logger.Plugin{},
	)
	assert.NoError(t, err)

	err = cont.Init()
	if err != nil {
		t.Fatal(err)
	}

	ch, err := cont.Serve()
	assert.NoError(t, err)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	wg := &sync.WaitGroup{}

	stopCh := make(chan struct{}, 1)

	wg.Go(func() {
		for {
			select {
			case e := <-ch:
				assert.Fail(t, "error", e.Error.Error())
				err = cont.Stop()
				if err != nil {
					assert.FailNow(t, "error", err.Error())
				}
			case <-sig:
				err = cont.Stop()
				if err != nil {
					assert.FailNow(t, "error", err.Error())
				}
				return
			case <-stopCh:
				// timeout
				err = cont.Stop()
				if err != nil {
					assert.FailNow(t, "error", err.Error())
				}
				return
			}
		}
	})

	time.Sleep(time.Second * 1)
	t.Run("REDIS", testRPCMethodsRedis("127.0.0.1:6001"))
	stopCh <- struct{}{}
	wg.Wait()
}

func TestRedisGlobalSection(t *testing.T) {
	cont := endure.New(slog.LevelDebug)

	cfg := &config.Plugin{
		Version: "2024.2.0",
		Path:    "configs/.rr-redis-global.yaml",
	}

	err := cont.RegisterAll(
		cfg,
		&kv.Plugin{},
		&redis.Plugin{},
		&rpcPlugin.Plugin{},
		&logger.Plugin{},
	)
	assert.NoError(t, err)

	err = cont.Init()
	if err != nil {
		t.Fatal(err)
	}

	ch, err := cont.Serve()
	assert.NoError(t, err)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	wg := &sync.WaitGroup{}

	stopCh := make(chan struct{}, 1)

	wg.Go(func() {
		for {
			select {
			case e := <-ch:
				assert.Fail(t, "error", e.Error.Error())
				err = cont.Stop()
				if err != nil {
					assert.FailNow(t, "error", err.Error())
				}
			case <-sig:
				err = cont.Stop()
				if err != nil {
					assert.FailNow(t, "error", err.Error())
				}
				return
			case <-stopCh:
				// timeout
				err = cont.Stop()
				if err != nil {
					assert.FailNow(t, "error", err.Error())
				}
				return
			}
		}
	})

	time.Sleep(time.Second * 1)
	t.Run("REDIS", testRPCMethodsRedis("127.0.0.1:6001"))
	stopCh <- struct{}{}
	wg.Wait()
}

func TestRedisNoConfig(t *testing.T) {
	cont := endure.New(slog.LevelDebug)

	cfg := &config.Plugin{
		Version: "2024.2.0",
		Path:    "configs/.rr-redis-no-config.yaml", // should be used default
	}

	err := cont.RegisterAll(
		cfg,
		&logger.Plugin{},
		&kv.Plugin{},
		&redis.Plugin{},
		&rpcPlugin.Plugin{},
	)
	assert.NoError(t, err)

	err = cont.Init()
	if err != nil {
		t.Fatal(err)
	}

	_, err = cont.Serve()
	assert.Error(t, err)
	_ = cont.Stop()
}

func TestRedisTLS(t *testing.T) {
	cont := endure.New(slog.LevelDebug)

	cfg := &config.Plugin{
		Version: "2024.2.0",
		Path:    "configs/.rr-redis-tls.yaml",
	}

	err := cont.RegisterAll(
		cfg,
		&kv.Plugin{},
		&redis.Plugin{},
		&rpcPlugin.Plugin{},
		&logger.Plugin{},
	)
	assert.NoError(t, err)

	err = cont.Init()
	if err != nil {
		t.Fatal(err)
	}

	ch, err := cont.Serve()
	assert.NoError(t, err)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	wg := &sync.WaitGroup{}

	stopCh := make(chan struct{}, 1)

	wg.Go(func() {
		for {
			select {
			case e := <-ch:
				assert.Fail(t, "error", e.Error.Error())
				err = cont.Stop()
				if err != nil {
					assert.FailNow(t, "error", err.Error())
				}
			case <-sig:
				err = cont.Stop()
				if err != nil {
					assert.FailNow(t, "error", err.Error())
				}
				return
			case <-stopCh:
				// timeout
				err = cont.Stop()
				if err != nil {
					assert.FailNow(t, "error", err.Error())
				}
				return
			}
		}
	})

	time.Sleep(time.Second * 1)
	t.Run("REDIS-TLS", testRPCMethodsRedis("127.0.0.1:6002"))
	stopCh <- struct{}{}
	wg.Wait()
}

func TestRedisMetrics(t *testing.T) {
	cont := endure.New(slog.LevelDebug)

	cfg := &config.Plugin{
		Version: "2024.2.0",
		Path:    "configs/.rr-redis-metrics.yaml",
	}

	err := cont.RegisterAll(
		cfg,
		&kv.Plugin{},
		&redis.Plugin{},
		&rpcPlugin.Plugin{},
		&logger.Plugin{},
		&metrics.Plugin{},
	)
	assert.NoError(t, err)

	err = cont.Init()
	if err != nil {
		t.Fatal(err)
	}

	ch, err := cont.Serve()
	assert.NoError(t, err)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	wg := &sync.WaitGroup{}

	stopCh := make(chan struct{}, 1)

	wg.Go(func() {
		for {
			select {
			case e := <-ch:
				assert.Fail(t, "error", e.Error.Error())
				err = cont.Stop()
				if err != nil {
					assert.FailNow(t, "error", err.Error())
				}
			case <-sig:
				err = cont.Stop()
				if err != nil {
					assert.FailNow(t, "error", err.Error())
				}
				return
			case <-stopCh:
				// timeout
				err = cont.Stop()
				if err != nil {
					assert.FailNow(t, "error", err.Error())
				}
				return
			}
		}
	})

	time.Sleep(time.Second * 1)
	t.Run("REDIS", testRPCMethodsRedis("127.0.0.1:6001"))

	time.Sleep(time.Second * 2)
	out, err := get("http://[::1]:2112/metrics")
	assert.NoError(t, err)

	assert.Contains(t, out, "rr_redis_pool_conn_idle_current")
	assert.Contains(t, out, "rr_redis_pool_conn_stale_total")
	assert.Contains(t, out, "rr_redis_pool_conn_total_current")
	assert.Contains(t, out, "rr_redis_pool_hit_total")
	assert.Contains(t, out, "rr_redis_pool_miss_total")
	assert.Contains(t, out, "rr_redis_pool_timeout_total")

	stopCh <- struct{}{}
	wg.Wait()
}

func testRPCMethodsRedis(addr string) func(t *testing.T) {
	return func(t *testing.T) {
		const storage = "redis-rr"

		client := newKVClient(t, addr)

		tt := durationpb.New(time.Second * 5)
		keys := &kvProto.KvRequest{
			Storage: storage,
			Items: []*kvProto.KvItem{
				{Key: "a"},
				{Key: "b"},
				{Key: "c"},
			},
		}

		data := &kvProto.KvRequest{
			Storage: storage,
			Items: []*kvProto.KvItem{
				{Key: "a", Value: []byte("aa")},
				{Key: "b", Value: []byte("bb")},
				{Key: "c", Value: []byte("cc"), Ttl: tt},
				{Key: "d", Value: []byte("dd")},
				{Key: "e", Value: []byte("ee")},
			},
		}

		var resp kvProto.KvResponse
		err := client.Call("kv.Set", data, &resp)
		assert.NoError(t, err)

		resp.Reset()
		err = client.Call("kv.Has", keys, &resp)
		assert.NoError(t, err)
		assert.Len(t, resp.GetItems(), 3)

		// key "c" should be deleted
		time.Sleep(time.Second * 7)

		resp.Reset()
		err = client.Call("kv.Has", keys, &resp)
		assert.NoError(t, err)
		assert.Len(t, resp.GetItems(), 2)

		resp.Reset()
		err = client.Call("kv.MGet", keys, &resp)
		assert.NoError(t, err)
		assert.Len(t, resp.GetItems(), 2) // c is expired

		tt2 := durationpb.New(time.Second * 10)

		data2 := &kvProto.KvRequest{
			Storage: storage,
			Items: []*kvProto.KvItem{
				{Key: "a", Ttl: tt2},
				{Key: "b", Ttl: tt2},
				{Key: "d", Ttl: tt2},
			},
		}

		resp.Reset()
		err = client.Call("kv.MExpire", data2, &resp)
		assert.NoError(t, err)

		keys2 := &kvProto.KvRequest{
			Storage: storage,
			Items: []*kvProto.KvItem{
				{Key: "a"},
				{Key: "b"},
				{Key: "d"},
			},
		}

		resp.Reset()
		err = client.Call("kv.TTL", keys2, &resp)
		assert.NoError(t, err)
		assert.Len(t, resp.GetItems(), 3)

		// HAS AFTER TTL
		time.Sleep(time.Second * 15)
		resp.Reset()
		err = client.Call("kv.Has", keys2, &resp)
		assert.NoError(t, err)
		assert.Empty(t, resp.GetItems())

		resp.Reset()
		err = client.Call("kv.TTL", keys2, &resp)
		assert.NoError(t, err)
		assert.Empty(t, resp.GetItems())

		keysDel := &kvProto.KvRequest{
			Storage: storage,
			Items:   []*kvProto.KvItem{{Key: "e"}},
		}

		resp.Reset()
		err = client.Call("kv.Delete", keysDel, &resp)
		assert.NoError(t, err)

		resp.Reset()
		err = client.Call("kv.Has", keysDel, &resp)
		assert.NoError(t, err)
		assert.Empty(t, resp.GetItems())

		dataClear := &kvProto.KvRequest{
			Storage: storage,
			Items: []*kvProto.KvItem{
				{Key: "a", Value: []byte("aa")},
				{Key: "b", Value: []byte("bb")},
				{Key: "c", Value: []byte("cc")},
				{Key: "d", Value: []byte("dd")},
				{Key: "e", Value: []byte("ee")},
			},
		}

		resp.Reset()
		err = client.Call("kv.Set", dataClear, &resp)
		assert.NoError(t, err)

		resp.Reset()
		err = client.Call("kv.Has", dataClear, &resp)
		assert.NoError(t, err)
		assert.Len(t, resp.GetItems(), 5)

		resp.Reset()
		err = client.Call("kv.Clear", &kvProto.KvRequest{Storage: storage}, &resp)
		assert.NoError(t, err)

		resp.Reset()
		err = client.Call("kv.Has", dataClear, &resp)
		assert.NoError(t, err)
		assert.Empty(t, resp.GetItems())
	}
}

func get(address string) (string, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, address, nil)
	if err != nil {
		return "", err
	}

	r, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	b, err := io.ReadAll(r.Body)
	if err != nil {
		return "", err
	}

	if err := r.Body.Close(); err != nil {
		return "", err
	}
	return string(b), nil
}
