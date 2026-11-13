package config

import (
	"os"
	"testing"
	"time"
)

func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"DATABASE_URL", "POSTGRES_DSN",
		"HTTP_ADDR", "METRICS_ADDR",
		"REDIS_ADDR", "REDIS_PASSWORD", "REDIS_DB",
		"JWT_SECRET", "JWT_TTL",
		"IP_SALT",
		"RETENTION_DAYS",
		"BATCH_SIZE", "BATCH_TIMEOUT",
		"SESSION_TIMEOUT",
		"STREAM_NAME", "CONSUMER_GROUP", "DEAD_LETTER_STREAM",
		"TRUST_PROXY_HEADERS",
		"MAX_URL_LENGTH",
		"REPORT_MAX_RANGE_DAYS", "REPORT_TIMEOUT",
		"APP_CACHE_TTL",
	} {
		if err := os.Unsetenv(k); err != nil {
			t.Fatalf("unsetenv %s: %v", k, err)
		}
	}
}

func TestLoadMissingRequired(t *testing.T) {
	clearEnv(t)

	if _, err := Load(); err == nil {
		t.Fatal("Load() 缺少 DATABASE_URL 时应返回错误，实际为 nil")
	}
}

func TestLoadDefaults(t *testing.T) {
	clearEnv(t)
	t.Setenv("DATABASE_URL", "postgres://localhost:5432/nodelogs?sslmode=disable")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() 返回意外错误: %v", err)
	}

	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr 默认值应为 :8080，实际为 %q", cfg.HTTPAddr)
	}
	if cfg.MetricsAddr != ":9090" {
		t.Errorf("MetricsAddr 默认值应为 :9090，实际为 %q", cfg.MetricsAddr)
	}
	if cfg.RedisAddr != "127.0.0.1:6379" {
		t.Errorf("RedisAddr 默认值应为 127.0.0.1:6379，实际为 %q", cfg.RedisAddr)
	}
	if cfg.RedisDB != 0 {
		t.Errorf("RedisDB 默认值应为 0，实际为 %d", cfg.RedisDB)
	}
	if cfg.RetentionDays != 30 {
		t.Errorf("RetentionDays 默认值应为 30，实际为 %d", cfg.RetentionDays)
	}
	if cfg.BatchSize != 500 {
		t.Errorf("BatchSize 默认值应为 500，实际为 %d", cfg.BatchSize)
	}
	if cfg.BatchTimeout != 2*time.Second {
		t.Errorf("BatchTimeout 默认值应为 2s，实际为 %v", cfg.BatchTimeout)
	}
	if cfg.SessionTimeout != 30*time.Minute {
		t.Errorf("SessionTimeout 默认值应为 30m，实际为 %v", cfg.SessionTimeout)
	}
	if cfg.StreamName == "" || cfg.ConsumerGroup == "" || cfg.DeadLetterStream == "" {
		t.Errorf("Stream 相关默认值不应为空: %+v", cfg)
	}
	if cfg.TrustProxyHeaders {
		t.Error("TrustProxyHeaders 默认值应为 false")
	}
	if cfg.MaxURLLength != 2048 {
		t.Errorf("MaxURLLength 默认值应为 2048，实际为 %d", cfg.MaxURLLength)
	}
	if cfg.ReportMaxRangeDays != 90 {
		t.Errorf("ReportMaxRangeDays 默认值应为 90，实际为 %d", cfg.ReportMaxRangeDays)
	}
	if cfg.AppCacheTTL != time.Minute {
		t.Errorf("AppCacheTTL 默认值应为 1m，实际为 %v", cfg.AppCacheTTL)
	}
	if cfg.IPSalt != "" {
		t.Errorf("IPSalt 不得有固定默认值（各部署必须独立配置），实际为 %q", cfg.IPSalt)
	}
}

func TestMetricsAddrEmptyDisables(t *testing.T) {
	clearEnv(t)
	t.Setenv("DATABASE_URL", "postgres://localhost:5432/nodelogs?sslmode=disable")
	t.Setenv("METRICS_ADDR", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() 返回意外错误: %v", err)
	}
	if cfg.MetricsAddr != "" {
		t.Errorf("METRICS_ADDR 显式置空时应禁用，实际为 %q", cfg.MetricsAddr)
	}
}

func TestLoadPostgresDSNAlias(t *testing.T) {
	clearEnv(t)
	t.Setenv("POSTGRES_DSN", "postgres://localhost:5432/aliasdb?sslmode=disable")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() 应接受 POSTGRES_DSN 别名，实际错误: %v", err)
	}
	if cfg.PostgresDSN != "postgres://localhost:5432/aliasdb?sslmode=disable" {
		t.Errorf("PostgresDSN 解析错误，实际为 %q", cfg.PostgresDSN)
	}
}

