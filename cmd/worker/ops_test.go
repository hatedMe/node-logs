package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/hatedMe/node-logs/internal/health"
	"github.com/hatedMe/node-logs/internal/metrics"
)

func testRedis(t *testing.T) (*redis.Client, func()) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("启动 miniredis 失败: %v", err)
	}
	c := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return c, func() {
		_ = c.Close()
		mr.Close()
	}
}

func TestStreamProbes(t *testing.T) {
	c, cleanup := testRedis(t)
	defer cleanup()
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		if err := c.XAdd(ctx, &redis.XAddArgs{
			Stream: "events:test", Values: map[string]any{"event_id": "e"},
		}).Err(); err != nil {
			t.Fatalf("XAdd 失败: %v", err)
		}
	}
	if err := c.XGroupCreateMkStream(ctx, "events:test", "g", "0").Err(); err != nil {
		t.Fatalf("建组失败: %v", err)
	}
	// 读 1 条不确认：队列 3，未确认 1。
	if err := c.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group: "g", Consumer: "c1", Streams: []string{"events:test", ">"}, Count: 1,
	}).Err(); err != nil {
		t.Fatalf("XReadGroup 失败: %v", err)
	}

	qlen, err := queueLenProbe(c, "events:test")(ctx)
	if err != nil {
		t.Fatalf("队列长度探测失败: %v", err)
	}
	if qlen != 3 {
		t.Errorf("队列长度应为 3，实际 %d", qlen)
	}
	pending, err := pendingProbe(c, "events:test", "g")(ctx)
	if err != nil {
		t.Fatalf("pending 探测失败: %v", err)
	}
	if pending != 1 {
		t.Errorf("pending 应为 1，实际 %d", pending)
	}
}

type opsFakePinger struct{ err error }

func (p opsFakePinger) Ping(_ context.Context) error { return p.err }

func TestWorkerMux(t *testing.T) {
	reg := metrics.New()
	reg.QueueLen = func(context.Context) (int64, error) { return 5, nil }
	reg.PendingCount = func(context.Context) (int64, error) { return 2, nil }
	h := &health.Handler{PG: opsFakePinger{}, Redis: opsFakePinger{}, Timeout: time.Second}
	mux := newWorkerMux(reg, h)

	for path, want := range map[string]int{"/metrics": 200, "/healthz": 200, "/readyz": 200} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != want {
			t.Errorf("GET %s 应为 %d，实际 %d", path, want, rec.Code)
		}
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	for _, want := range []string{"queue_length 5", "stream_pending_count 2", "worker_consume_total"} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("/metrics 缺失 %q\n%s", want, rec.Body.String())
		}
	}
}
