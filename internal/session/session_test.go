package session

import (
	"context"
	"testing"
	"time"

	"github.com/hatedMe/node-logs/internal/store"
)

// fakeStore 模拟会话存储。
type fakeStore struct {
	sessions map[string]*store.Session
	extended []string
	locks    []string
}

func newFakeStore() *fakeStore {
	return &fakeStore{sessions: map[string]*store.Session{}}
}

func (f *fakeStore) LockVisitor(_ context.Context, appID, visitorID string) error {
	f.locks = append(f.locks, appID+"\x00"+visitorID)
	return nil
}

func (f *fakeStore) FindOpenSession(_ context.Context, appID, visitorID string, since time.Time) (*store.Session, error) {
	for _, s := range f.sessions {
		if s.AppID == appID && s.VisitorID == visitorID && !s.EndedAt.Before(since) {
			cp := *s
			return &cp, nil
		}
	}
	return nil, nil
}

func (f *fakeStore) InsertSession(_ context.Context, s store.Session) error {
	cp := s
	f.sessions[s.SessionID] = &cp
	return nil
}

func (f *fakeStore) ExtendSession(_ context.Context, sessionID string, endedAt time.Time, exitURL string) (int64, error) {
	s := f.sessions[sessionID]
	s.EndedAt = endedAt
	s.ExitURL = exitURL
	s.EventCount++
	f.extended = append(f.extended, sessionID)
	return s.EventCount, nil
}

func TestAssignNewSession(t *testing.T) {
	db := newFakeStore()
	tr := NewTracker(30 * time.Minute)
	ctx := context.Background()
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)

	a, err := tr.Assign(ctx, db, "800800", "v1", at, "/a")
	if err != nil {
		t.Fatalf("Assign 失败: %v", err)
	}
	if !a.IsNew || a.SessionID == "" {
		t.Errorf("首个事件应创建新会话: %+v", a)
	}
	if a.SessionsDelta != 1 || a.BounceDelta != 1 {
		t.Errorf("新会话应 sessions+1 bounces+1: %+v", a)
	}
	if !a.BounceDate.Equal(at) {
		t.Errorf("跳出归因日期应为事件日期: %+v", a)
	}
	if len(db.sessions) != 1 {
		t.Fatalf("应持久化 1 个会话，实际 %d", len(db.sessions))
	}
}

func TestAssignExtendSession(t *testing.T) {
	db := newFakeStore()
	tr := NewTracker(30 * time.Minute)
	ctx := context.Background()
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)

	first, err := tr.Assign(ctx, db, "800800", "v1", at, "/a")
	if err != nil {
		t.Fatalf("Assign 失败: %v", err)
	}
	second, err := tr.Assign(ctx, db, "800800", "v1", at.Add(5*time.Minute), "/b")
	if err != nil {
		t.Fatalf("Assign 失败: %v", err)
	}
	if second.IsNew || second.SessionID != first.SessionID {
		t.Errorf("5 分钟内应归属同一会话: %+v", second)
	}
	if second.SessionsDelta != 0 || second.BounceDelta != -1 {
		t.Errorf("第二事件应 bounces-1: %+v", second)
	}
	third, err := tr.Assign(ctx, db, "800800", "v1", at.Add(10*time.Minute), "/c")
	if err != nil {
		t.Fatalf("Assign 失败: %v", err)
	}
	if third.BounceDelta != 0 {
		t.Errorf("第三事件不应再调整跳出: %+v", third)
	}
}

func TestAssignSessionTimeout(t *testing.T) {
	db := newFakeStore()
	tr := NewTracker(30 * time.Minute)
	ctx := context.Background()
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)

	first, err := tr.Assign(ctx, db, "800800", "v1", at, "/a")
	if err != nil {
		t.Fatalf("Assign 失败: %v", err)
	}
	second, err := tr.Assign(ctx, db, "800800", "v1", at.Add(31*time.Minute), "/b")
	if err != nil {
		t.Fatalf("Assign 失败: %v", err)
	}
	if !second.IsNew || second.SessionID == first.SessionID {
		t.Errorf("31 分钟后应创建新会话: %+v", second)
	}
}

func TestAssignTimeoutBoundaryInclusive(t *testing.T) {
	db := newFakeStore()
	tr := NewTracker(30 * time.Minute)
	ctx := context.Background()
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)

	first, err := tr.Assign(ctx, db, "800800", "v1", at, "/a")
	if err != nil {
		t.Fatalf("Assign 失败: %v", err)
	}
	second, err := tr.Assign(ctx, db, "800800", "v1", at.Add(30*time.Minute), "/b")
	if err != nil {
		t.Fatalf("Assign 失败: %v", err)
	}
	if second.IsNew || second.SessionID != first.SessionID {
		t.Errorf("恰好 30 分钟应仍属同一会话: %+v", second)
	}
}

func TestAssignResumesPersistedSession(t *testing.T) {
	db := newFakeStore()
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	db.sessions["persisted"] = &store.Session{
		SessionID: "persisted", AppID: "800800", VisitorID: "v1",
		StartedAt: at, EndedAt: at, EventCount: 1, EntryURL: "/a", ExitURL: "/a",
	}
	// 新 Tracker（模拟新批次）应恢复库中未超时的会话。
	tr := NewTracker(30 * time.Minute)
	a, err := tr.Assign(context.Background(), db, "800800", "v1", at.Add(5*time.Minute), "/b")
	if err != nil {
		t.Fatalf("Assign 失败: %v", err)
	}
	if a.IsNew || a.SessionID != "persisted" {
		t.Errorf("应恢复持久化会话: %+v", a)
	}
	if a.BounceDelta != -1 {
		t.Errorf("恢复会话的第二事件应 bounces-1: %+v", a)
	}
	if !a.BounceDate.Equal(at) {
		t.Errorf("跳出归因日期应为会话开始日期: %+v", a)
	}
}

func TestAssignLocksVisitorOnSlowPath(t *testing.T) {
	db := newFakeStore()
	tr := NewTracker(30 * time.Minute)
	ctx := context.Background()
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)

	if _, err := tr.Assign(ctx, db, "800800", "v1", at, "/a"); err != nil {
		t.Fatalf("Assign 失败: %v", err)
	}
	// 批内命中不重复加锁。
	if _, err := tr.Assign(ctx, db, "800800", "v1", at.Add(5*time.Minute), "/b"); err != nil {
		t.Fatalf("Assign 失败: %v", err)
	}
	if len(db.locks) != 1 {
		t.Fatalf("同访客批内只应加锁 1 次，实际 %d 次", len(db.locks))
	}
	// 超时后重新走慢路径，应再次加锁。
	if _, err := tr.Assign(ctx, db, "800800", "v1", at.Add(60*time.Minute), "/c"); err != nil {
		t.Fatalf("Assign 失败: %v", err)
	}
	if len(db.locks) != 2 {
		t.Fatalf("超时后应再次加锁，实际共 %d 次", len(db.locks))
	}
}

func TestAssignIsolatesVisitors(t *testing.T) {
	db := newFakeStore()
	tr := NewTracker(30 * time.Minute)
	ctx := context.Background()
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)

	a, err := tr.Assign(ctx, db, "800800", "v1", at, "/a")
	if err != nil {
		t.Fatalf("Assign 失败: %v", err)
	}
	b, err := tr.Assign(ctx, db, "800800", "v2", at, "/a")
	if err != nil {
		t.Fatalf("Assign 失败: %v", err)
	}
	if b.IsNew == false || b.SessionID == a.SessionID {
		t.Errorf("不同访客应独立会话: %+v", b)
	}
}
