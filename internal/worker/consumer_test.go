package worker

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/hatedMe/node-logs/internal/queue"
	"github.com/hatedMe/node-logs/internal/store"
)

// fakeMsg 是内存 Stream 消息。
type fakeMsg struct {
	id         string
	values     map[string]any
	deliveries int64
	acked      bool
	idle       time.Duration
}

// fakeGhost 是正文已裁剪但 pending 残留的条目。
type fakeGhost struct {
	count int64
	idle  time.Duration
}

// fakeStream 是内存 Stream 实现，供消费逻辑单测。
type fakeStream struct {
	mu     sync.Mutex
	msgs   []*fakeMsg
	ghosts map[string]*fakeGhost
	groups map[string]bool
	seq    int64
}

func newFakeStream() *fakeStream {
	return &fakeStream{groups: map[string]bool{}, ghosts: map[string]*fakeGhost{}}
}

func (f *fakeStream) Add(values map[string]any) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	id := strconv.FormatInt(f.seq, 10) + "-0"
	f.msgs = append(f.msgs, &fakeMsg{id: id, values: values})
	return id
}

// AddGhost 添加一条无正文的 pending 条目。
func (f *fakeStream) AddGhost(id string, count int64, idle time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ghosts[id] = &fakeGhost{count: count, idle: idle}
}

func (f *fakeStream) Pending() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := len(f.ghosts)
	for _, m := range f.msgs {
		if m.deliveries > 0 && !m.acked {
			n++
		}
	}
	return n
}

// IsPending 报告指定消息是否仍处于 pending。
func (f *fakeStream) IsPending(id string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.ghosts[id]; ok {
		return true
	}
	for _, m := range f.msgs {
		if m.id == id {
			return m.deliveries > 0 && !m.acked
		}
	}
	return false
}

// SetIdle 设置消息的模拟空闲时长。
func (f *fakeStream) SetIdle(id string, idle time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, m := range f.msgs {
		if m.id == id {
			m.idle = idle
		}
	}
}

// forceAck 无条件确认（供收割模拟使用）。
func (f *fakeStream) forceAck(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.ghosts, id)
	for _, m := range f.msgs {
		if m.id == id {
			m.acked = true
		}
	}
}

func (f *fakeStream) XGroupCreateMkStream(ctx context.Context, stream, group, _ string) *redis.StatusCmd {
	f.mu.Lock()
	defer f.mu.Unlock()
	cmd := redis.NewStatusCmd(ctx)
	if f.groups[stream+"\x00"+group] {
		cmd.SetErr(errors.New("BUSYGROUP Consumer Group name already exists"))
		return cmd
	}
	f.groups[stream+"\x00"+group] = true
	cmd.SetVal("OK")
	return cmd
}

func (f *fakeStream) XReadGroup(ctx context.Context, args *redis.XReadGroupArgs) *redis.XStreamSliceCmd {
	f.mu.Lock()
	defer f.mu.Unlock()
	cmd := redis.NewXStreamSliceCmd(ctx)
	var out []redis.XMessage
	for _, m := range f.msgs {
		if m.deliveries == 0 && int64(len(out)) < args.Count {
			m.deliveries = 1
			m.idle = 0
			out = append(out, redis.XMessage{ID: m.id, Values: m.values})
		}
	}
	cmd.SetVal([]redis.XStream{{Stream: args.Streams[0], Messages: out}})
	return cmd
}

func (f *fakeStream) XAck(ctx context.Context, _ string, _ string, ids ...string) *redis.IntCmd {
	f.mu.Lock()
	defer f.mu.Unlock()
	cmd := redis.NewIntCmd(ctx)
	var n int64
	for _, id := range ids {
		if _, ok := f.ghosts[id]; ok {
			delete(f.ghosts, id)
			n++
		}
		for _, m := range f.msgs {
			if m.id == id && !m.acked {
				m.acked = true
				n++
			}
		}
	}
	cmd.SetVal(n)
	return cmd
}

