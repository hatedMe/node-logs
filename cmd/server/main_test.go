package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hatedMe/node-logs/internal/collector"
	"github.com/hatedMe/node-logs/internal/health"
	"github.com/hatedMe/node-logs/internal/metrics"
	"github.com/hatedMe/node-logs/internal/store"
)

type opsFakePublisher struct{}

func (opsFakePublisher) Publish(_ context.Context, _ store.RawEvent) (string, error) {
	return "stream-id-1", nil
}

type opsFakeApps struct{}

func (opsFakeApps) ValidateApp(_ context.Context, _ string) error { return nil }

type opsFakePinger struct{ err error }

func (p opsFakePinger) Ping(_ context.Context) error { return p.err }

func testOpsRouter() (*metrics.Registry, *health.Handler, http.Handler) {
	collect := &collector.CollectorHandler{
		Normalizer:     &collector.EventNormalizer{IPSalt: "test-salt"},
		Validator:      collector.Validator{MaxURLLength: 2048},
		Apps:           opsFakeApps{},
		Publisher:      opsFakePublisher{},
		PublishTimeout: 2 * time.Second,
	}
	reg := metrics.New()
	h := &health.Handler{PG: opsFakePinger{}, Redis: opsFakePinger{}}
	// 应用/报表处理器在路由注册时不调用，传 nil 即可。
	return reg, h, newRouter(collect, nil, nil, nil, "test-secret", h, reg)
}

func TestOpsRoutes(t *testing.T) {
	_, _, r := testOpsRouter()
	for path, want := range map[string]int{
		"/healthz": 200,
		"/readyz":  200,
		"/metrics": 200,
	} {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != want {
			t.Errorf("GET %s 应为 %d，实际 %d：%s", path, want, rec.Code, rec.Body.String())
		}
	}
}

func TestCollectCountedInMetrics(t *testing.T) {
	_, _, r := testOpsRouter()
	url := fmt.Sprintf("/pingd/front/web?pvid=800800&uvmark=u1&url=/home&d=%d", time.Now().UnixMilli())
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("采集应返回 204，实际 %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if !strings.Contains(rec.Body.String(), `collect_requests_total{status="2xx"} 1`) {
		t.Errorf("采集计数缺失\n%s", rec.Body.String())
	}
}

func TestReadyzDegraded503(t *testing.T) {
	collect := &collector.CollectorHandler{
		Normalizer: &collector.EventNormalizer{IPSalt: "test-salt"},
		Validator:  collector.Validator{MaxURLLength: 2048},
		Apps:       opsFakeApps{},
		Publisher:  opsFakePublisher{},
	}
	h := &health.Handler{PG: opsFakePinger{err: fmt.Errorf("pg down")}, Redis: opsFakePinger{}}
	r := newRouter(collect, nil, nil, nil, "test-secret", h, metrics.New())
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("/readyz 依赖故障时应为 503，实际 %d", rec.Code)
	}
}
