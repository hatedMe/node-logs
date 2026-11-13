package store

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPing(t *testing.T) {
	s := &Store{pool: testPool(t)}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.Ping(ctx); err != nil {
		t.Fatalf("Ping 应成功: %v", err)
	}
}

func TestPoolStatsOffline(t *testing.T) {
	// pgxpool.New 只解析 DSN 不建连，可离线断言零值统计。
	pool, err := pgxpool.New(context.Background(), "postgres://127.0.0.1:1/nonexistent")
	if err != nil {
		t.Fatalf("创建连接池失败: %v", err)
	}
	defer pool.Close()
	s := &Store{pool: pool}
	acquired, total := s.PoolStats()
	if acquired != 0 || total != 0 {
		t.Fatalf("空闲池统计应为零，实际 acquired=%d total=%d", acquired, total)
	}
}
