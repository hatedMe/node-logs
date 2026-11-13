package store

import (
	"context"
	"errors"
	"testing"
	"time"
)

func testEvent(appID, eventID, visitorID string) RawEvent {
	return RawEvent{
		EventID:     eventID,
		AppID:       appID,
		VisitorID:   visitorID,
		SessionID:   "sess-" + visitorID,
		URL:         "/home",
		Arg:         "?a=1",
		Version:     "1.0.0",
		Screen:      "1920x1080",
		Lang:        "zh-cn",
		TZ:          "-8",
		Domain:      "example.com",
		EventTime:   time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC),
		ReceivedAt:  time.Date(2026, 11, 12, 10, 0, 1, 0, time.UTC),
		UA:          "test-agent",
		IPHash:      "iphash-1",
		BrowserName: "Chrome",
		BrowserVer:  "120",
		System:      "Windows",
		SystemVer:   "10",
		DeviceType:  "desktop",
		Extra:       map[string]string{"dm": "example.com"},
	}
}

func TestInsertRawEventsBatchAndIdempotent(t *testing.T) {
	s := testStore(t)
	ensureApp(t, s, "800800")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	events := []RawEvent{
		testEvent("800800", "evt-1", "visitor-1"),
		testEvent("800800", "evt-2", "visitor-1"),
		testEvent("800800", "evt-3", "visitor-2"),
	}
	if err := s.InsertRawEvents(ctx, events); err != nil {
		t.Fatalf("批量写入原始事件失败: %v", err)
	}
	// 重复 event_id 不得重复写入。
	if err := s.InsertRawEvents(ctx, events[:2]); err != nil {
		t.Fatalf("重复写入应幂等成功，实际错误: %v", err)
	}

	var count int
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM raw_events`).Scan(&count); err != nil {
		t.Fatalf("统计原始事件失败: %v", err)
	}
	if count != 3 {
		t.Errorf("raw_events 应有 3 行，实际 %d 行", count)
	}
}

func TestInsertRawEventsReturning(t *testing.T) {
	s := testStore(t)
	ensureApp(t, s, "800800")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	events := []RawEvent{testEvent("800800", "evt-1", "v1")}
	if err := s.InsertRawEvents(ctx, events); err != nil {
		t.Fatalf("首次写入失败: %v", err)
	}
	got, err := s.InsertRawEventsReturning(ctx, append(events, testEvent("800800", "evt-2", "v1")))
	if err != nil {
		t.Fatalf("InsertRawEventsReturning 失败: %v", err)
	}
	if len(got) != 1 || got[0] != "evt-2" {
		t.Errorf("应只返回新插入的 [evt-2]，实际为 %v", got)
	}
}

func TestUpdateRawSessionIDs(t *testing.T) {
	s := testStore(t)
	ensureApp(t, s, "800800")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	events := []RawEvent{testEvent("800800", "evt-1", "v1"), testEvent("800800", "evt-2", "v2")}
	if err := s.InsertRawEvents(ctx, events); err != nil {
		t.Fatalf("写入原始事件失败: %v", err)
	}
	err := s.UpdateRawSessionIDs(ctx, map[string]string{"evt-1": "sess-1", "evt-2": "sess-2"})
	if err != nil {
		t.Fatalf("回填 session_id 失败: %v", err)
	}
	var s1, s2 string
	err = s.pool.QueryRow(ctx, `SELECT session_id FROM raw_events WHERE event_id = 'evt-1'`).Scan(&s1)
	if err != nil {
		t.Fatalf("读取回填结果失败: %v", err)
	}
	err = s.pool.QueryRow(ctx, `SELECT session_id FROM raw_events WHERE event_id = 'evt-2'`).Scan(&s2)
	if err != nil {
		t.Fatalf("读取回填结果失败: %v", err)
	}
	if s1 != "sess-1" || s2 != "sess-2" {
		t.Errorf("回填错误: %q %q", s1, s2)
	}
	if err := s.UpdateRawSessionIDs(ctx, map[string]string{}); err != nil {
		t.Errorf("空映射应直接成功: %v", err)
	}
}

func TestInsertVisitorAndIPIfNew(t *testing.T) {
	s := testStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	day := time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC)

	isNew, err := s.InsertVisitorIfNew(ctx, VisitorKey{AppID: "800800", Date: day, VisitorID: "v1"})
	if err != nil || !isNew {
		t.Fatalf("首次访客应返回 new=true，实际 new=%v err=%v", isNew, err)
	}
	isNew, err = s.InsertVisitorIfNew(ctx, VisitorKey{AppID: "800800", Date: day, VisitorID: "v1"})
	if err != nil || isNew {
		t.Fatalf("重复访客应返回 new=false，实际 new=%v err=%v", isNew, err)
	}

	isNew, err = s.InsertIPIfNew(ctx, IPKey{AppID: "800800", Date: day, IPHash: "h1"})
	if err != nil || !isNew {
		t.Fatalf("首次 IP 应返回 new=true，实际 new=%v err=%v", isNew, err)
	}
	isNew, err = s.InsertIPIfNew(ctx, IPKey{AppID: "800800", Date: day, IPHash: "h1"})
	if err != nil || isNew {
		t.Fatalf("重复 IP 应返回 new=false，实际 new=%v err=%v", isNew, err)
	}
}

func TestUpsertDailyStats(t *testing.T) {
	s := testStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	day := time.Date(2026, 11, 12, 15, 4, 5, 0, time.UTC) // 非零时刻也应归到同一自然日

	if err := s.UpsertDailyStats(ctx, StatsDelta{AppID: "800800", Date: day, PV: 3, UV: 2, IPCount: 2, Sessions: 2, Bounces: 1}); err != nil {
		t.Fatalf("首次聚合更新失败: %v", err)
	}
	if err := s.UpsertDailyStats(ctx, StatsDelta{AppID: "800800", Date: day, PV: 2, UV: 1, Bounces: -1}); err != nil {
		t.Fatalf("增量聚合更新失败: %v", err)
	}

	var pv, uv, ips, sessions, bounces int64
	err := s.pool.QueryRow(ctx,
		`SELECT pv, uv, ip_count, sessions, bounces FROM daily_stats WHERE app_id = $1 AND stat_date = $2`,
		"800800", "2026-11-12").Scan(&pv, &uv, &ips, &sessions, &bounces)
	if err != nil {
		t.Fatalf("读取聚合结果失败: %v", err)
	}
	if pv != 5 || uv != 3 || ips != 2 || sessions != 2 || bounces != 0 {
		t.Errorf("聚合结果错误: pv=%d uv=%d ips=%d sessions=%d bounces=%d", pv, uv, ips, sessions, bounces)
	}
}

func TestWithTxRollback(t *testing.T) {
	s := testStore(t)
	ensureApp(t, s, "800800")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wantErr := errors.New("boom")
	err := s.WithTx(ctx, func(tx *Tx) error {
		if err := tx.InsertRawEvents(ctx, []RawEvent{testEvent("800800", "evt-rollback", "v1")}); err != nil {
			return err
		}
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("WithTx 应透传业务错误，实际为 %v", err)
	}

	var count int
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM raw_events`).Scan(&count); err != nil {
		t.Fatalf("统计原始事件失败: %v", err)
	}
	if count != 0 {
		t.Errorf("事务回滚后 raw_events 应为 0 行，实际 %d 行", count)
	}
}

