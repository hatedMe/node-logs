package main

import (
	"context"
	"net/http"

	"github.com/redis/go-redis/v9"

	"github.com/hatedMe/node-logs/internal/health"
	"github.com/hatedMe/node-logs/internal/metrics"
)

// streamProber 是指标探测依赖的 Redis 子集；*redis.Client 已实现。
type streamProber interface {
	XLen(ctx context.Context, stream string) *redis.IntCmd
	XPending(ctx context.Context, stream, group string) *redis.XPendingCmd
	Ping(ctx context.Context) *redis.StatusCmd
}

var _ streamProber = (*redis.Client)(nil)

// queueLenProbe 返回查询主 Stream 长度的探测函数。
func queueLenProbe(c streamProber, stream string) func(context.Context) (int64, error) {
	return func(ctx context.Context) (int64, error) {
		return c.XLen(ctx, stream).Result()
	}
}

// pendingProbe 返回查询消费组未确认消息数的探测函数。
func pendingProbe(c streamProber, stream, group string) func(context.Context) (int64, error) {
	return func(ctx context.Context) (int64, error) {
		res, err := c.XPending(ctx, stream, group).Result()
		if err != nil {
			return 0, err
		}
		return res.Count, nil
	}
}

// redisPinger 适配 *redis.Client 到 health.Pinger。
type redisPinger struct{ c streamProber }

func (p redisPinger) Ping(ctx context.Context) error { return p.c.Ping(ctx).Err() }

// newWorkerMux 组装 worker 运维端点：/metrics、/healthz、/readyz。
func newWorkerMux(reg *metrics.Registry, h *health.Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/metrics", reg.Handler())
	mux.HandleFunc("/healthz", h.Live)
	mux.HandleFunc("/readyz", h.Ready)
	return mux
}
