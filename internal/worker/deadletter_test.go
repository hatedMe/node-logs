package worker

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// fakeDLQCall 记录一次死信 Lua 调用。
type fakeDLQCall struct {
	key    string
	stream string
	maxlen string
	ttl    string
	fields map[string]any
}

// fakeDLQClient 模拟死信 Lua（含幂等语义，镜像真实脚本）；方法均加锁。
type fakeDLQClient struct {
	mu   sync.Mutex
	adds []fakeDLQCall
	seen map[string]string
	seq  int
}

func (f *fakeDLQClient) Eval(ctx context.Context, _ string, keys []string, args ...any) *redis.Cmd {
	f.mu.Lock()
	defer f.mu.Unlock()
	cmd := redis.NewCmd(ctx)
	if id, ok := f.seen[keys[0]]; ok {
		cmd.SetVal(id)
		return cmd
	}
	stream, _ := args[0].(string)
	ttl, _ := args[1].(string)
	maxlen, _ := args[2].(string)
	fields := map[string]any{}
	for i := 3; i+1 < len(args); i += 2 {
		k, _ := args[i].(string)
		fields[k] = args[i+1]
	}
	f.seq++
	id := fmt.Sprintf("dlq-id-%d", f.seq)
	f.adds = append(f.adds, fakeDLQCall{key: keys[0], stream: stream, maxlen: maxlen, ttl: ttl, fields: fields})
	if f.seen == nil {
		f.seen = map[string]string{}
	}
	f.seen[keys[0]] = id
	cmd.SetVal(id)
	return cmd
}

// append 供收割模拟写入（调用方已持有脚本锁，此处只取自身锁；顺序固定无死锁）。
func (f *fakeDLQClient) append(call fakeDLQCall) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.adds = append(f.adds, call)
}

func TestDeadLetterWriteRaw(t *testing.T) {
	client := &fakeDLQClient{}
	dlq := &DeadLetter{Client: client, Stream: "events:dlq", SourceStream: "events:test", MaxLen: 10000}

	values := map[string]any{"event_id": "e1", "app_id": "800800", "url": "/a"}
	if err := dlq.WriteRaw(context.Background(), values, "1-0", "解析失败：缺少字段"); err != nil {
		t.Fatalf("WriteRaw 失败: %v", err)
	}
	if len(client.adds) != 1 {
		t.Fatalf("应写入 1 条死信，实际 %d 条", len(client.adds))
	}
	got := client.adds[0]
	if got.stream != "events:dlq" {
		t.Errorf("死信流错误: %q", got.stream)
	}
	if got.key != "dlq:idem:events:test:1-0" {
		t.Errorf("幂等键应按源流作用域，实际为 %q", got.key)
	}
	if got.fields["event_id"] != "e1" || got.fields["app_id"] != "800800" || got.fields["url"] != "/a" {
		t.Errorf("死信应保留原始字段: %v", got.fields)
	}
	if got.fields["reason"] != "解析失败：缺少字段" {
		t.Errorf("死信应记录原因: %v", got.fields)
	}
	if got.fields["failed_at"] == "" || got.fields["failed_at"] == nil {
		t.Errorf("死信应记录失败时间: %v", got.fields)
	}
	if got.fields["source_id"] != "1-0" {
		t.Errorf("死信应记录来源 ID: %v", got.fields)
	}
}

func TestMaxLenArg(t *testing.T) {
	cases := map[int64]string{
		-1:    "0",
		-100:  "0",
		0:     "0",
		5:     "5",
		10000: "10000",
	}
	for in, want := range cases {
		if got := maxLenArg(in); got != want {
			t.Errorf("maxLenArg(%d) 应为 %q，实际为 %q", in, want, got)
		}
	}
}

func TestDeadLetterNegativeMaxLenUnlimited(t *testing.T) {
	client := &fakeDLQClient{}
	dlq := &DeadLetter{Client: client, Stream: "events:dlq", SourceStream: "events:test", MaxLen: -1}
	if err := dlq.WriteRaw(context.Background(), map[string]any{"event_id": "e1"}, "1-0", "boom"); err != nil {
		t.Fatalf("WriteRaw 失败: %v", err)
	}
	if len(client.adds) != 1 {
		t.Fatalf("应写入 1 条，实际 %d 条", len(client.adds))
	}
	if client.adds[0].maxlen != "0" {
		t.Errorf("负 MaxLen 应转为不限长 0，实际传参 %q", client.adds[0].maxlen)
	}
}

