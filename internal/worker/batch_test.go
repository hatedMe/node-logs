package worker

import (
	"context"
	"testing"
	"time"

	"github.com/hatedMe/node-logs/internal/aggregation"
	"github.com/hatedMe/node-logs/internal/store"
	"github.com/hatedMe/node-logs/internal/testdb"
)

func batchEvent(eventID, appID, visitorID, url string, at time.Time) store.RawEvent {
	return store.RawEvent{
		EventID: eventID, AppID: appID, VisitorID: visitorID, URL: url,
		EventTime: at, ReceivedAt: at,
		UA:     "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		IPHash: "ip-" + visitorID,
		Extra:  map[string]string{},
	}
}

func testProcessor() *BatchProcessor {
	return &BatchProcessor{Agg: &aggregation.Aggregator{SessionTimeout: 30 * time.Minute}}
}

func activeApp(appID string) store.App {
	return store.App{AppID: appID, SecretKey: "sk", Name: "t", Status: "active"}
}

func TestProcessTxInsertsAndAggregates(t *testing.T) {
	db := testdb.NewFakeStore()
	db.Apps["800800"] = activeApp("800800")
	bp := testProcessor()
	ctx := context.Background()
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)

	e3 := batchEvent("e3", "800800", "v2", "/a", at.Add(2*time.Minute))
	e3.IPHash = "ip-v1" // 与 v1 同 IP
	events := []store.RawEvent{
		batchEvent("e1", "800800", "v1", "/a", at),
		batchEvent("e2", "800800", "v1", "/b", at.Add(time.Minute)),
		e3,
	}
	res, err := bp.processTx(ctx, db, events)
	if err != nil {
		t.Fatalf("processTx 失败: %v", err)
	}
	if res.Total != 3 || res.Inserted != 3 || res.Dropped != 0 {
		t.Errorf("处理结果错误: %+v", res)
	}
	if len(db.Raw) != 3 {
		t.Errorf("应落库 3 条原始事件，实际 %d 条", len(db.Raw))
	}
	if db.Raw["e1"].BrowserName != "Chrome" || db.Raw["e1"].DeviceType != "desktop" {
		t.Errorf("原始事件应填充 UA 解析结果: %+v", db.Raw["e1"])
	}
	day := testdb.DayKey{AppID: "800800", Day: time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC)}
	d := db.Daily[day]
	if d == nil || d.PV != 3 || d.UV != 2 || d.IPCount != 1 {
		t.Errorf("日汇总错误: %+v", d)
	}
}

func TestProcessTxDropsUnknownAndDisabled(t *testing.T) {
	db := testdb.NewFakeStore()
	db.Apps["800800"] = activeApp("800800")
	db.Apps["800801"] = store.App{AppID: "800801", SecretKey: "sk", Status: "disabled"}
	bp := testProcessor()
	ctx := context.Background()
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)

	events := []store.RawEvent{
		batchEvent("e1", "800800", "v1", "/a", at),
		batchEvent("e2", "no-such", "v1", "/a", at),
		batchEvent("e3", "800801", "v1", "/a", at),
	}
	res, err := bp.processTx(ctx, db, events)
	if err != nil {
		t.Fatalf("processTx 失败: %v", err)
	}
	if res.Inserted != 1 || res.Dropped != 2 {
		t.Errorf("应插入 1 条丢弃 2 条: %+v", res)
	}
	if len(db.Raw) != 1 {
		t.Errorf("只应落库 1 条，实际 %d 条", len(db.Raw))
	}
}

func TestProcessTxIdempotent(t *testing.T) {
	db := testdb.NewFakeStore()
	db.Apps["800800"] = activeApp("800800")
	bp := testProcessor()
	ctx := context.Background()
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)

	events := []store.RawEvent{batchEvent("e1", "800800", "v1", "/a", at)}
	if _, err := bp.processTx(ctx, db, events); err != nil {
		t.Fatalf("首次处理失败: %v", err)
	}
	res, err := bp.processTx(ctx, db, events)
	if err != nil {
		t.Fatalf("重复处理失败: %v", err)
	}
	if res.Inserted != 0 {
		t.Errorf("重复批次插入数应为 0: %+v", res)
	}
	day := testdb.DayKey{AppID: "800800", Day: time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC)}
	if db.Daily[day].PV != 1 {
		t.Errorf("重复处理不应重复计数: %+v", db.Daily[day])
	}
}

