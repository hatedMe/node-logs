package store

import (
	"context"
	"testing"
	"time"
)

func TestMigrateAppliesAll(t *testing.T) {
	pool := testPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tables := []string{
		"schema_migrations",
		"apps", "users", "raw_events",
		"daily_visitors", "daily_ips", "daily_stats", "daily_page_stats",
		"daily_page_visitors", "daily_page_sessions", "daily_device_stats",
		"sessions",
	}
	for _, table := range tables {
		var exists bool
		err := pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = $1)`, table).Scan(&exists)
		if err != nil {
			t.Fatalf("检查表 %s 失败: %v", table, err)
		}
		if !exists {
			t.Errorf("迁移后缺少表 %s", table)
		}
	}
}

func TestMigrateIdempotent(t *testing.T) {
	pool := testPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("重复应用迁移应成功，实际错误: %v", err)
	}
}

func TestMigrationPrimaryKeys(t *testing.T) {
	pool := testPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cases := map[string][]string{
		"apps":               {"app_id"},
		"users":              {"id"},
		"raw_events":         {"event_id"},
		"daily_visitors":     {"app_id", "stat_date", "visitor_id"},
		"daily_ips":          {"app_id", "stat_date", "ip_hash"},
		"daily_stats":        {"app_id", "stat_date"},
		"daily_page_stats":   {"app_id", "stat_date", "url"},
		"sessions":           {"session_id"},
		"daily_device_stats": {"app_id", "stat_date", "dimension", "value"},
	}
	for table, wantCols := range cases {
		rows, err := pool.Query(ctx, `
			SELECT a.attname
			FROM pg_index i
			JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
			JOIN pg_class c ON c.oid = i.indrelid
			WHERE c.relname = $1 AND i.indisprimary
			ORDER BY array_position(i.indkey, a.attnum)`, table)
		if err != nil {
			t.Fatalf("查询表 %s 主键失败: %v", table, err)
		}
		var got []string
		for rows.Next() {
			var col string
			if err := rows.Scan(&col); err != nil {
				rows.Close()
				t.Fatalf("读取主键列失败: %v", err)
			}
			got = append(got, col)
		}
		rows.Close()
		if len(got) != len(wantCols) {
			t.Errorf("表 %s 主键应为 %v，实际为 %v", table, wantCols, got)
			continue
		}
		for i := range wantCols {
			if got[i] != wantCols[i] {
				t.Errorf("表 %s 主键应为 %v，实际为 %v", table, wantCols, got)
				break
			}
		}
	}
}

func TestMigrationIndexes(t *testing.T) {
	pool := testPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	indexes := []string{
		"idx_raw_events_app_time",
		"idx_raw_events_received",
		"idx_sessions_visitor",
	}
	for _, idx := range indexes {
		var exists bool
		err := pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM pg_class WHERE relname = $1 AND relkind = 'i')`, idx).Scan(&exists)
		if err != nil {
			t.Fatalf("检查索引 %s 失败: %v", idx, err)
		}
		if !exists {
			t.Errorf("迁移后缺少索引 %s", idx)
		}
	}
}

func TestUsersUsernameUnique(t *testing.T) {
	s := testStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := s.CreateUser(ctx, "alice", "hash-1"); err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	if _, err := s.CreateUser(ctx, "alice", "hash-2"); err == nil {
		t.Fatal("重复用户名应违反唯一约束，实际成功")
	}
}