func (f *fakeStream) XAutoClaim(ctx context.Context, args *redis.XAutoClaimArgs) *redis.XAutoClaimCmd {
	f.mu.Lock()
	defer f.mu.Unlock()
	cmd := redis.NewXAutoClaimCmd(ctx)
	var out []redis.XMessage
	for _, m := range f.msgs {
		if m.deliveries > 0 && !m.acked && m.idle >= args.MinIdle && int64(len(out)) < args.Count {
			m.deliveries++
			m.idle = 0
			out = append(out, redis.XMessage{ID: m.id, Values: m.values})
		}
	}
	cmd.SetVal(out, "0-0")
	return cmd
}

func (f *fakeStream) XPendingExt(ctx context.Context, args *redis.XPendingExtArgs) *redis.XPendingExtCmd {
	f.mu.Lock()
	defer f.mu.Unlock()
	cmd := redis.NewXPendingExtCmd(ctx)
	var out []redis.XPendingExt
	for id, g := range f.ghosts {
		if int64(len(out)) < args.Count {
			out = append(out, redis.XPendingExt{ID: id, RetryCount: g.count, Idle: g.idle})
		}
	}
	for _, m := range f.msgs {
		if m.deliveries > 0 && !m.acked && int64(len(out)) < args.Count {
			out = append(out, redis.XPendingExt{ID: m.id, RetryCount: m.deliveries, Idle: m.idle})
		}
	}
	cmd.SetVal(out)
	return cmd
}

// stubProcessor 模拟批处理。
type stubProcessor struct {
	calls int
	batch []store.RawEvent
	err   error
	failN int
}

func (s *stubProcessor) Process(_ context.Context, events []store.RawEvent) (ProcessResult, error) {
	s.calls++
	s.batch = append(s.batch, events...)
	if s.failN > 0 {
		s.failN--
		return ProcessResult{}, s.err
	}
	if s.err != nil && s.failN < 0 {
		return ProcessResult{}, s.err
	}
	return ProcessResult{Total: len(events), Inserted: len(events)}, nil
}

func testConsumer(fs *fakeStream, proc *stubProcessor) (*Consumer, *fakeDLQClient) {
	dlqClient := &fakeDLQClient{}
	scripts := newFakeScriptClient(fs, dlqClient)
	dlq := &DeadLetter{Client: dlqClient, Stream: "events:dlq", SourceStream: "events:test"}
	return &Consumer{
		Streams:     fs,
		Stream:      "events:test",
		Group:       "test-group",
		Name:        "test-consumer",
		BatchSize:   10,
		MaxAttempts: 2,
		Retry:       RetryPolicy{Base: time.Millisecond, Max: time.Millisecond},
		Processor:   proc,
		DLQ:         dlq,
		Leases:      &Leases{Client: scripts, Stream: "events:test", Owner: "test-consumer", TTL: 30 * time.Second},
		Reaper: &Reaper{
			Client: scripts, DLQ: dlq,
			MainStream: "events:test", Group: "test-group", Owner: "test-consumer",
		},
	}, dlqClient
}

func streamValues(t *testing.T, e store.RawEvent) map[string]any {
	t.Helper()
	values, err := queue.EncodeEvent(e)
	if err != nil {
		t.Fatalf("编码事件失败: %v", err)
	}
	return values
}

func TestTickAcksOnSuccess(t *testing.T) {
	fs := newFakeStream()
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	fs.Add(streamValues(t, batchEvent("e1", "800800", "v1", "/a", at)))
	fs.Add(streamValues(t, batchEvent("e2", "800800", "v1", "/b", at)))
	proc := &stubProcessor{}
	c, _ := testConsumer(fs, proc)

	if err := c.Tick(context.Background()); err != nil {
		t.Fatalf("Tick 失败: %v", err)
	}
	if proc.calls != 1 || len(proc.batch) != 2 {
		t.Errorf("应处理 1 批 2 条，实际 calls=%d batch=%d", proc.calls, len(proc.batch))
	}
	if fs.Pending() != 0 {
		t.Errorf("成功后不应残留 pending，实际 %d", fs.Pending())
	}
}