func TestDeadLetterTTLDefaults(t *testing.T) {
	client := &fakeDLQClient{}
	dlq := &DeadLetter{Client: client, Stream: "events:dlq", SourceStream: "events:test"}
	if err := dlq.WriteRaw(context.Background(), map[string]any{"event_id": "e1"}, "1-0", "boom"); err != nil {
		t.Fatalf("WriteRaw 失败: %v", err)
	}
	// 默认 30 天（2592000 秒），与业务原始事件保留周期对齐。
	if client.adds[0].ttl != "2592000" {
		t.Errorf("默认 TTL 应为 2592000 秒，实际为 %q", client.adds[0].ttl)
	}

	custom := &DeadLetter{Client: client, Stream: "events:dlq", SourceStream: "events:test", IdemTTL: time.Hour}
	if err := custom.WriteRaw(context.Background(), map[string]any{"event_id": "e2"}, "2-0", "boom"); err != nil {
		t.Fatalf("WriteRaw 失败: %v", err)
	}
	if client.adds[1].ttl != "3600" {
		t.Errorf("自定义 TTL 应透传 3600 秒，实际为 %q", client.adds[1].ttl)
	}
}

func TestDeadLetterIdempotent(t *testing.T) {
	client := &fakeDLQClient{}
	dlq := &DeadLetter{Client: client, Stream: "events:dlq"}

	values := map[string]any{"event_id": "e1"}
	for range 3 {
		if err := dlq.WriteRaw(context.Background(), values, "1-0", "boom"); err != nil {
			t.Fatalf("WriteRaw 失败: %v", err)
		}
	}
	if len(client.adds) != 1 {
		t.Fatalf("同一 source_id 重试只应产生 1 条死信，实际 %d 条", len(client.adds))
	}
	if err := dlq.WriteRaw(context.Background(), values, "2-0", "boom"); err != nil {
		t.Fatalf("WriteRaw 失败: %v", err)
	}
	if len(client.adds) != 2 {
		t.Fatalf("不同 source_id 应新增死信，实际 %d 条", len(client.adds))
	}
}

func TestDeadLetterIdempotentIntegration(t *testing.T) {
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("未设置 TEST_REDIS_ADDR，跳过死信幂等集成测试")
	}
	client := redis.NewClient(&redis.Options{Addr: addr, Password: os.Getenv("TEST_REDIS_PASSWORD")})
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stream := "events:dlq-idem-test"
	dlq := &DeadLetter{Client: client, Stream: stream, SourceStream: "events:idem-src", MaxLen: 10000}
	values := map[string]any{"event_id": "e1", "app_id": "800800"}
	uniq := time.Now().UnixNano()
	src1 := fmt.Sprintf("idem-%d-1", uniq)
	src2 := fmt.Sprintf("idem-%d-2", uniq)

	// 串行重试同一 source。
	for range 3 {
		if err := dlq.WriteRaw(ctx, values, src1, "boom"); err != nil {
			t.Fatalf("WriteRaw 失败: %v", err)
		}
	}
	// 并发同一 source。
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = dlq.WriteRaw(ctx, values, src1, "boom")
		}()
	}
	wg.Wait()
	// 不同 source 新增。
	if err := dlq.WriteRaw(ctx, values, src2, "boom"); err != nil {
		t.Fatalf("WriteRaw 失败: %v", err)
	}

	n, err := client.XLen(ctx, stream).Result()
	if err != nil {
		t.Fatalf("XLEN 失败: %v", err)
	}
	if n != 2 {
		t.Errorf("串行+并发重试后应只有 2 条死信，实际 %d 条", n)
	}
	msgs, err := client.XRange(ctx, stream, "-", "+").Result()
	if err != nil {
		t.Fatalf("XRange 失败: %v", err)
	}
	seen := map[string]int{}
	for _, m := range msgs {
		if sid, ok := m.Values["source_id"].(string); ok {
			seen[sid]++
		}
	}
	if seen[src1] != 1 || seen[src2] != 1 {
		t.Errorf("各 source 应恰好 1 条: %v", seen)
	}

	// 清理流与幂等键（避免 7 天 TTL 影响重跑）。
	_ = client.Del(ctx, stream).Err()
	_ = client.Del(ctx, "dlq:idem:events:idem-src:"+src1, "dlq:idem:events:idem-src:"+src2).Err()
}
