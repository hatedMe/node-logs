package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type fakePinger struct {
	err   error
	delay time.Duration
	calls int
}

func (f *fakePinger) Ping(ctx context.Context) error {
	f.calls++
	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return f.err
}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var v map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("响应不是合法 JSON: %v", err)
	}
	return v
}

func TestLiveAlwaysOK(t *testing.T) {
	h := &Handler{PG: &fakePinger{err: errors.New("pg down")}, Redis: &fakePinger{err: errors.New("redis down")}}
	rec := httptest.NewRecorder()
	h.Live(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("/healthz 应恒为 200，实际 %d", rec.Code)
	}
}

func TestReadyOK(t *testing.T) {
	pg, redis := &fakePinger{}, &fakePinger{}
	h := &Handler{PG: pg, Redis: redis, Timeout: time.Second}
	rec := httptest.NewRecorder()
	h.Ready(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("依赖正常时 /readyz 应为 200，实际 %d：%s", rec.Code, rec.Body.String())
	}
	if pg.calls != 1 || redis.calls != 1 {
		t.Fatalf("应各探测一次，实际 pg=%d redis=%d", pg.calls, redis.calls)
	}
}

func TestReadyDegraded(t *testing.T) {
	h := &Handler{PG: &fakePinger{err: errors.New("pg down")}, Redis: &fakePinger{}, Timeout: time.Second}
	rec := httptest.NewRecorder()
	h.Ready(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("依赖故障时 /readyz 应为 503，实际 %d", rec.Code)
	}
	body := decodeBody(t, rec)
	if body["code"] != float64(503) {
		t.Errorf("包络 code 应为 503，实际 %v", body)
	}
}

func TestReadyTimeout(t *testing.T) {
	h := &Handler{
		PG:      &fakePinger{},
		Redis:   &fakePinger{delay: 5 * time.Second},
		Timeout: 50 * time.Millisecond,
	}
	start := time.Now()
	rec := httptest.NewRecorder()
	h.Ready(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Fatalf("就绪探测应快速超时，实际耗时 %v", elapsed)
	}
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("探测超时时 /readyz 应为 503，实际 %d", rec.Code)
	}
}