func TestTickNoAckOnFailureThenRedeliver(t *testing.T) {
	fs := newFakeStream()
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	id := fs.Add(streamValues(t, batchEvent("e1", "800800", "v1", "/a", at)))
	proc := &stubProcessor{err: errors.New("pg down"), failN: 1}
	c, dlqClient := testConsumer(fs, proc)

	if err := c.Tick(context.Background()); err == nil {
		t.Fatal("处理失败时 Tick 应返回错误，实际为 nil")
	}
	if fs.Pending() != 1 {
		t.Fatalf("失败后消息应保持 pending，实际 %d", fs.Pending())
	}
	if len(dlqClient.adds) != 0 {
		t.Fatalf("偶发失败不应进死信，实际 %d 条", len(dlqClient.adds))
	}

	// 第二轮：消息空闲超时后被认领并成功处理，只 ACK 一次。
	fs.SetIdle(id, 2*time.Minute)
	if err := c.Tick(context.Background()); err != nil {
		t.Fatalf("重试 Tick 失败: %v", err)
	}
	if fs.Pending() != 0 {
		t.Errorf("重试成功后不应残留 pending，实际 %d", fs.Pending())
	}
	if proc.calls != 2 {
		t.Errorf("处理器应被调用 2 次，实际 %d 次", proc.calls)
	}
}

func TestTickParseFailureToDLQ(t *testing.T) {
	fs := newFakeStream()
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	idBad := fs.Add(map[string]any{"app_id": "800800"}) // 缺少 event_id
	fs.Add(streamValues(t, batchEvent("e1", "800800", "v1", "/a", at)))
	proc := &stubProcessor{}
	c, dlqClient := testConsumer(fs, proc)

	if err := c.Tick(context.Background()); err != nil {
		t.Fatalf("Tick 失败: %v", err)
	}
	if len(proc.batch) != 1 || proc.batch[0].EventID != "e1" {
		t.Errorf("合法事件应被处理: %+v", proc.batch)
	}
	if len(dlqClient.adds) != 1 {
		t.Fatalf("非法消息应进死信 1 条，实际 %d 条", len(dlqClient.adds))
	}
	fields := dlqClient.adds[0].fields
	if fields["source_id"] != idBad {
		t.Errorf("死信应记录源记录 ID，实际为 %v", fields)
	}
	if fs.Pending() != 0 {
		t.Errorf("解析失败消息应被 ACK，不应残留 pending，实际 %d", fs.Pending())
	}
}

func TestTickPoisonToDLQ(t *testing.T) {
	fs := newFakeStream()
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	id := fs.Add(streamValues(t, batchEvent("e1", "800800", "v1", "/a", at)))
	proc := &stubProcessor{err: errors.New("always fails"), failN: -1}
	c, dlqClient := testConsumer(fs, proc)

	// MaxAttempts=2：前 2 轮失败重试，第 3 轮收割为死信（需空闲超时）。
	if err := c.Tick(context.Background()); err == nil {
		t.Fatal("第 1 轮应失败")
	}
	fs.SetIdle(id, 2*time.Minute)
	if err := c.Tick(context.Background()); err == nil {
		t.Fatal("第 2 轮应失败")
	}
	fs.SetIdle(id, 2*time.Minute)
	if err := c.Tick(context.Background()); err != nil {
		t.Fatalf("第 3 轮收割不应返回错误: %v", err)
	}
	if len(dlqClient.adds) != 1 {
		t.Fatalf("毒丸消息应进死信 1 条，实际 %d 条", len(dlqClient.adds))
	}
	fields := dlqClient.adds[0].fields
	if fields["source_id"] != id {
		t.Errorf("死信应记录源记录 ID，实际为 %v", fields)
	}
	if fields["attempts"] != "2" {
		t.Errorf("死信应记录投递次数，实际为 %v", fields)
	}
	if fs.Pending() != 0 {
		t.Errorf("收割后不应残留 pending，实际 %d", fs.Pending())
	}
	if proc.calls != 2 {
		t.Errorf("处理器应被调用 2 次（第 3 轮直接收割），实际 %d 次", proc.calls)
	}
}

