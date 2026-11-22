package redisjobs_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/roadrunner-server/api-plugins/v6/jobs"
	priorityqueue "github.com/roadrunner-server/priority_queue"
	"github.com/roadrunner-server/redis/v6/redisjobs"
	"github.com/stretchr/testify/require"
)

type olderServer struct{ redis.UniversalClient }

func (olderServer) InfoMap(ctx context.Context, sections ...string) *redis.InfoCmd {
	cmd := redis.NewInfoCmd(ctx, "info", sections)
	cmd.SetVal(map[string]map[string]string{"Server": {"redis_version": "7.4.0"}})
	return cmd
}

func TestRejectUnsupportedPipeline(t *testing.T) {
	cases := []struct {
		name string
		cfg  redisjobs.Config
		want string
	}{
		{name: "unsupported Redis version", want: `requires Redis 8.0 or later, got "7.4.0"`},
		{name: "ambiguous cluster hash tag", cfg: redisjobs.Config{Queue: "mail{west}"}, want: "braces"},
		{name: "negative prefetch", cfg: redisjobs.Config{Prefetch: -1}, want: "prefetch"},
		{name: "submillisecond visibility timeout", cfg: redisjobs.Config{VisibilityTimeout: time.Microsecond}, want: "visibility_timeout"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := redisjobs.New(t.Context(), olderServer{}, slog.Default(), priorityqueue.NewBinHeap[jobs.Job](1), "mail", tc.cfg)
			require.ErrorContains(t, err, tc.want)
		})
	}
}
