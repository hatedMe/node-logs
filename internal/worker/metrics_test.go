package worker

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hatedMe/node-logs/internal/metrics"
)

func scrapeWorker(t *testing.T, r *metrics.Registry) string {
	t.Helper()
	rec := httptest.NewRecorder()
	r.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	return rec.Body.String()
}

func TestDeadLetterObserved(t *testing.T) {
	reg := metrics.New()
	dlq := &DeadLetter{
		Client: &fakeDLQClient{}, Stream: "events:dlq", SourceStream: "events:test",
		Metrics: reg,
	}
	if err := dlq.WriteRaw(context.Background(),
		map[string]any{"event_id": "e1"}, "1-0", "测试"); err != nil {
		t.Fatalf("WriteRaw 失败: %v", err)
	}
	if body := scrapeWorker(t, reg); !strings.Contains(body, "worker_dlq_total 1") {
		t.Errorf("死信计数缺失\n%s", body)
	}
}

func TestConsumerConsumeObserved(t *testing.T) {
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)

	regOK := metrics.New()
	fs := newFakeStream()
	fs.Add(streamValues(t, batchEvent("e1", "800800", "v1", "/a", at)))
	cOK, _ := testConsumer(fs, &stubProcessor{})
	cOK.Metrics = regOK
	if err := cOK.Tick(context.Background()); err != nil {
		t.Fatalf("成功 Tick 失败: %v", err)
	}
	if body := scrapeWorker(t, regOK); !strings.Contains(body, `worker_consume_total{result="ok"} 1`) {
		t.Errorf("消费成功计数缺失\n%s", body)
	}

	regFail := metrics.New()
	fs2 := newFakeStream()
	fs2.Add(streamValues(t, batchEvent("e2", "800800", "v1", "/b", at)))
	cFail, _ := testConsumer(fs2, &stubProcessor{err: errors.New("pg down"), failN: 1})
	cFail.Metrics = regFail
	if err := cFail.Tick(context.Background()); err == nil {
		t.Fatal("处理失败时 Tick 应返回错误，实际为 nil")
	}
	if body := scrapeWorker(t, regFail); !strings.Contains(body, `worker_consume_total{result="fail"} 1`) {
		t.Errorf("消费失败计数缺失\n%s", body)
	}
}

func TestConsumerEmptyTickNotObserved(t *testing.T) {
	reg := metrics.New()
	fs := newFakeStream()
	c, _ := testConsumer(fs, &stubProcessor{})
	c.Metrics = reg
	if err := c.Tick(context.Background()); err != nil {
		t.Fatalf("空 Tick 失败: %v", err)
	}
	if body := scrapeWorker(t, reg); strings.Contains(body, `worker_consume_total{result="ok"} 1`) {
		t.Errorf("空轮询不应计为消费\n%s", body)
	}
}

func TestConsumerDelayObserved(t *testing.T) {
	now := time.Date(2026, 11, 12, 10, 0, 5, 0, time.UTC)
	reg := metrics.New()
	fs := newFakeStream()
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	e1 := batchEvent("e1", "800800", "v1", "/a", at)
	e1.ReceivedAt = now.Add(-1500 * time.Millisecond)
	e2 := batchEvent("e2", "800800", "v1", "/b", at)
	e2.ReceivedAt = time.Time{} // 缺失接收时间：跳过
	e3 := batchEvent("e3", "800800", "v1", "/c", at)
	e3.ReceivedAt = now.Add(time.Second) // 未来时间：钳制为 0
	fs.Add(streamValues(t, e1))
	fs.Add(streamValues(t, e2))
	fs.Add(streamValues(t, e3))
	c, _ := testConsumer(fs, &stubProcessor{})
	c.Metrics = reg
	c.Now = func() time.Time { return now }
	if err := c.Tick(context.Background()); err != nil {
		t.Fatalf("Tick 失败: %v", err)
	}
	body := scrapeWorker(t, reg)
	for _, want := range []string{
		`worker_consume_delay_seconds_bucket{le="1"} 1`,
		`worker_consume_delay_seconds_bucket{le="5"} 2`,
		`worker_consume_delay_seconds_bucket{le="+Inf"} 2`,
		`worker_consume_delay_seconds_count 2`,
		`worker_consume_delay_seconds_sum 1.500000`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("延迟指标缺失 %q\n%s", want, body)
		}
	}
}