func TestTickReapRetryDoesNotDuplicateDLQ(t *testing.T) {
	fs := newFakeStream()
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	id := fs.Add(streamValues(t, batchEvent("e1", "800800", "v1", "/a", at)))
	// 毒丸已达收割条件且空闲超时。
	fs.msgs[0].deliveries = 2
	fs.SetIdle(id, 2*time.Minute)
	proc := &stubProcessor{}
	dlqClient := &fakeDLQClient{}
	scripts := newFakeScriptClient(fs, dlqClient)
	c := &Consumer{
		Streams: fs, Stream: "events:test", Group: "test-group", Name: "test-consumer",
		BatchSize: 10, MaxAttempts: 2, Retry: RetryPolicy{Base: time.Millisecond, Max: time.Millisecond},
		Processor: proc,
		DLQ:       &DeadLetter{Client: dlqClient, Stream: "events:dlq", SourceStream: "events:test"},
		Leases:    &Leases{Client: scripts, Stream: "events:test", Owner: "test-consumer", TTL: 30 * time.Second},
		Reaper: &Reaper{
			Client:     scripts,
			DLQ:        &DeadLetter{Client: dlqClient, Stream: "events:dlq", SourceStream: "events:test"},
			MainStream: "events:test", Group: "test-group", Owner: "test-consumer",
		},
	}

	// 第 1 轮：收割脚本失败（模拟 XADD 成功后崩溃/XACK 失败）。
	scripts.failEval = 1
	if err := c.Tick(context.Background()); err == nil {
		t.Fatal("收割失败时 Tick 应返回错误，实际为 nil")
	}
	if !fs.IsPending(id) {
		t.Fatal("收割失败后消息应保持 pending")
	}
	// 第 2 轮：重试只产生一条死信。
	if err := c.Tick(context.Background()); err != nil {
		t.Fatalf("重试 Tick 失败: %v", err)
	}
	if len(dlqClient.adds) != 1 {
		t.Fatalf("重试只应产生 1 条死信，实际 %d 条", len(dlqClient.adds))
	}
	if fs.IsPending(id) {
		t.Fatal("重试成功后不应残留 pending")
	}
	if proc.calls != 0 {
		t.Errorf("收割路径不应调用处理器，实际 %d 次", proc.calls)
	}
}

func TestTickReapGhostAcked(t *testing.T) {
	fs := newFakeStream()
	fs.AddGhost("99-0", 5, 2*time.Minute)
	proc := &stubProcessor{}
	c, dlqClient := testConsumer(fs, proc)

	if err := c.Tick(context.Background()); err != nil {
		t.Fatalf("Tick 失败: %v", err)
	}
	if len(dlqClient.adds) != 0 {
		t.Fatalf("无正文不应写死信，实际 %d 条", len(dlqClient.adds))
	}
	if fs.IsPending("99-0") {
		t.Fatal("无正文应被确认清理，不应残留 pending")
	}
}

// gateProcessor 阻塞处理直到放行，用于模拟长事务。
type gateProcessor struct {
	started chan struct{}
	release chan struct{}
	calls   int
}

func (g *gateProcessor) Process(_ context.Context, events []store.RawEvent) (ProcessResult, error) {
	g.calls++
	select {
	case g.started <- struct{}{}:
	default:
	}
	<-g.release
	return ProcessResult{Total: len(events), Inserted: len(events)}, nil
}

