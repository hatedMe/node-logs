package worker

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hatedMe/node-logs/internal/aggregation"
	"github.com/hatedMe/node-logs/internal/store"
	"github.com/hatedMe/node-logs/internal/testdb"
)

// TestProcessIntegration 用真实 PostgreSQL 验证端到端落库与幂等。
func TestProcessIntegration(t *testing.T) {
	dsn := testdb.Prepare(t, "nodelogs_test_worker")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := store.New(ctx, dsn)
	if err != nil {
		t.Fatalf("创建 Store 失败: %v", err)
	}
	defer db.Close()
	if err := db.CreateApp(ctx, store.App{AppID: "800800", SecretKey: "sk", Name: "t"}); err != nil {
		t.Fatalf("预置应用失败: %v", err)
	}

	bp := &BatchProcessor{DB: db, Agg: &aggregation.Aggregator{SessionTimeout: 30 * time.Minute}}
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	e3 := batchEvent("e3", "800800", "v2", "/a", at.Add(2*time.Minute))
	e3.IPHash = "ip-v1"
	events := []store.RawEvent{
		batchEvent("e1", "800800", "v1", "/a", at),
		batchEvent("e2", "800800", "v1", "/b", at.Add(time.Minute)),
		e3,
	}

	res, err := bp.Process(ctx, events)
	if err != nil {
		t.Fatalf("Process 失败: %v", err)
	}
	if res.Inserted != 3 {
		t.Fatalf("首次处理应插入 3 条: %+v", res)
	}
	res, err = bp.Process(ctx, events)
	if err != nil {
		t.Fatalf("重复处理失败: %v", err)
	}
	if res.Inserted != 0 {
		t.Fatalf("重复处理插入数应为 0: %+v", res)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("连接测试库失败: %v", err)
	}
	defer pool.Close()
	var pv, uv, ips, sessions int64
	err = pool.QueryRow(ctx, `SELECT pv, uv, ip_count, sessions FROM daily_stats
		WHERE app_id = '800800' AND stat_date = '2026-11-12'`).Scan(&pv, &uv, &ips, &sessions)
	if err != nil {
		t.Fatalf("读取聚合结果失败: %v", err)
	}
	if pv != 3 || uv != 2 || ips != 1 || sessions != 2 {
		t.Errorf("聚合结果错误: pv=%d uv=%d ips=%d sessions=%d", pv, uv, ips, sessions)
	}
	var raw int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM raw_events`).Scan(&raw); err != nil {
		t.Fatalf("统计原始事件失败: %v", err)
	}
	if raw != 3 {
		t.Errorf("原始事件应为 3 条，实际 %d 条", raw)
	}
}

// TestProcessConcurrentSameVisitor 验证多 worker 并发处理同访客时只创建一个会话。
// 注意：同日期同访客会被 UV 去重键的投机锁意外串行化，真竞态发生在跨日期批次
// （UV 键不同，会话窗口重叠），因此本测试用跨午夜的两批构造并发碰撞。
func TestProcessConcurrentSameVisitor(t *testing.T) {
	dsn := testdb.Prepare(t, "nodelogs_test_worker")
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	db, err := store.New(ctx, dsn)
	if err != nil {
		t.Fatalf("创建 Store 失败: %v", err)
	}
	defer db.Close()
	if err := db.CreateApp(ctx, store.App{AppID: "800800", SecretKey: "sk", Name: "t"}); err != nil {
		t.Fatalf("预置应用失败: %v", err)
	}

	bp := &BatchProcessor{DB: db, Agg: &aggregation.Aggregator{SessionTimeout: 30 * time.Minute}}
	day1 := time.Date(2026, 11, 12, 23, 50, 0, 0, time.UTC)
	day2 := time.Date(2026, 11, 13, 0, 5, 0, 0, time.UTC)

	const perWorker = 40
	build := func(prefix string, base time.Time) []store.RawEvent {
		batch := make([]store.RawEvent, 0, perWorker)
		for j := range perWorker {
			batch = append(batch, batchEvent(fmt.Sprintf("%s-%d", prefix, j), "800800", "v-conc", "/a", base.Add(time.Duration(j)*time.Second)))
		}
		return batch
	}

	start := make(chan struct{})
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i, batch := range [][]store.RawEvent{build("conc-a", day1), build("conc-b", day2)} {
		wg.Add(1)
		go func(i int, batch []store.RawEvent) {
			defer wg.Done()
			<-start
			_, errs[i] = bp.Process(ctx, batch)
		}(i, batch)
	}
	close(start)
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("worker %d 处理失败: %v", i, err)
		}
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("连接测试库失败: %v", err)
	}
	defer pool.Close()
	var sessions int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM sessions WHERE app_id = '800800' AND visitor_id = 'v-conc'`).Scan(&sessions); err != nil {
		t.Fatalf("统计会话失败: %v", err)
	}
	if sessions != 1 {
		t.Errorf("同访客并发只应创建 1 个会话，实际 %d 个", sessions)
	}
	var statSessions int64
	if err := pool.QueryRow(ctx, `SELECT COALESCE(SUM(sessions),0) FROM daily_stats WHERE app_id = '800800'`).Scan(&statSessions); err != nil {
		t.Fatalf("读取会话统计失败: %v", err)
	}
	if statSessions != 1 {
		t.Errorf("会话统计应为 1，实际为 %d", statSessions)
	}
	var eventCount int64
	if err := pool.QueryRow(ctx, `SELECT event_count FROM sessions WHERE app_id = '800800' AND visitor_id = 'v-conc'`).Scan(&eventCount); err != nil {
		t.Fatalf("读取会话事件数失败: %v", err)
	}
	if eventCount != 2*perWorker {
		t.Errorf("唯一会话应包含全部 %d 个事件，实际为 %d", 2*perWorker, eventCount)
	}
}
