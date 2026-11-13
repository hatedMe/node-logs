package queue

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"

	"github.com/hatedMe/node-logs/internal/store"
)

func testEvent() store.RawEvent {
	return store.RawEvent{
		EventID:    "evt-queue-1",
		AppID:      "800800",
		VisitorID:  "visitor-1",
		SessionID:  "sess-1",
		URL:        "/home",
		Arg:        "?a=1",
		Version:    "1.0.0",
		Screen:     "1920x1080",
		Lang:       "zh-cn",
		TZ:         "-8",
		Domain:     "example.com",
		EventTime:  time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC),
		ReceivedAt: time.Date(2026, 11, 12, 10, 0, 1, 0, time.UTC),
		UA:         "test-agent",
		IPHash:     "iphash-1",
		Extra:      map[string]string{"ty": "0", "random": "abc"},
	}
}

func testProducer(t *testing.T) (*StreamProducer, func()) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("启动 miniredis 失败: %v", err)
	}
	p, err := NewStreamProducer(mr.Addr(), "", 0, "events:test", 2*time.Second)
	if err != nil {
		mr.Close()
		t.Fatalf("创建 StreamProducer 失败: %v", err)
	}
	return p, func() {
		_ = p.Close()
		mr.Close()
	}
}

func TestPublishRoundTrip(t *testing.T) {
	p, cleanup := testProducer(t)
	defer cleanup()
	ctx := context.Background()

	id, err := p.Publish(ctx, testEvent())
	if err != nil {
		t.Fatalf("Publish 失败: %v", err)
	}
	if id == "" {
		t.Fatal("Publish 应返回 Stream 记录 ID")
	}

	length, err := p.Len(ctx)
	if err != nil {
		t.Fatalf("Len 失败: %v", err)
	}
	if length != 1 {
		t.Fatalf("队列长度应为 1，实际为 %d", length)
	}

	msgs, err := p.client.XRange(ctx, "events:test", "-", "+").Result()
	if err != nil {
		t.Fatalf("XRange 失败: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("应读回 1 条消息，实际 %d 条", len(msgs))
	}
	got, err := ParseEvent(msgs[0].Values)
	if err != nil {
		t.Fatalf("ParseEvent 失败: %v", err)
	}
	want := testEvent()
	if got.EventID != want.EventID || got.AppID != want.AppID || got.VisitorID != want.VisitorID {
		t.Errorf("回读事件标识错误: %+v", got)
	}
	if got.URL != want.URL || got.Arg != want.Arg || got.Domain != want.Domain {
		t.Errorf("回读页面字段错误: %+v", got)
	}
	if !got.EventTime.Equal(want.EventTime) || !got.ReceivedAt.Equal(want.ReceivedAt) {
		t.Errorf("回读时间错误: %v / %v", got.EventTime, got.ReceivedAt)
	}
	if got.UA != want.UA || got.IPHash != want.IPHash {
		t.Errorf("回读服务端字段错误: %+v", got)
	}
	if got.Extra["ty"] != "0" || got.Extra["random"] != "abc" {
		t.Errorf("回读扩展字段错误: %v", got.Extra)
	}
}

func TestParseEventMissingFields(t *testing.T) {
	if _, err := ParseEvent(map[string]any{"app_id": "800800"}); err == nil {
		t.Fatal("缺失 event_id 时应返回错误，实际为 nil")
	}
	if _, err := ParseEvent(map[string]any{"event_id": "e1"}); err == nil {
		t.Fatal("缺失 app_id 时应返回错误，实际为 nil")
	}
	if _, err := ParseEvent(map[string]any{"event_id": "e1", "app_id": "a", "event_time_ms": "abc"}); err == nil {
		t.Fatal("非法 event_time_ms 时应返回错误，实际为 nil")
	}
}

func TestPoolStats(t *testing.T) {
	p, cleanup := testProducer(t)
	defer cleanup()

	if _, err := p.Publish(context.Background(), testEvent()); err != nil {
		t.Fatalf("Publish 失败: %v", err)
	}
	stats := p.PoolStats()
	if stats == nil {
		t.Fatal("PoolStats 不应返回 nil")
	}
}

func TestPublishIntegrationRealRedis(t *testing.T) {
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("未设置 TEST_REDIS_ADDR，跳过真实 Redis 集成测试")
	}
	p, err := NewStreamProducer(addr, os.Getenv("TEST_REDIS_PASSWORD"), 0, "events:integration-test", 2*time.Second)
	if err != nil {
		t.Fatalf("连接真实 Redis 失败: %v", err)
	}
	defer func() { _ = p.Close() }()
	ctx := context.Background()

	event := testEvent()
	event.EventID = "evt-integration-1"
	if _, err := p.Publish(ctx, event); err != nil {
		t.Fatalf("真实 Redis Publish 失败: %v", err)
	}
	length, err := p.Len(ctx)
	if err != nil || length < 1 {
		t.Fatalf("真实 Redis 队列长度异常: length=%d err=%v", length, err)
	}
	// 清理集成测试写入。
	if err := p.client.Del(ctx, "events:integration-test").Err(); err != nil {
		t.Fatalf("清理集成测试流失败: %v", err)
	}
}