func TestLockVisitor(t *testing.T) {
	s := testStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 事务内可重入加锁。
	err := s.WithTx(ctx, func(tx *Tx) error {
		if err := tx.LockVisitor(ctx, "800800", "v1"); err != nil {
			return err
		}
		return tx.LockVisitor(ctx, "800800", "v1")
	})
	if err != nil {
		t.Fatalf("事务内咨询锁失败: %v", err)
	}
	// 池直连无事务语义，必须明确拒绝而非静默无锁。
	if err := s.LockVisitor(ctx, "800800", "v1"); err == nil {
		t.Error("池直连 LockVisitor 应返回错误，实际为 nil")
	}
}

func TestAdvisoryLockSerializes(t *testing.T) {
	s := testStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// tx1 持锁不提交，tx2 的同键加锁必须阻塞；tx1 提交后 tx2 通过。
	tx1, err := s.pool.Begin(ctx)
	if err != nil {
		t.Fatalf("开启事务失败: %v", err)
	}
	defer func() { _ = tx1.Rollback(ctx) }()
	if err := (&Tx{tx: tx1}).LockVisitor(ctx, "800800", "v1"); err != nil {
		t.Fatalf("tx1 加锁失败: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		tx2, err := s.pool.Begin(ctx)
		if err != nil {
			done <- err
			return
		}
		defer func() { _ = tx2.Rollback(ctx) }()
		if err := (&Tx{tx: tx2}).LockVisitor(ctx, "800800", "v1"); err != nil {
			done <- err
			return
		}
		done <- tx2.Commit(ctx)
	}()

	select {
	case err := <-done:
		t.Fatalf("tx2 不应立即通过（锁未生效），结果: %v", err)
	case <-time.After(300 * time.Millisecond):
	}

	if err := tx1.Commit(ctx); err != nil {
		t.Fatalf("tx1 提交失败: %v", err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("tx1 提交后 tx2 应通过，实际错误: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("tx1 提交后 tx2 仍阻塞")
	}
}

func TestSessionLifecycle(t *testing.T) {
	s := testStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	start := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)

	got, err := s.FindOpenSession(ctx, "800800", "v1", start.Add(-30*time.Minute))
	if err != nil || got != nil {
		t.Fatalf("无会话时应返回 (nil, nil)，实际 (%v, %v)", got, err)
	}

	sess := Session{SessionID: "sess-1", AppID: "800800", VisitorID: "v1",
		StartedAt: start, EndedAt: start, EventCount: 1, EntryURL: "/a", ExitURL: "/a"}
	if err := s.InsertSession(ctx, sess); err != nil {
		t.Fatalf("创建会话失败: %v", err)
	}
	got, err = s.FindOpenSession(ctx, "800800", "v1", start.Add(-30*time.Minute))
	if err != nil || got == nil || got.SessionID != "sess-1" {
		t.Fatalf("应找到未超时会话，实际 (%v, %v)", got, err)
	}

	newCount, err := s.ExtendSession(ctx, "sess-1", start.Add(5*time.Minute), "/b")
	if err != nil || newCount != 2 {
		t.Fatalf("延长会话应返回 count=2，实际 count=%d err=%v", newCount, err)
	}
	got, err = s.FindOpenSession(ctx, "800800", "v1", start.Add(60*time.Minute))
	if err != nil || got != nil {
		t.Fatalf("超时后应找不到会话，实际 (%v, %v)", got, err)
	}
}

func TestAppCRUD(t *testing.T) {
	s := testStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.CreateApp(ctx, App{AppID: "800801", SecretKey: "sk", Name: "demo", Domain: "demo.com"}); err != nil {
		t.Fatalf("创建应用失败: %v", err)
	}
	app, err := s.GetAppByID(ctx, "800801")
	if err != nil || app == nil || app.Domain != "demo.com" || app.Status != "active" {
		t.Fatalf("读取应用错误: (%v, %v)", app, err)
	}
	if err := s.SetAppStatus(ctx, "800801", "disabled"); err != nil {
		t.Fatalf("停用应用失败: %v", err)
	}
	app, err = s.GetAppByID(ctx, "800801")
	if err != nil || app.Status != "disabled" {
		t.Fatalf("应用状态应为 disabled: (%v, %v)", app, err)
	}
	if _, err := s.GetAppByID(ctx, "no-such-app"); !errors.Is(err, ErrNotFound) {
		t.Errorf("未知应用应返回 ErrNotFound，实际为 %v", err)
	}
}

func TestUserCRUD(t *testing.T) {
	s := testStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	u, err := s.CreateUser(ctx, "bob", "hash-bob")
	if err != nil || u.ID == 0 || u.Username != "bob" {
		t.Fatalf("创建用户错误: (%v, %v)", u, err)
	}
	got, err := s.GetUserByUsername(ctx, "bob")
	if err != nil || got.PasswordHash != "hash-bob" {
		t.Fatalf("读取用户错误: (%v, %v)", got, err)
	}
	if _, err := s.GetUserByUsername(ctx, "ghost"); !errors.Is(err, ErrNotFound) {
		t.Errorf("未知用户应返回 ErrNotFound，实际为 %v", err)
	}
}
