// Package metrics 提供零依赖的 Prometheus 文本格式指标。
package metrics

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"
)

// consumeDelayBuckets 是消费延迟直方图的桶边界（秒），另有隐式 +Inf 桶。
var consumeDelayBuckets = []float64{0.1, 0.5, 1, 5, 15, 60, 300}

// Registry 持有采集与消费计数器，以及队列/连接池的即时探测函数。
type Registry struct {
	collect2xx atomic.Int64
	collect4xx atomic.Int64
	collect5xx atomic.Int64

	consumeOK   atomic.Int64
	consumeFail atomic.Int64
	dlqTotal    atomic.Int64

	// delayCounts 是累计桶计数，末位为 +Inf；delaySumMicros 为延迟总和（微秒）。
	delayCounts    [8]atomic.Int64
	delaySumMicros atomic.Int64
	delayCount     atomic.Int64

	// QueueLen 抓取时查询队列长度；为 nil 或返回错误时不输出该指标。
	QueueLen func(ctx context.Context) (int64, error)
	// PendingCount 抓取时查询消费组未确认（pending）消息数，即消费积压；
	// 为 nil 或返回错误时不输出该指标。
	PendingCount func(ctx context.Context) (int64, error)
	// DBStats 返回连接池已用/总数；为 nil 时不输出。
	DBStats func() (acquired, total int32)
	// ProbeTimeout 是单次抓取中队列探测的超时；<=0 时默认为 2 秒。
	ProbeTimeout time.Duration
}

// New 创建空注册表。
func New() *Registry { return &Registry{} }

// ObserveCollect 按状态码档记录一次采集请求。
func (r *Registry) ObserveCollect(status int) {
	switch {
	case status >= 200 && status < 300:
		r.collect2xx.Add(1)
	case status >= 400 && status < 500:
		r.collect4xx.Add(1)
	case status >= 500:
		r.collect5xx.Add(1)
	}
}

// ObserveConsume 记录一次消费批次结果。
func (r *Registry) ObserveConsume(ok bool) {
	if ok {
		r.consumeOK.Add(1)
	} else {
		r.consumeFail.Add(1)
	}
}

// ObserveDLQ 记录一次死信写入。
func (r *Registry) ObserveDLQ() { r.dlqTotal.Add(1) }

// ObserveConsumeDelay 记录一次消费延迟（处理开始时刻 - 事件接收时刻）。
// 负值（server/worker 时钟偏斜）钳制为 0。
func (r *Registry) ObserveConsumeDelay(d time.Duration) {
	if d < 0 {
		d = 0
	}
	micros := int64(d / time.Microsecond)
	secs := float64(micros) / 1e6
	idx := len(consumeDelayBuckets)
	for i, b := range consumeDelayBuckets {
		if secs <= b {
			idx = i
			break
		}
	}
	for i := idx; i <= len(consumeDelayBuckets); i++ {
		r.delayCounts[i].Add(1)
	}
	r.delaySumMicros.Add(micros)
	r.delayCount.Add(1)
}

func (r *Registry) probeTimeout() time.Duration {
	if r.ProbeTimeout <= 0 {
		return 2 * time.Second
	}
	return r.ProbeTimeout
}

// Handler 返回 /metrics 抓取处理器。
func (r *Registry) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		fmt.Fprintf(w, "collect_requests_total{status=\"2xx\"} %d\n", r.collect2xx.Load())
		fmt.Fprintf(w, "collect_requests_total{status=\"4xx\"} %d\n", r.collect4xx.Load())
		fmt.Fprintf(w, "collect_requests_total{status=\"5xx\"} %d\n", r.collect5xx.Load())
		fmt.Fprintf(w, "worker_consume_total{result=\"ok\"} %d\n", r.consumeOK.Load())
		fmt.Fprintf(w, "worker_consume_total{result=\"fail\"} %d\n", r.consumeFail.Load())
		fmt.Fprintf(w, "worker_dlq_total %d\n", r.dlqTotal.Load())
		for i, b := range consumeDelayBuckets {
			fmt.Fprintf(w, "worker_consume_delay_seconds_bucket{le=%q} %d\n",
				strconv.FormatFloat(b, 'f', -1, 64), r.delayCounts[i].Load())
		}
		fmt.Fprintf(w, "worker_consume_delay_seconds_bucket{le=\"+Inf\"} %d\n",
			r.delayCounts[len(consumeDelayBuckets)].Load())
		fmt.Fprintf(w, "worker_consume_delay_seconds_sum %s\n",
			strconv.FormatFloat(float64(r.delaySumMicros.Load())/1e6, 'f', 6, 64))
		fmt.Fprintf(w, "worker_consume_delay_seconds_count %d\n", r.delayCount.Load())
		if r.QueueLen != nil {
			ctx, cancel := context.WithTimeout(context.Background(), r.probeTimeout())
			n, err := r.QueueLen(ctx)
			cancel()
			if err == nil {
				fmt.Fprintf(w, "queue_length %d\n", n)
			}
		}
		if r.PendingCount != nil {
			ctx, cancel := context.WithTimeout(context.Background(), r.probeTimeout())
			n, err := r.PendingCount(ctx)
			cancel()
			if err == nil {
				fmt.Fprintf(w, "stream_pending_count %d\n", n)
			}
		}
		if r.DBStats != nil {
			acquired, total := r.DBStats()
			fmt.Fprintf(w, "db_pool_acquired %d\n", acquired)
			fmt.Fprintf(w, "db_pool_total %d\n", total)
		}
	})
}