func TestLoadFull(t *testing.T) {
	clearEnv(t)
	t.Setenv("DATABASE_URL", "postgres://db:5432/prod?sslmode=require")
	t.Setenv("HTTP_ADDR", ":9090")
	t.Setenv("REDIS_ADDR", "redis:6379")
	t.Setenv("REDIS_PASSWORD", "s3cret")
	t.Setenv("REDIS_DB", "2")
	t.Setenv("JWT_SECRET", "jwt-secret")
	t.Setenv("JWT_TTL", "2h")
	t.Setenv("IP_SALT", "custom-salt")
	t.Setenv("RETENTION_DAYS", "7")
	t.Setenv("BATCH_SIZE", "100")
	t.Setenv("BATCH_TIMEOUT", "500ms")
	t.Setenv("SESSION_TIMEOUT", "15m")
	t.Setenv("STREAM_NAME", "events:test")
	t.Setenv("CONSUMER_GROUP", "test-group")
	t.Setenv("DEAD_LETTER_STREAM", "events:test:dlq")
	t.Setenv("TRUST_PROXY_HEADERS", "true")
	t.Setenv("MAX_URL_LENGTH", "4096")
	t.Setenv("REPORT_MAX_RANGE_DAYS", "31")
	t.Setenv("REPORT_TIMEOUT", "5s")
	t.Setenv("APP_CACHE_TTL", "30s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() 返回意外错误: %v", err)
	}

	if cfg.PostgresDSN != "postgres://db:5432/prod?sslmode=require" {
		t.Errorf("PostgresDSN = %q", cfg.PostgresDSN)
	}
	if cfg.HTTPAddr != ":9090" {
		t.Errorf("HTTPAddr = %q", cfg.HTTPAddr)
	}
	if cfg.RedisAddr != "redis:6379" || cfg.RedisPassword != "s3cret" || cfg.RedisDB != 2 {
		t.Errorf("Redis 配置解析错误: %+v", cfg)
	}
	if cfg.JWTSecret != "jwt-secret" {
		t.Errorf("JWTSecret = %q", cfg.JWTSecret)
	}
	if cfg.JWTTTL != 2*time.Hour {
		t.Errorf("JWTTTL = %v", cfg.JWTTTL)
	}
	if cfg.IPSalt != "custom-salt" {
		t.Errorf("IPSalt = %q", cfg.IPSalt)
	}
	if cfg.RetentionDays != 7 {
		t.Errorf("RetentionDays = %d", cfg.RetentionDays)
	}
	if cfg.BatchSize != 100 {
		t.Errorf("BatchSize = %d", cfg.BatchSize)
	}
	if cfg.BatchTimeout != 500*time.Millisecond {
		t.Errorf("BatchTimeout = %v", cfg.BatchTimeout)
	}
	if cfg.SessionTimeout != 15*time.Minute {
		t.Errorf("SessionTimeout = %v", cfg.SessionTimeout)
	}
	if cfg.StreamName != "events:test" || cfg.ConsumerGroup != "test-group" || cfg.DeadLetterStream != "events:test:dlq" {
		t.Errorf("Stream 配置解析错误: %+v", cfg)
	}
	if !cfg.TrustProxyHeaders {
		t.Error("TrustProxyHeaders 应为 true")
	}
	if cfg.MaxURLLength != 4096 {
		t.Errorf("MaxURLLength = %d", cfg.MaxURLLength)
	}
	if cfg.ReportMaxRangeDays != 31 {
		t.Errorf("ReportMaxRangeDays = %d", cfg.ReportMaxRangeDays)
	}
	if cfg.ReportTimeout != 5*time.Second {
		t.Errorf("ReportTimeout = %v", cfg.ReportTimeout)
	}
	if cfg.AppCacheTTL != 30*time.Second {
		t.Errorf("AppCacheTTL = %v", cfg.AppCacheTTL)
	}
}

func TestValidateForServer(t *testing.T) {
	clearEnv(t)
	t.Setenv("DATABASE_URL", "postgres://localhost:5432/nodelogs?sslmode=disable")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() 返回意外错误: %v", err)
	}
	if err := cfg.ValidateForServer(); err == nil {
		t.Error("缺少 JWT_SECRET 时 ValidateForServer 应返回错误，实际为 nil")
	}

	t.Setenv("JWT_SECRET", "s3cret")
	cfg, err = Load()
	if err != nil {
		t.Fatalf("Load() 返回意外错误: %v", err)
	}
	if err := cfg.ValidateForServer(); err == nil {
		t.Error("缺少 IP_SALT 时 ValidateForServer 应返回错误，实际为 nil")
	}

	t.Setenv("IP_SALT", "test-salt")
	cfg, err = Load()
	if err != nil {
		t.Fatalf("Load() 返回意外错误: %v", err)
	}
	if err := cfg.ValidateForServer(); err != nil {
		t.Errorf("必填项齐全时应通过，实际错误: %v", err)
	}
}

func TestLoadInvalidValues(t *testing.T) {
	cases := map[string]string{
		"RETENTION_DAYS":        "abc",
		"BATCH_SIZE":            "abc",
		"REDIS_DB":              "abc",
		"MAX_URL_LENGTH":        "abc",
		"REPORT_MAX_RANGE_DAYS": "abc",
		"BATCH_TIMEOUT":         "not-a-duration",
		"SESSION_TIMEOUT":       "not-a-duration",
		"JWT_TTL":               "not-a-duration",
		"REPORT_TIMEOUT":        "not-a-duration",
		"APP_CACHE_TTL":         "not-a-duration",
		"TRUST_PROXY_HEADERS":   "not-a-bool",
	}
	for key, val := range cases {
		t.Run(key, func(t *testing.T) {
			clearEnv(t)
			t.Setenv("DATABASE_URL", "postgres://localhost:5432/nodelogs?sslmode=disable")
			t.Setenv(key, val)
			if _, err := Load(); err == nil {
				t.Fatalf("Load() 在 %s=%q 时应返回错误，实际为 nil", key, val)
			}
		})
	}
}
