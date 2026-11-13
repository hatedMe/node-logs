// Command server 启动 Go 采集与 API 服务。
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/hatedMe/node-logs/internal/app"
	"github.com/hatedMe/node-logs/internal/auth"
	"github.com/hatedMe/node-logs/internal/collector"
	"github.com/hatedMe/node-logs/internal/config"
	"github.com/hatedMe/node-logs/internal/health"
	"github.com/hatedMe/node-logs/internal/httpx"
	"github.com/hatedMe/node-logs/internal/logging"
	"github.com/hatedMe/node-logs/internal/metrics"
	"github.com/hatedMe/node-logs/internal/queue"
	"github.com/hatedMe/node-logs/internal/report"
	"github.com/hatedMe/node-logs/internal/store"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server 退出", "err", err)
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

	if err := cfg.ValidateForServer(); err != nil {
		return err
	}

	ctx := context.Background()
	db, err := store.New(ctx, cfg.PostgresDSN)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.Migrate(ctx); err != nil {
		return err
	}

	producer, err := queue.NewStreamProducer(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, cfg.StreamName, 2*time.Second)
	if err != nil {
		return err
	}
	defer func() { _ = producer.Close() }()

	collect := &collector.CollectorHandler{
		Normalizer: &collector.EventNormalizer{
			IPSalt:            cfg.IPSalt,
			TrustProxyHeaders: cfg.TrustProxyHeaders,
		},
		Validator:      collector.Validator{MaxURLLength: cfg.MaxURLLength},
		Apps:           collector.NewCachingAppValidator(db, cfg.AppCacheTTL),
		Publisher:      producer,
		PublishTimeout: 2 * time.Second,
	}
	appHandler := &app.Handler{Service: &app.Service{
		Store:     db,
		JWTSecret: cfg.JWTSecret,
		JWTTTL:    cfg.JWTTTL,
	}}
	reportHandler := &report.Handler{
		Service: &report.Service{DB: db, MaxRangeDays: cfg.ReportMaxRangeDays},
		Timeout: cfg.ReportTimeout,
	}

	// 可观测性：就绪探测依赖 PG/Redis；指标抓取时实时查询队列长度与连接池。
	reg := metrics.New()
	reg.QueueLen = producer.Len
	reg.DBStats = db.PoolStats
	healthH := &health.Handler{PG: db, Redis: producer}

	router := newRouter(collect, appHandler, reportHandler, db, cfg.JWTSecret, healthH, reg)
	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		// 防慢速读/写与连接耗尽：采集请求体上限 1MB，报表查询另有 5 秒上下文超时。
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	sigCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("server 启动", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http 服务异常退出", "err", err)
			stop()
		}
	}()

	<-sigCtx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

// newRouter 组装全部 HTTP 路由。h/reg 为 nil 时不挂载对应运维路由。
func newRouter(collect *collector.CollectorHandler, appHandler *app.Handler, reportHandler *report.Handler, db *store.Store, jwtSecret string, h *health.Handler, reg *metrics.Registry) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	// 注意：不使用 RealIP 中间件。客户端 IP 由采集器按 TRUST_PROXY_HEADERS
	// 配置显式解析，默认不信任 X-Forwarded-For 等代理头（见任务 3）。
	r.Use(middleware.Recoverer)

	if h != nil {
		r.Get("/healthz", h.Live)
		r.Get("/readyz", h.Ready)
	} else {
		r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
			httpx.WriteSuccess(w, map[string]string{"status": "ok"})
		})
	}
	if reg != nil {
		r.Get("/metrics", reg.Handler().ServeHTTP)
	}
	collected := http.HandlerFunc(collect.Collect)
	if reg != nil {
		collected = observeCollect(reg, collected)
	}
	// 兼容旧版 JS SDK 的上报路径。
	r.Get("/pingd/front/web", collected.ServeHTTP)
	r.Post("/pingd/front/web", collected.ServeHTTP)
	// 新 SDK 上报路径。
	r.Get("/api/v1/collect", collected.ServeHTTP)
	r.Post("/api/v1/collect", collected.ServeHTTP)

	// 注册登录（公开）。
	r.Post("/api/v1/auth/register", appHandler.Register)
	r.Post("/api/v1/auth/login", appHandler.Login)
	// 应用管理（需登录）。
	r.With(func(next http.Handler) http.Handler {
		return auth.AuthMiddleware(jwtSecret, next)
	}).Post("/api/v1/apps", appHandler.CreateApp)

	// 报表（需登录 + 应用属主）。
	r.Route("/api/v1/reports", func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return auth.AuthMiddleware(jwtSecret, next)
		})
		r.Use(func(next http.Handler) http.Handler {
			return auth.RequireAppOwner(db, next)
		})
		r.Get("/overview", reportHandler.Overview)
		r.Get("/trend", reportHandler.Trend)
		r.Get("/pages", reportHandler.Pages)
		r.Get("/devices", reportHandler.Devices)
		r.Get("/sources", reportHandler.Sources)
	})
	return r
}

// statusWriter 记录 handler 写出的状态码，供指标统计使用。
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// observeCollect 记录采集请求的状态码分布。
func observeCollect(reg *metrics.Registry, next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		reg.ObserveCollect(sw.status)
	}
}
