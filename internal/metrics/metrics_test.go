package metrics

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var errProbe = errors.New("probe boom")

func scrape(t *testing.T, r *Registry) string {
	t.Helper()
	rec := httptest.NewRecorder()
	r.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("/metrics 应返回 200，实际 %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/plain") {
		t.Fatalf("Content-Type 应为 Prometheus 文本格式，实际 %q", ct)
	}
	return rec.Body.String()
}

func TestObserveCollect(t *testing.T) {
	r := New()
	r.ObserveCollect(http.StatusNoContent)
	r.ObserveCollect(http.StatusBadRequest)
	r.ObserveCollect(http.StatusServiceUnavailable)
	body := scrape(t, r)
	for _, want := range []string{
		`collect_requests_total{status="2xx"} 1`,
		`collect_requests_total{status="4xx"} 1`,
		`collect_requests_total{status="5xx"} 1`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("指标缺失 %q\n%s", want, body)
		}
	}
}

func TestObserveConsumeAndDLQ(t *testing.T) {
	r := New()
	r.ObserveConsume(true)
	r.ObserveConsume(false)
	r.ObserveDLQ()
	body := scrape(t, r)
	for _, want := range []string{
		`worker_consume_total{result="ok"} 1`,
		`worker_consume_total{result="fail"} 1`,
		`worker_dlq_total 1`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("指标缺失 %q\n%s", want, body)
		}
	}
}

func TestQueueLenAndDBPool(t *testing.T) {
	r := New()
	r.QueueLen = func(context.Context) (int64, error) { return 42, nil }
	r.PendingCount = func(context.Context) (int64, error) { return 7, nil }
	r.DBStats = func() (int32, int32) { return 3, 10 }
	body := scrape(t, r)
	for _, want := range []string{
		`queue_length 42`,
		`stream_pending_count 7`,
		`db_pool_acquired 3`,
		`db_pool_total 10`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("指标缺失 %q\n%s", want, body)
		}
	}
}

func TestProbeFailuresOmitted(t *testing.T) {
	r := New()
	r.QueueLen = func(context.Context) (int64, error) { return 0, errProbe }
	body := scrape(t, r)
	if strings.Contains(body, "queue_length") {
		t.Errorf("队列探测失败时不应输出 queue_length\n%s", body)
	}
	// 计数器不受影响。
	r.ObserveCollect(http.StatusNoContent)
	if body := scrape(t, r); !strings.Contains(body, `collect_requests_total{status="2xx"} 1`) {
		t.Errorf("计数器应正常输出\n%s", body)
	}
}

func TestObserveConsumeDelay(t *testing.T) {
	r := New()
	r.ObserveConsumeDelay(50 * time.Millisecond)
	r.ObserveConsumeDelay(2 * time.Second)
	r.ObserveConsumeDelay(-time.Second) // 时钟偏斜钳制为 0
	body := scrape(t, r)
	for _, want := range []string{
		`worker_consume_delay_seconds_bucket{le="0.1"} 2`,
		`worker_consume_delay_seconds_bucket{le="1"} 2`,
		`worker_consume_delay_seconds_bucket{le="5"} 3`,
		`worker_consume_delay_seconds_bucket{le="+Inf"} 3`,
		`worker_consume_delay_seconds_count 3`,
		`worker_consume_delay_seconds_sum 2.050000`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("指标缺失 %q\n%s", want, body)
		}
	}
}
