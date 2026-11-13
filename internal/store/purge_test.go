package store

import (
	"context"
	"testing"
	"time"
)

func TestPurgeRawEvents(t *testing.T) {
	pool := testPool(t)
	s := &Store{pool: pool}
	ctx := context.Background()

	old := RawEvent{
		EventID: "evt-purge-old", AppID: "800800", VisitorID: "u1",
		URL: "/old", EventTime: time.Now().AddDate(0, 0, -40).UTC(),
		ReceivedAt: time.Now().UTC(), IPHash: "h1",
	}
	fresh := RawEvent{
		EventID: "evt-purge-new", AppID: "800800", VisitorID: "u1",
		URL: "/new", EventTime: time.Now().UTC(),
		ReceivedAt: time.Now().UTC(), IPHash: "h1",
	}
	if err := s.InsertRawEvents(ctx, []RawEvent{old, fresh}); err != nil {
		t.Fatalf("准备数据失败: %v", err)
	}

	deleted, err := s.PurgeRawEvents(ctx, 30)
	if err != nil {
		t.Fatalf("PurgeRawEvents 失败: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("应删除 1 条过期事件，实际删除 %d 条", deleted)
	}
	var remaining int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM raw_events`).Scan(&remaining); err != nil {
		t.Fatalf("统计剩余事件失败: %v", err)
	}
	if remaining != 1 {
		t.Fatalf("应剩余 1 条事件，实际 %d 条", remaining)
	}
}

func TestPurgeRawEventsNegativeDays(t *testing.T) {
	s := &Store{pool: nil}
	if _, err := s.PurgeRawEvents(context.Background(), -1); err == nil {
		t.Fatal("负保留天数应返回错误，实际为 nil")
	}
}
