package store

import (
	"context"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// testDBName 是 store 包独占的测试库名，避免与其他测试包互相干扰。
const testDBName = "nodelogs_test_store"

// testPool 连接测试数据库并应用全部迁移；未设置 TEST_DATABASE_URL 时跳过。
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := ensureTestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("连接测试数据库失败: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("应用迁移失败: %v", err)
	}
	if _, err := pool.Exec(ctx, `TRUNCATE apps, users, raw_events, daily_visitors, daily_ips,
		daily_stats, daily_page_stats, daily_page_visitors, daily_page_sessions,
		daily_device_stats, sessions RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("清空测试数据库失败: %v", err)
	}
	return pool
}

// ensureTestDB 确保独占测试库存在，返回其 DSN。
func ensureTestDB(t *testing.T) string {
	t.Helper()
	maintenance := os.Getenv("TEST_DATABASE_URL")
	if maintenance == "" {
		t.Skip("未设置 TEST_DATABASE_URL，跳过 PostgreSQL 集成测试")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	mpool, err := pgxpool.New(ctx, maintenance)
	if err != nil {
		t.Fatalf("连接维护库失败: %v", err)
	}
	defer mpool.Close()
	var exists bool
	if err := mpool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)`, testDBName).Scan(&exists); err != nil {
		t.Fatalf("检查测试库失败: %v", err)
	}
	if !exists {
		if _, err := mpool.Exec(ctx, `CREATE DATABASE `+testDBName); err != nil {
			t.Fatalf("创建测试库失败: %v", err)
		}
	}
	// 注意：pgxpool.Config.ConnString 返回解析前的原始串，不反映字段改写，
	// 因此直接改写 URL（要求 TEST_DATABASE_URL 为 URL 格式）。
	if !strings.Contains(maintenance, "://") {
		t.Fatalf("TEST_DATABASE_URL 必须为 URL 格式")
	}
	u, err := url.Parse(maintenance)
	if err != nil {
		t.Fatalf("解析测试 DSN 失败: %v", err)
	}
	u.Path = "/" + testDBName
	return u.String()
}

// testStore 返回已迁移并清空的 Store，每用例调用一次保证隔离。
func testStore(t *testing.T) *Store {
	t.Helper()
	pool := testPool(t)
	return &Store{pool: pool}
}

// ensureApp 在测试库中预置应用行（raw_events 对 apps 有外键约束）。
func ensureApp(t *testing.T, s *Store, appID string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.CreateApp(ctx, App{AppID: appID, SecretKey: "test-secret", Name: "test"}); err != nil {
		t.Fatalf("预置应用失败: %v", err)
	}
}
