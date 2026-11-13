package worker

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func testReaper(dlq *fakeDLQClient, scripts *fakeScriptClient) *Reaper {
	return &Reaper{
		Client: scripts,
		DLQ: &DeadLetter{
			Client: dlq, Stream: "events:dlq", SourceStream: "events:test", MaxLen: 10000,
		},
		MainStream: "events:test",
		Group:      "test-group",
		Owner:      "worker-a",
	}
}

func TestReapOutcomes(t *testing.T) {
	fs := newFakeStream()
	dlq := &fakeDLQClient{}
	r := testReaper(dlq, newFakeScriptClient(fs, dlq))
	ctx := context.Background()

	id := fs.Add(map[string]any{"event_id": "e1"})
	fs.msgs[0].deliveries = 2 // 模拟已投递待定
	// 他人租约持有中：跳过。
	r.Client.(*fakeScriptClient).leases[LeaseKey("events:test", id)] = "worker-b"
	out, err := r.Reap(ctx, id, 5, "poison")
	if err != nil {
		t.Fatalf("Reap 失败: %v", err)
	}
	if out != OutcomeLeased {
		t.Fatalf("他人租约应返回 leased，实际为 %q", out)
	}
	if len(dlq.adds) != 0 || !fs.IsPending(id) {
		t.Fatalf("leased 不应写死信或确认：adds=%d pending=%v", len(dlq.adds), fs.IsPending(id))
	}

	// 自身租约不阻塞收割（本 worker 的旧毒丸）。
	r.Client.(*fakeScriptClient).leases[LeaseKey("events:test", id)] = "worker-a"
	out, err = r.Reap(ctx, id, 5, "poison")
	if err != nil {
		t.Fatalf("Reap 失败: %v", err)
	}
	if out != OutcomeReaped {
		t.Fatalf("自身租约应正常收割，实际为 %q", out)
	}
	if len(dlq.adds) != 1 {
		t.Fatalf("应写死信 1 条，实际 %d 条", len(dlq.adds))
	}
	fields := dlq.adds[0].fields
	if fields["source_id"] != id || fields["attempts"] != "5" || fields["reason"] != "poison" {
		t.Errorf("死信字段错误: %v", fields)
	}
	if fs.IsPending(id) {
		t.Fatal("收割后不应残留 pending")
	}

	// 重复收割：幂等去重。
	out, err = r.Reap(ctx, id, 5, "poison")
	if err != nil {
		t.Fatalf("Reap 失败: %v", err)
	}
	if out != OutcomeDup {
		t.Fatalf("重复收割应返回 dup，实际为 %q", out)
	}
	if len(dlq.adds) != 1 {
		t.Fatalf("重复收割不得重复写死信，实际 %d 条", len(dlq.adds))
	}
}

func TestReapGhost(t *testing.T) {
	fs := newFakeStream()
	dlq := &fakeDLQClient{}
	r := testReaper(dlq, newFakeScriptClient(fs, dlq))
	ctx := context.Background()

	fs.AddGhost("99-0", 5, 2*time.Minute)
	out, err := r.Reap(ctx, "99-0", 5, "poison")
	if err != nil {
		t.Fatalf("Reap 失败: %v", err)
	}
	if out != OutcomeGone {
		t.Fatalf("无正文应返回 gone，实际为 %q", out)
	}
	if len(dlq.adds) != 0 {
		t.Fatalf("无正文不应写死信，实际 %d 条", len(dlq.adds))
	}
	if fs.IsPending("99-0") {
		t.Fatal("无正文应被确认清理，不应残留 pending")
	}
}

func TestReapIntegration(t *testing.T) {
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("未设置 TEST_REDIS_ADDR，跳过收割集成测试")
	}
	client := redis.NewClient(&redis.Options{Addr: addr, Password: os.Getenv("TEST_REDIS_PASSWORD")})
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stream := "events:reap-test"
	group := "reap-group"
	dlqStream := "events:reap-test:dlq"
	r := &Reaper{
		Client:     client,
		DLQ:        &DeadLetter{Client: client, Stream: dlqStream, SourceStream: stream, MaxLen: 10000},
		MainStream: stream,
		Group:      group,
		Owner:      "worker-a",
	}
	_ = client.Del(ctx, stream, dlqStream).Err()
	_ = client.XGroupDestroy(ctx, stream, group).Err()
	if err := client.XGroupCreateMkStream(ctx, stream, group, "0").Err(); err != nil {
		t.Fatalf("建组失败: %v", err)
	}

	newPending := func(t *testing.T) string {
		t.Helper()
		id, err := client.XAdd(ctx, &redis.XAddArgs{Stream: stream, Values: map[string]any{"event_id": "e1"}}).Result()
		if err != nil {
			t.Fatalf("XADD 失败: %v", err)
		}
		res, err := client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group: group, Consumer: "worker-a", Streams: []string{stream, ">"}, Count: 10,
		}).Result()
		if err != nil || len(res) == 0 || len(res[0].Messages) == 0 {
			t.Fatalf("认领失败: %v %+v", err, res)
		}
		return id
	}

	// 正常收割。
	id1 := newPending(t)
	out, err := r.Reap(ctx, id1, 5, "poison")
	if err != nil {
		t.Fatalf("Reap 失败: %v", err)
	}
	if out != OutcomeReaped {
		t.Fatalf("应收割，实际为 %q", out)
	}
	if n, _ := client.XLen(ctx, dlqStream).Result(); n != 1 {
		t.Fatalf("死信流应有 1 条，实际 %d 条", n)
	}
	// 重复收割幂等。
	out, err = r.Reap(ctx, id1, 5, "poison")
	if err != nil || out != OutcomeDup {
		t.Fatalf("重复收割应返回 dup，实际 (%q, %v)", out, err)
	}
	if n, _ := client.XLen(ctx, dlqStream).Result(); n != 1 {
		t.Fatalf("重复收割不得重复写，实际 %d 条", n)
	}

	// 他人租约：跳过。
	id2 := newPending(t)
	leases := &Leases{Client: client, Stream: stream, Owner: "worker-b", TTL: 30 * time.Second}
	if err := leases.Acquire(ctx, []string{id2}); err != nil {
		t.Fatalf("Acquire 失败: %v", err)
	}
	out, err = r.Reap(ctx, id2, 5, "poison")
	if err != nil || out != OutcomeLeased {
		t.Fatalf("他人租约应跳过，实际 (%q, %v)", out, err)
	}
	// 自身租约：不阻塞。
	leasesA := &Leases{Client: client, Stream: stream, Owner: "worker-a", TTL: 30 * time.Second}
	if err := leasesA.Acquire(ctx, []string{id2}); err != nil {
		t.Fatalf("Acquire 失败: %v", err)
	}
	out, err = r.Reap(ctx, id2, 5, "poison")
	if err != nil || out != OutcomeReaped {
		t.Fatalf("自身租约应收割，实际 (%q, %v)", out, err)
	}

	// 无正文：清理 pending。
	id3 := newPending(t)
	if _, err := client.XDel(ctx, stream, id3).Result(); err != nil {
		t.Fatalf("XDel 失败: %v", err)
	}
	out, err = r.Reap(ctx, id3, 5, "poison")
	if err != nil || out != OutcomeGone {
		t.Fatalf("无正文应清理，实际 (%q, %v)", out, err)
	}

	// 清理。
	_ = client.Del(ctx, stream, dlqStream).Err()
	_ = client.Del(ctx,
		"dlq:idem:"+stream+":"+id1,
		"dlq:idem:"+stream+":"+id2,
		"lease:"+stream+":"+id2,
	).Err()
}
