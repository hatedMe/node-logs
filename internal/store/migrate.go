package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	nodelogsMigrations "github.com/hatedMe/node-logs/migrations"
)

// migrationsFS 内嵌仓库根目录的 migrations/*.sql。
var migrationsFS = nodelogsMigrations.FS

// Migrate 按文件名顺序应用全部未执行迁移；重复调用安全。
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`); err != nil {
		return fmt.Errorf("store: 初始化 schema_migrations 失败: %w", err)
	}

	entries, err := migrationsFS.ReadDir(".")
	if err != nil {
		return fmt.Errorf("store: 读取内嵌迁移失败: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		version := entry.Name()
		var applied bool
		if err := pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, version).Scan(&applied); err != nil {
			return fmt.Errorf("store: 检查迁移 %s 失败: %w", version, err)
		}
		if applied {
			continue
		}
		sqlBytes, err := migrationsFS.ReadFile(version)
		if err != nil {
			return fmt.Errorf("store: 读取迁移 %s 失败: %w", version, err)
		}
		if err := applyMigration(ctx, pool, version, string(sqlBytes)); err != nil {
			return err
		}
	}
	return nil
}

func applyMigration(ctx context.Context, pool *pgxpool.Pool, version, sql string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("store: 迁移 %s 开启事务失败: %w", version, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// 无参数 Exec 走 simple 协议，单文件多语句可一次执行。
	if _, err := tx.Exec(ctx, sql); err != nil {
		return fmt.Errorf("store: 应用迁移 %s 失败: %w", version, err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO schema_migrations (version) VALUES ($1) ON CONFLICT (version) DO NOTHING`, version); err != nil {
		return fmt.Errorf("store: 记录迁移 %s 失败: %w", version, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("store: 提交迁移 %s 失败: %w", version, err)
	}
	return nil
}
