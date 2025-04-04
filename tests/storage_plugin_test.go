package kv

import (
	"github.com/roadrunner-server/metrics/v5"
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

	kvProto "github.com/roadrunner-server/api/v4/build/kv/v1"
	"github.com/roadrunner-server/config/v5"
	"github.com/roadrunner-server/endure/v2"
	goridgeRpc "github.com/roadrunner-server/goridge/v3/pkg/rpc"
	"github.com/roadrunner-server/kv/v5"
	"github.com/roadrunner-server/logger/v5"
	"github.com/roadrunner-server/redis/v5"
	rpcPlugin "github.com/roadrunner-server/rpc/v5"
	"github.com/stretchr/testify/assert"
)

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
	wg.Add(1)

	stopCh := make(chan struct{}, 1)

	go func() {
		defer wg.Done()
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
	}()

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
	wg.Add(1)

	stopCh := make(chan struct{}, 1)

	go func() {
		defer wg.Done()
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
	}()

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
	wg.Add(1)

	stopCh := make(chan struct{}, 1)

	go func() {
		defer wg.Done()
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
	}()

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
	wg.Add(1)

	stopCh := make(chan struct{}, 1)

	go func() {
		defer wg.Done()
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
	}()

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
		conn, err := net.Dial("tcp", addr)
		assert.NoError(t, err)
		client := rpc.NewClientWithCodec(goridgeRpc.NewClientCodec(conn))

		// add 5 second ttl
		tt := time.Now().Add(time.Second * 5).Format(time.RFC3339)
		keys := &kvProto.Request{
			Storage: "redis-rr",
			Items: []*kvProto.Item{
				{
					Key: "a",
				},
				{
					Key: "b",
				},
				{
					Key: "c",
				},
			},
		}

		data := &kvProto.Request{
			Storage: "redis-rr",
			Items: []*kvProto.Item{
				{
					Key:   "a",
					Value: []byte("aa"),
				},
				{
					Key:   "b",
					Value: []byte("bb"),
				},
				{
					Key:     "c",
					Value:   []byte("cc"),
					Timeout: tt,
				},
				{
					Key:   "d",
					Value: []byte("dd"),
				},
				{
					Key:   "e",
					Value: []byte("ee"),
				},
			},
		}

		ret := &kvProto.Response{}
		// Register 3 keys with values
		err = client.Call("kv.Set", data, ret)
		assert.NoError(t, err)

		ret = &kvProto.Response{}
		err = client.Call("kv.Has", keys, ret)
		assert.NoError(t, err)
		assert.Len(t, ret.GetItems(), 3) // should be 3

		// key "c" should be deleted
		time.Sleep(time.Second * 7)

		ret = &kvProto.Response{}
		err = client.Call("kv.Has", keys, ret)
		assert.NoError(t, err)
		assert.Len(t, ret.GetItems(), 2) // should be 2

		ret = &kvProto.Response{}
		err = client.Call("kv.MGet", keys, ret)
		assert.NoError(t, err)
		assert.Len(t, ret.GetItems(), 2) // c is expired

		tt2 := time.Now().Add(time.Second * 10).Format(time.RFC3339)

		data2 := &kvProto.Request{
			Storage: "redis-rr",
			Items: []*kvProto.Item{
				{
					Key:     "a",
					Timeout: tt2,
				},
				{
					Key:     "b",
					Timeout: tt2,
				},
				{
					Key:     "d",
					Timeout: tt2,
				},
			},
		}

		// MEXPIRE
		ret = &kvProto.Response{}
		err = client.Call("kv.MExpire", data2, ret)
		assert.NoError(t, err)

		// TTL
		keys2 := &kvProto.Request{
			Storage: "redis-rr",
			Items: []*kvProto.Item{
				{
					Key: "a",
				},
				{
					Key: "b",
				},
				{
					Key: "d",
				},
			},
		}

		ret = &kvProto.Response{}
		err = client.Call("kv.TTL", keys2, ret)
		assert.NoError(t, err)
		assert.Len(t, ret.GetItems(), 3)

		// HAS AFTER TTL
		time.Sleep(time.Second * 15)
		ret = &kvProto.Response{}
		err = client.Call("kv.Has", keys2, ret)
		assert.NoError(t, err)
		assert.Len(t, ret.GetItems(), 0)

		ret = &kvProto.Response{}
		err = client.Call("kv.TTL", keys2, ret)
		assert.NoError(t, err)
		assert.Len(t, ret.GetItems(), 0)

		// DELETE
		keysDel := &kvProto.Request{
			Storage: "redis-rr",
			Items: []*kvProto.Item{
				{
					Key: "e",
				},
			},
		}

		ret = &kvProto.Response{}
		err = client.Call("kv.Delete", keysDel, ret)
		assert.NoError(t, err)

		// HAS AFTER DELETE
		ret = &kvProto.Response{}
		err = client.Call("kv.Has", keysDel, ret)
		assert.NoError(t, err)
		assert.Len(t, ret.GetItems(), 0)

		dataClear := &kvProto.Request{
			Storage: "redis-rr",
			Items: []*kvProto.Item{
				{
					Key:   "a",
					Value: []byte("aa"),
				},
				{
					Key:   "b",
					Value: []byte("bb"),
				},
				{
					Key:   "c",
					Value: []byte("cc"),
				},
				{
					Key:   "d",
					Value: []byte("dd"),
				},
				{
					Key:   "e",
					Value: []byte("ee"),
				},
			},
		}

		clr := &kvProto.Request{Storage: "redis-rr"}

		ret = &kvProto.Response{}
		// Register 3 keys with values
		err = client.Call("kv.Set", dataClear, ret)
		assert.NoError(t, err)

		ret = &kvProto.Response{}
		err = client.Call("kv.Has", dataClear, ret)
		assert.NoError(t, err)
		assert.Len(t, ret.GetItems(), 5) // should be 5

		ret = &kvProto.Response{}
		err = client.Call("kv.Clear", clr, ret)
		assert.NoError(t, err)

		ret = &kvProto.Response{}
		err = client.Call("kv.Has", dataClear, ret)
		assert.NoError(t, err)
		assert.Len(t, ret.GetItems(), 0) // should be 5
	}
}

func get(address string) (string, error) {
	r, err := http.Get(address) //nolint:gosec
	if err != nil {
		return "", err
	}

	b, err := io.ReadAll(r.Body)
	if err != nil {
		return "", err
	}

	err = r.Body.Close()
	if err != nil {
		return "", err
	}
	// unsafe
	return string(b), err
}
