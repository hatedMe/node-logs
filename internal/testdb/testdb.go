// Package testdb 为集成测试准备按包隔离的 PostgreSQL 测试库。
// 各测试包使用独立库名，避免 go test 并行执行时互相干扰。
package testdb

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hatedMe/node-logs/internal/store"
)

// Prepare 确保名为 dbName 的测试库存在、已迁移并被清空，返回其 DSN。
// 未设置 TEST_DATABASE_URL（维护库 DSN）时跳过测试。
func Prepare(t *testing.T, dbName string) string {
	t.Helper()
	dsn := EnsureDB(t, dbName)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("连接测试库 %s 失败: %v", dbName, err)
	}
	defer pool.Close()
	if err := store.Migrate(ctx, pool); err != nil {
		t.Fatalf("迁移测试库 %s 失败: %v", dbName, err)
	}
	if err := Truncate(ctx, pool); err != nil {
		t.Fatalf("清空测试库 %s 失败: %v", dbName, err)
	}
	return dsn
}

// EnsureDB 确保名为 dbName 的测试库存在，返回其 DSN（不迁移、不清空）。
// store 包内部测试请使用本函数（配合 store.Migrate 与 Truncate），
// 以避免测试包导入环。
func EnsureDB(t *testing.T, dbName string) string {
	t.Helper()
	maintenance := maintenanceDSN(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	mpool, err := pgxpool.New(ctx, maintenance)
	if err != nil {
		t.Fatalf("连接维护库失败: %v", err)
	}
	defer mpool.Close()
	var exists bool
	if err := mpool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)`, dbName).Scan(&exists); err != nil {
		t.Fatalf("检查测试库失败: %v", err)
	}
	if !exists {
		stmt := fmt.Sprintf(`CREATE DATABASE %s`, pgx.Identifier{dbName}.Sanitize())
		if _, err := mpool.Exec(ctx, stmt); err != nil {
			t.Fatalf("创建测试库 %s 失败: %v", dbName, err)
		}
	}

	return replaceDBName(t, maintenance, dbName)
}

// replaceDBName 将维护库 DSN 的库名替换为 dbName。
// 注意：pgxpool.Config.ConnString 返回解析前的原始串，不反映字段改写，
// 因此直接改写 URL（要求 TEST_DATABASE_URL 为 URL 格式）。
func replaceDBName(t *testing.T, dsn, dbName string) string {
	t.Helper()
	if !strings.Contains(dsn, "://") {
		t.Fatalf("TEST_DATABASE_URL 必须为 URL 格式")
	}
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("解析测试 DSN 失败: %v", err)
	}
	u.Path = "/" + dbName
	return u.String()
}

// Truncate 清空测试库全部业务表。
func Truncate(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `TRUNCATE apps, users, raw_events, daily_visitors, daily_ips,
		daily_stats, daily_page_stats, daily_page_visitors, daily_page_sessions,
		daily_device_stats, sessions RESTART IDENTITY CASCADE`)
	return err
}

func maintenanceDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("未设置 TEST_DATABASE_URL，跳过 PostgreSQL 集成测试")
	}
	return dsn
}
