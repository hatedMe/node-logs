// Command worker 启动 Go 队列消费与聚合进程。
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/hatedMe/node-logs/internal/aggregation"
	"github.com/hatedMe/node-logs/internal/config"
	"github.com/hatedMe/node-logs/internal/health"
	"github.com/hatedMe/node-logs/internal/logging"
	"github.com/hatedMe/node-logs/internal/metrics"
	"github.com/hatedMe/node-logs/internal/store"
	"github.com/hatedMe/node-logs/internal/worker"
)

func main() {
	if err := run(); err != nil {
		slog.Error("worker 退出", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger, err := logging.New(os.Getenv("LOG_LEVEL"))
	if err != nil {
		return err
	}
	slog.SetDefault(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := store.New(ctx, cfg.PostgresDSN)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.Migrate(ctx); err != nil {
		return err
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr, Password: cfg.RedisPassword, DB: cfg.RedisDB,
	})
	defer func() { _ = redisClient.Close() }()
	pingCtx, pingCancel := context.WithTimeout(ctx, 5*time.Second)
	defer pingCancel()
	if err := redisClient.Ping(pingCtx).Err(); err != nil {
		return fmt.Errorf("worker: 连接 Redis 失败: %w", err)
	}

	hostname, _ := os.Hostname()
	consumerName := fmt.Sprintf("%s-%d", hostname, os.Getpid())
	reg := metrics.New()
	reg.QueueLen = queueLenProbe(redisClient, cfg.StreamName)
	reg.PendingCount = pendingProbe(redisClient, cfg.StreamName, cfg.ConsumerGroup)
	reg.DBStats = db.PoolStats
	dlq := &worker.DeadLetter{Client: redisClient, Stream: cfg.DeadLetterStream, SourceStream: cfg.StreamName, MaxLen: 10000, Metrics: reg}
	consumer := &worker.Consumer{
		Streams:     redisClient,
		Stream:      cfg.StreamName,
		Group:       cfg.ConsumerGroup,
		Name:        consumerName,
		BatchSize:   cfg.BatchSize,
		Block:       cfg.BatchTimeout,
		ClaimIdle:   time.Minute,
		MaxAttempts: 5,
		Retry:       worker.RetryPolicy{Base: 100 * time.Millisecond, Max: 30 * time.Second},
		Processor: &worker.BatchProcessor{
			DB:  db,
			Agg: &aggregation.Aggregator{SessionTimeout: cfg.SessionTimeout},
		},
		DLQ:    dlq,
		Leases: &worker.Leases{Client: redisClient, Stream: cfg.StreamName, Owner: consumerName, TTL: 30 * time.Second},
		Reaper: &worker.Reaper{
			Client:     redisClient,
			DLQ:        dlq,
			MainStream: cfg.StreamName,
			Group:      cfg.ConsumerGroup,
			Owner:      consumerName,
		},
		Metrics: reg,
	}

	// 运维端口：先 Listen，绑定失败直接 fail fast；METRICS_ADDR 为空表示禁用。
	var opsSrv *http.Server
	if cfg.MetricsAddr != "" {
		ln, err := net.Listen("tcp", cfg.MetricsAddr)
		if err != nil {
			return fmt.Errorf("worker: 监听运维端口 %s 失败: %w", cfg.MetricsAddr, err)
		}
		opsSrv = &http.Server{
			Handler:           newWorkerMux(reg, &health.Handler{PG: db, Redis: redisPinger{redisClient}}),
			ReadHeaderTimeout: 5 * time.Second,
		}
		go func() {
			if err := opsSrv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.Error("worker 运维端口异常退出", "err", err)
			}
		}()
		logger.Info("worker 运维端口监听", "addr", cfg.MetricsAddr)
	}

	sigCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	logger.Info("worker 启动",
		"stream", cfg.StreamName,
		"group", cfg.ConsumerGroup,
		"batch_size", cfg.BatchSize,
		"retention_days", cfg.RetentionDays,
	)
	// 留存清理与消费循环同生命周期；RETENTION_DAYS<=0 时 janitor 自行禁用。
	janitor := &worker.RetentionJanitor{Store: db, RetentionDays: cfg.RetentionDays}
	go janitor.Run(sigCtx)
	runErr := consumer.Run(sigCtx)
	if opsSrv != nil {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		_ = opsSrv.Shutdown(shutdownCtx)
	}
	if runErr != nil {
		return runErr
	}
	logger.Info("worker 退出")
	return nil
}
