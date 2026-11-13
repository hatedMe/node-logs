// Package config 从环境变量加载 Go 服务运行配置。
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config 聚合 HTTP、PostgreSQL、Redis、认证与采集流水线参数。
type Config struct {
	HTTPAddr string

	// MetricsAddr 是 worker 运维端口（/metrics、/healthz、/readyz）；
	// 为空表示禁用该端口。
	MetricsAddr string

	PostgresDSN string

	RedisAddr     string
	RedisPassword string
	RedisDB       int

	JWTSecret string
	JWTTTL    time.Duration

	IPSalt string

	RetentionDays int

	BatchSize    int
	BatchTimeout time.Duration

	SessionTimeout time.Duration

	StreamName       string
	ConsumerGroup    string
	DeadLetterStream string

	TrustProxyHeaders bool

	MaxURLLength int

	ReportMaxRangeDays int
	ReportTimeout      time.Duration

	AppCacheTTL time.Duration
}

// ValidateForServer 校验 server 进程的额外必填配置（JWT_SECRET、IP_SALT）。
// Load 本身不强制要求这些，因为 worker 进程不需要它们。
// IP_SALT 故意无默认值：各部署必须独立配置，固定盐会削弱哈希隔离。
func (c Config) ValidateForServer() error {
	if c.JWTSecret == "" {
		return errors.New("config: server 缺少必填配置 JWT_SECRET")
	}
	if c.IPSalt == "" {
		return errors.New("config: server 缺少必填配置 IP_SALT")
	}
	return nil
}

// Load 从环境变量读取配置。DATABASE_URL（或 POSTGRES_DSN 别名）为必填项，
// 其余字段均有面向本地开发的默认值。
func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:           getEnv("HTTP_ADDR", ":8080"),
		MetricsAddr:        getEnvAllowEmpty("METRICS_ADDR", ":9090"),
		PostgresDSN:        firstNonEmpty(os.Getenv("DATABASE_URL"), os.Getenv("POSTGRES_DSN")),
		RedisAddr:          getEnv("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPassword:      os.Getenv("REDIS_PASSWORD"),
		JWTSecret:          os.Getenv("JWT_SECRET"),
		IPSalt:             os.Getenv("IP_SALT"),
		StreamName:         getEnv("STREAM_NAME", "events:pageviews"),
		ConsumerGroup:      getEnv("CONSUMER_GROUP", "analytics-workers"),
		DeadLetterStream:   getEnv("DEAD_LETTER_STREAM", "events:pageviews:dlq"),
		TrustProxyHeaders:  false,
		RetentionDays:      30,
		BatchSize:          500,
		BatchTimeout:       2 * time.Second,
		SessionTimeout:     30 * time.Minute,
		JWTTTL:             2 * time.Hour,
		MaxURLLength:       2048,
		ReportMaxRangeDays: 90,
		ReportTimeout:      5 * time.Second,
		AppCacheTTL:        time.Minute,
	}

	if cfg.PostgresDSN == "" {
		return Config{}, errors.New("config: 缺少必填配置 DATABASE_URL（或 POSTGRES_DSN）")
	}

	var err error
	if cfg.RedisDB, err = getInt("REDIS_DB", cfg.RedisDB); err != nil {
		return Config{}, err
	}
	if cfg.RetentionDays, err = getInt("RETENTION_DAYS", cfg.RetentionDays); err != nil {
		return Config{}, err
	}
	if cfg.BatchSize, err = getInt("BATCH_SIZE", cfg.BatchSize); err != nil {
		return Config{}, err
	}
	if cfg.MaxURLLength, err = getInt("MAX_URL_LENGTH", cfg.MaxURLLength); err != nil {
		return Config{}, err
	}
	if cfg.ReportMaxRangeDays, err = getInt("REPORT_MAX_RANGE_DAYS", cfg.ReportMaxRangeDays); err != nil {
		return Config{}, err
	}
	if cfg.BatchTimeout, err = getDuration("BATCH_TIMEOUT", cfg.BatchTimeout); err != nil {
		return Config{}, err
	}
	if cfg.SessionTimeout, err = getDuration("SESSION_TIMEOUT", cfg.SessionTimeout); err != nil {
		return Config{}, err
	}
	if cfg.JWTTTL, err = getDuration("JWT_TTL", cfg.JWTTTL); err != nil {
		return Config{}, err
	}
	if cfg.ReportTimeout, err = getDuration("REPORT_TIMEOUT", cfg.ReportTimeout); err != nil {
		return Config{}, err
	}
	if cfg.AppCacheTTL, err = getDuration("APP_CACHE_TTL", cfg.AppCacheTTL); err != nil {
		return Config{}, err
	}
	if cfg.TrustProxyHeaders, err = getBool("TRUST_PROXY_HEADERS", cfg.TrustProxyHeaders); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// getEnvAllowEmpty 与 getEnv 相同，但变量显式设为空时返回空（用于可禁用的开关型地址）。
func getEnvAllowEmpty(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func getInt(key string, def int) (int, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return def, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("config: 环境变量 %s=%q 不是合法整数: %w", key, raw, err)
	}
	return v, nil
}

func getDuration(key string, def time.Duration) (time.Duration, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return def, nil
	}
	v, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("config: 环境变量 %s=%q 不是合法时长: %w", key, raw, err)
	}
	return v, nil
}

func getBool(key string, def bool) (bool, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return def, nil
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("config: 环境变量 %s=%q 不是合法布尔值: %w", key, raw, err)
	}
	return v, nil
}
