// Package scripts 存放运维脚本的测试：采集吞吐冒烟（纯内存，不连真实 Redis/PG）。
package scripts

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hatedMe/node-logs/internal/collector"
	"github.com/hatedMe/node-logs/internal/store"
)

type loadPublisher struct {
	count atomic.Int64
}

func (p *loadPublisher) Publish(_ context.Context, _ store.RawEvent) (string, error) {
	p.count.Add(1)
	return "stream-id", nil
}

type loadApps struct{}

func (loadApps) ValidateApp(_ context.Context, _ string) error { return nil }

func loadHandler(pub *loadPublisher) *collector.CollectorHandler {
	// 注意：此处构造的采集器不持有任何 *store.Store，只访问内存缓存与发布器，
	// 即"采集接口不直接写 PostgreSQL"的结构性保证。
	return &collector.CollectorHandler{
		Normalizer:     &collector.EventNormalizer{IPSalt: "load-salt"},
		Validator:      collector.Validator{MaxURLLength: 2048},
		Apps:           loadApps{},
		Publisher:      pub,
		PublishTimeout: 2 * time.Second,
	}
}

func loadRequest(t *testing.T, i int) *http.Request {
	t.Helper()
	url := fmt.Sprintf("/pingd/front/web?pvid=800800&uvmark=load-u%d&url=/p%d&event_id=load-e%d&d=%d",
		i%500, i%50, i, time.Now().UnixMilli())
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.RemoteAddr = "198.51.100.9:1234"
	return req
}

// TestCollectorLoadSequential 顺序吞吐冒烟：2000 请求全部 204，只记录吞吐不设阈值断言。
func TestCollectorLoadSequential(t *testing.T) {
	const n = 2000
	pub := &loadPublisher{}
	h := loadHandler(pub)
	start := time.Now()
	for i := 0; i < n; i++ {
		rec := httptest.NewRecorder()
		h.Collect(rec, loadRequest(t, i))
		if rec.Code != http.StatusNoContent {
			t.Fatalf("第 %d 个请求应为 204，实际 %d", i, rec.Code)
		}
	}
	elapsed := time.Since(start)
	if got := pub.count.Load(); got != n {
		t.Fatalf("应发布 %d 个事件，实际 %d 个", n, got)
	}
	t.Logf("顺序吞吐：%d 请求 / %v ≈ %.0f req/s", n, elapsed.Round(time.Millisecond), float64(n)/elapsed.Seconds())
}

// TestCollectorLoadConcurrent 并发吞吐冒烟：8 并发 × 250 请求全部 204。
func TestCollectorLoadConcurrent(t *testing.T) {
	const workers, perWorker = 8, 250
	pub := &loadPublisher{}
	h := loadHandler(pub)
	var wg sync.WaitGroup
	errs := make(chan string, workers*perWorker)
	start := time.Now()
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				rec := httptest.NewRecorder()
				h.Collect(rec, loadRequest(t, w*perWorker+i))
				if rec.Code != http.StatusNoContent {
					errs <- fmt.Sprintf("worker %d 请求 %d 返回 %d", w, i, rec.Code)
					return
				}
			}
		}(w)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Fatal(e)
	}
	elapsed := time.Since(start)
	want := int64(workers * perWorker)
	if got := pub.count.Load(); got != want {
		t.Fatalf("应发布 %d 个事件，实际 %d 个", want, got)
	}
	t.Logf("并发吞吐：%d 请求 / %v ≈ %.0f req/s", want, elapsed.Round(time.Millisecond), float64(want)/elapsed.Seconds())
}
