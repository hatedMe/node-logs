package worker

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type fakePurgingStore struct {
	calls atomic.Int64
	days  atomic.Int64
	err   error
}

func (f *fakePurgingStore) PurgeRawEvents(ctx context.Context, days int) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	f.calls.Add(1)
	f.days.Store(int64(days))
	if f.err != nil {
		return 0, f.err
	}
	return 7, nil
}

func TestJanitorRunsAtStartThenTicks(t *testing.T) {
	store := &fakePurgingStore{}
	j := &RetentionJanitor{Store: store, RetentionDays: 30, Interval: 20 * time.Millisecond}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	j.Run(ctx)
	if got := store.calls.Load(); got < 2 {
		t.Fatalf("启动即清一次 + 周期触发，调用次数应 >=2，实际 %d", got)
	}
	if got := store.days.Load(); got != 30 {
		t.Fatalf("应透传保留天数 30，实际 %d", got)
	}
}

func TestJanitorDisabledWhenNonPositiveDays(t *testing.T) {
	store := &fakePurgingStore{}
	j := &RetentionJanitor{Store: store, RetentionDays: 0, Interval: time.Millisecond}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	j.Run(ctx)
	if got := store.calls.Load(); got != 0 {
		t.Fatalf("保留天数 <=0 时应禁用清理，实际调用 %d 次", got)
	}
}

func TestJanitorErrorDoesNotStop(t *testing.T) {
	store := &fakePurgingStore{err: errors.New("pg down")}
	j := &RetentionJanitor{Store: store, RetentionDays: 30, Interval: 20 * time.Millisecond}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	j.Run(ctx)
	if got := store.calls.Load(); got < 2 {
		t.Fatalf("单次失败不应停止后续清理，调用次数应 >=2，实际 %d", got)
	}
}
