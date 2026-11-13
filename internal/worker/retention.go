package worker

import (
	"context"
	"log/slog"
	"time"
)

// PurgingStore 是留存清理依赖的存储子集；*store.Store 已实现。
type PurgingStore interface {
	PurgeRawEvents(ctx context.Context, retentionDays int) (int64, error)
}

// RetentionJanitor 按 RETENTION_DAYS 周期清理过期原始事件。
// 多 worker 部署下重复执行是幂等的（DELETE 条件相同）。
type RetentionJanitor struct {
	Store PurgingStore
	// RetentionDays 为保留天数；<=0 时禁用清理。
	RetentionDays int
	// Interval 为清理周期；<=0 时默认为 24 小时。
	Interval time.Duration
	// Logger 为空时使用 slog.Default。
	Logger *slog.Logger
}

func (j *RetentionJanitor) interval() time.Duration {
	if j.Interval <= 0 {
		return 24 * time.Hour
	}
	return j.Interval
}

func (j *RetentionJanitor) logger() *slog.Logger {
	if j.Logger != nil {
		return j.Logger
	}
	return slog.Default()
}

// Run 启动即清理一次，之后按周期清理，直到 ctx 取消。
// 单次失败只记录日志，不影响后续周期。
func (j *RetentionJanitor) Run(ctx context.Context) {
	log := j.logger()
	if j.RetentionDays <= 0 {
		log.Info("留存清理已禁用（RETENTION_DAYS<=0）")
		return
	}
	j.purge(ctx)
	t := time.NewTicker(j.interval())
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			j.purge(ctx)
		}
	}
}

func (j *RetentionJanitor) purge(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	deleted, err := j.Store.PurgeRawEvents(ctx, j.RetentionDays)
	if err != nil {
		j.logger().Error("留存清理失败", "err", err, "retention_days", j.RetentionDays)
		return
	}
	j.logger().Info("留存清理完成", "deleted", deleted, "retention_days", j.RetentionDays)
}