func TestProcessTxDedupsWithinBatch(t *testing.T) {
	db := testdb.NewFakeStore()
	db.Apps["800800"] = activeApp("800800")
	bp := testProcessor()
	ctx := context.Background()
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)

	dup := batchEvent("e1", "800800", "v1", "/other", at.Add(time.Minute))
	events := []store.RawEvent{
		batchEvent("e1", "800800", "v1", "/a", at),
		dup,
		batchEvent("e2", "800800", "v2", "/a", at),
	}
	res, err := bp.processTx(ctx, db, events)
	if err != nil {
		t.Fatalf("processTx 失败: %v", err)
	}
	if res.Total != 3 || res.Dupes != 1 || res.Inserted != 2 {
		t.Errorf("批内去重结果错误: %+v", res)
	}
	if len(db.Raw) != 2 {
		t.Fatalf("批内重复只应落库 1 条，实际 raw=%d", len(db.Raw))
	}
	if db.Raw["e1"].URL != "/a" {
		t.Errorf("重复 event_id 应保留首条，实际为 %+v", db.Raw["e1"])
	}
	day := testdb.DayKey{AppID: "800800", Day: time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC)}
	if db.Daily[day].PV != 2 {
		t.Errorf("批内重复不应重复计 PV: %+v", db.Daily[day])
	}
}

func TestProcessTxBackfillsSessionID(t *testing.T) {
	db := testdb.NewFakeStore()
	db.Apps["800800"] = activeApp("800800")
	bp := testProcessor()
	ctx := context.Background()
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)

	events := []store.RawEvent{
		batchEvent("e1", "800800", "v1", "/a", at),
		batchEvent("e2", "800800", "v1", "/b", at.Add(time.Minute)),
		batchEvent("e3", "800800", "v2", "/a", at),
	}
	if _, err := bp.processTx(ctx, db, events); err != nil {
		t.Fatalf("processTx 失败: %v", err)
	}
	s1, s2, s3 := db.Raw["e1"].SessionID, db.Raw["e2"].SessionID, db.Raw["e3"].SessionID
	if s1 == "" || s1 != s2 {
		t.Errorf("同访客原始事件应回填同一 session_id: %q %q", s1, s2)
	}
	if s3 == "" || s3 == s1 {
		t.Errorf("不同访客应回填不同 session_id: %q vs %q", s3, s1)
	}
}

func TestProcessTxLocksVisitorsSorted(t *testing.T) {
	db := testdb.NewFakeStore()
	db.Apps["800800"] = activeApp("800800")
	bp := testProcessor()
	ctx := context.Background()
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)

	// 事件时间顺序与访客字典序故意相反，加锁必须按全局固定顺序。
	events := []store.RawEvent{
		batchEvent("e1", "800800", "v2", "/a", at),
		batchEvent("e2", "800800", "v1", "/a", at.Add(time.Minute)),
		batchEvent("e3", "800800", "v2", "/b", at.Add(2*time.Minute)),
	}
	if _, err := bp.processTx(ctx, db, events); err != nil {
		t.Fatalf("processTx 失败: %v", err)
	}
	// 预锁定（有序去重）+ 聚合慢路径锁（重入）：预锁定部分必须有序。
	if len(db.Locks) < 2 {
		t.Fatalf("应记录加锁，实际 %v", db.Locks)
	}
	if db.Locks[0] != "800800\x00v1" || db.Locks[1] != "800800\x00v2" {
		t.Errorf("预锁定必须按全局固定顺序，实际 %q", db.Locks)
	}
}

func TestProcessTxEmpty(t *testing.T) {
	bp := testProcessor()
	res, err := bp.processTx(context.Background(), testdb.NewFakeStore(), nil)
	if err != nil {
		t.Fatalf("空批次应成功: %v", err)
	}
	if res.Total != 0 || res.Inserted != 0 {
		t.Errorf("空批次结果应为零值: %+v", res)
	}
}

func TestProcessTxNormalizesZeroTimes(t *testing.T) {
	db := testdb.NewFakeStore()
	db.Apps["800800"] = activeApp("800800")
	bp := testProcessor()

	e := batchEvent("e1", "800800", "v1", "/a", time.Time{})
	e.ReceivedAt = time.Time{}
	if _, err := bp.processTx(context.Background(), db, []store.RawEvent{e}); err != nil {
		t.Fatalf("processTx 失败: %v", err)
	}
	got := db.Raw["e1"]
	if got.EventTime.IsZero() || got.ReceivedAt.IsZero() {
		t.Errorf("零时间应被规范化: %+v", got)
	}
}
