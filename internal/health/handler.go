// Package health 提供存活与就绪探测。
package health

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/hatedMe/node-logs/internal/httpx"
)

// Pinger 是依赖探测接口；*store.Store 与 *queue.StreamProducer 均已实现。
type Pinger interface {
	Ping(ctx context.Context) error
}

// Handler 持有各依赖的探测器。
type Handler struct {
	PG    Pinger
	Redis Pinger
	// Timeout 是单次就绪探测超时；<=0 时默认为 2 秒。
	Timeout time.Duration
}

func (h *Handler) timeout() time.Duration {
	if h.Timeout <= 0 {
		return 2 * time.Second
	}
	return h.Timeout
}

// Live 存活探测：进程能响应即 200，不检查依赖。
func (h *Handler) Live(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteSuccess(w, map[string]string{"status": "ok"})
}

// Ready 就绪探测：PostgreSQL 与 Redis 均可用才 200，否则 503。
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.timeout())
	defer cancel()

	type result struct {
		name string
		err  error
	}
	results := make(chan result, 2)
	var wg sync.WaitGroup
	for name, p := range map[string]Pinger{"postgres": h.PG, "redis": h.Redis} {
		wg.Go(func() {
			if p == nil {
				results <- result{name: name, err: fmt.Errorf("%s 探测器未配置", name)}
				return
			}
			results <- result{name: name, err: p.Ping(ctx)}
		})
	}
	wg.Wait()
	close(results)

	var failed []string
	for res := range results {
		if res.err != nil {
			failed = append(failed, fmt.Sprintf("%s=%v", res.name, res.err))
		}
	}
	if len(failed) > 0 {
		httpx.WriteError(w, http.StatusServiceUnavailable, fmt.Sprintf("not ready: %v", failed))
		return
	}
	httpx.WriteSuccess(w, map[string]string{"status": "ok"})
}
