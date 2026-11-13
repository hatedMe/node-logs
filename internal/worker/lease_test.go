package worker

import (
	"context"
	"errors"
	"maps"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// fakeScriptClient 模拟租约/收割 Lua，供无 Redis 单测。
// 与真实 Lua 语义一致：单调用内原子（Go 函数调用不可交错）。
type fakeScriptClient struct {
	mu     sync.Mutex
	leases map[string]string
	idem   map[string]string
	fs     *fakeStream
	dlq    *fakeDLQClient
	seq    int
	// failEval 使接下来 N 次 Eval 返回错误。
	failEval int
}

func newFakeScriptClient(fs *fakeStream, dlq *fakeDLQClient) *fakeScriptClient {
	return &fakeScriptClient{leases: map[string]string{}, idem: map[string]string{}, fs: fs, dlq: dlq}
}

func (f *fakeScriptClient) Eval(ctx context.Context, script string, keys []string, args ...any) *redis.Cmd {
	f.mu.Lock()
	defer f.mu.Unlock()
	cmd := redis.NewCmd(ctx)
	if f.failEval > 0 {
		f.failEval--
		cmd.SetErr(errors.New("EVAL failed"))
		return cmd
	}
	switch script {
	case leaseAcquireScript:
		owner, _ := args[0].(string)
		for _, k := range keys {
			f.leases[k] = owner
		}
		cmd.SetVal(int64(len(keys)))
		return cmd
	case leaseRenewScript:
		owner, _ := args[0].(string)
		var n int64
		for _, k := range keys {
			if f.leases[k] == owner {
				n++
			}
		}
		cmd.SetVal(n)
		return cmd
	case reapScript:
		return f.mimicReap(ctx, cmd, keys, args)
	}
	cmd.SetErr(errUnknownScript(script))
	return cmd
}

// mimicReap 镜像收割 Lua 语义：租约检查 → 正文 → 幂等 → 写死信 → 确认。
// 单调用内原子，与真实 Lua 一致。
func (f *fakeScriptClient) mimicReap(ctx context.Context, cmd *redis.Cmd, keys []string, args []any) *redis.Cmd {
	leaseKey, idemKey := keys[0], keys[2]
	id, _ := args[0].(string)
	dlqStream, _ := args[2].(string)
	reason, _ := args[5].(string)
	attempts, _ := args[6].(string)
	failedAt, _ := args[7].(string)
	owner, _ := args[8].(string)

	if o, ok := f.leases[leaseKey]; ok && o != owner {
		cmd.SetVal(OutcomeLeased)
		return cmd
	}
	values := f.findBody(id)
	if values == nil {
		f.fs.forceAck(id)
		cmd.SetVal(OutcomeGone)
		return cmd
	}
	if _, ok := f.idem[idemKey]; ok {
		f.fs.forceAck(id)
		cmd.SetVal(OutcomeDup)
		return cmd
	}
	fields := make(map[string]any, len(values)+4)
	maps.Copy(fields, values)
	fields["source_id"] = id
	fields["attempts"] = attempts
	fields["reason"] = reason
	fields["failed_at"] = failedAt
	f.seq++
	dlqID := "dlq-id-" + strconv.Itoa(f.seq)
	f.dlq.append(fakeDLQCall{key: idemKey, stream: dlqStream, fields: fields})
	f.idem[idemKey] = dlqID
	f.fs.forceAck(id)
	_ = ctx
	cmd.SetVal(OutcomeReaped)
	return cmd
}

func (f *fakeScriptClient) findBody(id string) map[string]any {
	f.fs.mu.Lock()
	defer f.fs.mu.Unlock()
	for _, m := range f.fs.msgs {
		if m.id == id {
			return m.values
		}
	}
	return nil
}

func errUnknownScript(script string) error {
	if len(script) > 32 {
		script = script[:32]
	}
	return &fakeScriptError{script: script}
}

type fakeScriptError struct{ script string }

func (e *fakeScriptError) Error() string { return "unknown script: " + e.script }

func TestLeasesAcquireRenew(t *testing.T) {
	fs := newFakeStream()
	dlq := &fakeDLQClient{}
	leases := &Leases{Client: newFakeScriptClient(fs, dlq), Stream: "events:test", Owner: "worker-a", TTL: 30 * time.Second}
	ctx := context.Background()

	if err := leases.Acquire(ctx, []string{"1-0", "2-0"}); err != nil {
		t.Fatalf("Acquire 失败: %v", err)
	}
	n, err := leases.Renew(ctx, []string{"1-0", "2-0"})
	if err != nil {
		t.Fatalf("Renew 失败: %v", err)
	}
	if n != 2 {
		t.Errorf("同属主应续约 2 个，实际 %d 个", n)
	}

	other := &Leases{Client: leases.Client, Stream: "events:test", Owner: "worker-b", TTL: 30 * time.Second}
	n, err = other.Renew(ctx, []string{"1-0"})
	if err != nil {
		t.Fatalf("Renew 失败: %v", err)
	}
	if n != 0 {
		t.Errorf("异属主不得续约，实际续约 %d 个", n)
	}

	// 重新获取（投递转移属主）覆盖旧属主。
	if err := other.Acquire(ctx, []string{"1-0"}); err != nil {
		t.Fatalf("Acquire 失败: %v", err)
	}
	n, _ = other.Renew(ctx, []string{"1-0"})
	if n != 1 {
		t.Errorf("转移后新属主应可续约，实际 %d 个", n)
	}
}

func TestLeaseKey(t *testing.T) {
	if got := LeaseKey("events:test", "1-0"); got != "lease:events:test:1-0" {
		t.Errorf("租约键格式错误: %q", got)
	}
}