func TestTickLongProcessingNotReaped(t *testing.T) {
	fs := newFakeStream()
	dlq := &fakeDLQClient{}
	scripts := newFakeScriptClient(fs, dlq)
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	id := fs.Add(streamValues(t, batchEvent("e1", "800800", "v1", "/a", at)))
	// 已投递 1 次：A 认领后计数达阈值并开始长处理。
	fs.msgs[0].deliveries = 1
	fs.SetIdle(id, 2*time.Minute)

	newConsumer := func(name string, proc Processor) *Consumer {
		return &Consumer{
			Streams: fs, Stream: "events:test", Group: "test-group", Name: name,
			BatchSize: 10, MaxAttempts: 2, Retry: RetryPolicy{Base: time.Millisecond, Max: time.Millisecond},
			Processor: proc,
			DLQ:       &DeadLetter{Client: dlq, Stream: "events:dlq", SourceStream: "events:test"},
			Leases:    &Leases{Client: scripts, Stream: "events:test", Owner: name, TTL: 30 * time.Second},
			Reaper: &Reaper{
				Client:     scripts,
				DLQ:        &DeadLetter{Client: dlq, Stream: "events:dlq", SourceStream: "events:test"},
				MainStream: "events:test", Group: "test-group", Owner: name,
			},
		}
	}
	procA := &gateProcessor{started: make(chan struct{}, 1), release: make(chan struct{})}
	procB := &stubProcessor{}
	cA := newConsumer("worker-a", procA)
	cB := newConsumer("worker-b", procB)

	// A 开始长处理。
	doneA := make(chan error, 1)
	go func() { doneA <- cA.Tick(context.Background()) }()
	select {
	case <-procA.started:
	case <-time.After(5 * time.Second):
		t.Fatal("worker-a 未开始处理")
	}
	// 模拟时间流逝：消息空闲超阈且计数达阈。
	fs.SetIdle(id, 2*time.Minute)

	// B 整轮 Tick：不得收割 A 处理中的消息。
	if err := cB.Tick(context.Background()); err != nil {
		t.Fatalf("worker-b Tick 失败: %v", err)
	}
	// 放行 A 并等待完成。
	close(procA.release)
	select {
	case err := <-doneA:
		if err != nil {
			t.Fatalf("worker-a Tick 失败: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("worker-a 未完成")
	}

	if len(dlq.adds) != 0 {
		t.Fatalf("处理中消息不得被收割，死信 %d 条", len(dlq.adds))
	}
	if fs.Pending() != 0 {
		t.Errorf("完成后不应残留 pending，实际 %d", fs.Pending())
	}
	if procA.calls != 1 || procB.calls != 1 {
		t.Errorf("双方各处理 1 次（重复处理由下游幂等吸收），实际 A=%d B=%d", procA.calls, procB.calls)
	}
}

func TestTickSkipsFreshPendingInReap(t *testing.T) {
	fs := newFakeStream()
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	id := fs.Add(streamValues(t, batchEvent("e1", "800800", "v1", "/a", at)))
	// 模拟其他 worker 刚刚投递（次数已达阈值但仍在处理中）：不得收割。
	fs.msgs[0].deliveries = 5
	proc := &stubProcessor{}
	c, dlqClient := testConsumer(fs, proc)

	if err := c.Tick(context.Background()); err != nil {
		t.Fatalf("Tick 失败: %v", err)
	}
	if len(dlqClient.adds) != 0 {
		t.Fatalf("处理中的消息不得进死信，实际 %d 条", len(dlqClient.adds))
	}
	if !fs.IsPending(id) {
		t.Fatal("处理中的消息应保持 pending")
	}
	if proc.calls != 0 {
		t.Errorf("未超时的 pending 不应被认领，实际调用 %d 次", proc.calls)
	}
}

func TestTickFailureAcksOnlyParseFailures(t *testing.T) {
	fs := newFakeStream()
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	// 合法与非法消息交错：失败时只能 ACK 解析失败者，合法者必须保持 pending。
	id1 := fs.Add(streamValues(t, batchEvent("e1", "800800", "v1", "/a", at)))
	idBad := fs.Add(map[string]any{"app_id": "800800"})
	id2 := fs.Add(streamValues(t, batchEvent("e2", "800800", "v1", "/b", at)))
	proc := &stubProcessor{err: errors.New("pg down"), failN: 1}
	c, dlqClient := testConsumer(fs, proc)

	if err := c.Tick(context.Background()); err == nil {
		t.Fatal("处理失败时 Tick 应返回错误，实际为 nil")
	}
	if len(dlqClient.adds) != 1 {
		t.Fatalf("解析失败消息应进死信 1 条，实际 %d 条", len(dlqClient.adds))
	}
	if !fs.IsPending(id1) || !fs.IsPending(id2) {
		t.Fatalf("合法事件必须保持 pending：e1=%v e2=%v", fs.IsPending(id1), fs.IsPending(id2))
	}
	if fs.IsPending(idBad) {
		t.Fatal("解析失败消息应被 ACK，不应保持 pending")
	}
}

func TestTickEmpty(t *testing.T) {
	fs := newFakeStream()
	proc := &stubProcessor{}
	c, _ := testConsumer(fs, proc)

	if err := c.Tick(context.Background()); err != nil {
		t.Fatalf("空轮询应成功: %v", err)
	}
	if proc.calls != 0 {
		t.Errorf("无消息时不应调用处理器，实际 %d 次", proc.calls)
	}
}

func TestConsumerNormalizeDefaults(t *testing.T) {
	c := &Consumer{}
	c.normalize()
	if c.BatchSize != 100 {
		t.Errorf("BatchSize 默认应为 100，实际为 %d", c.BatchSize)
	}
	if c.MaxAttempts != 5 {
		t.Errorf("MaxAttempts 默认应为 5，实际为 %d", c.MaxAttempts)
	}
	if c.Block != time.Second {
		t.Errorf("Block 默认应为 1s（避免零值空转），实际为 %v", c.Block)
	}
	if c.Retry.Base != 100*time.Millisecond || c.Retry.Max != 30*time.Second {
		t.Errorf("Retry 默认值错误: %+v", c.Retry)
	}
	if c.ClaimIdle != time.Minute {
		t.Errorf("ClaimIdle 默认应为 1m（避免误认领处理中消息），实际为 %v", c.ClaimIdle)
	}
	if c.ReapIdle != time.Minute {
		t.Errorf("ReapIdle 默认应为 1m（避免误转处理中消息），实际为 %v", c.ReapIdle)
	}
	// 显式配置不得被覆盖。
	custom := &Consumer{BatchSize: 7, MaxAttempts: 2, Block: 3 * time.Second, Retry: RetryPolicy{Base: time.Second, Max: time.Minute}, ClaimIdle: 2 * time.Second, ReapIdle: 3 * time.Second}
	custom.normalize()
	if custom.BatchSize != 7 || custom.MaxAttempts != 2 || custom.Block != 3*time.Second {
		t.Errorf("显式配置不应被覆盖: %+v", custom)
	}
	if custom.ClaimIdle != 2*time.Second || custom.ReapIdle != 3*time.Second {
		t.Errorf("显式 idle 配置不应被覆盖: %+v", custom)
	}
}

func TestRetryPolicy(t *testing.T) {
	p := RetryPolicy{Base: 100 * time.Millisecond, Max: 250 * time.Millisecond}
	if got := p.Next(1); got != 100*time.Millisecond {
		t.Errorf("Next(1) 应为 100ms，实际为 %v", got)
	}
	if got := p.Next(2); got != 200*time.Millisecond {
		t.Errorf("Next(2) 应为 200ms，实际为 %v", got)
	}
	if got := p.Next(5); got != 250*time.Millisecond {
		t.Errorf("Next(5) 应封顶 250ms，实际为 %v", got)
	}
}
