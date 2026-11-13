package collector

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hatedMe/node-logs/internal/store"
	"github.com/hatedMe/node-logs/internal/testdb"
)

// TestCachingAppValidatorWithRealStore 验证停用应用不能继续写入事件：
// 应用停用后，缓存刷新后校验必须返回 ErrAppDisabled。
func TestCachingAppValidatorWithRealStore(t *testing.T) {
	dsn := testdb.Prepare(t, "nodelogs_test_collector")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := store.New(ctx, dsn)
	if err != nil {
		t.Fatalf("创建 Store 失败: %v", err)
	}
	defer db.Close()

	if err := db.CreateApp(ctx, store.App{AppID: "800800", SecretKey: "sk", Name: "t"}); err != nil {
		t.Fatalf("预置应用失败: %v", err)
	}

	v := NewCachingAppValidator(db, time.Millisecond)
	if err := v.ValidateApp(ctx, "800800"); err != nil {
		t.Fatalf("启用应用应通过校验: %v", err)
	}

	if err := db.SetAppStatus(ctx, "800800", "disabled"); err != nil {
		t.Fatalf("停用应用失败: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	if err := v.ValidateApp(ctx, "800800"); !errors.Is(err, ErrAppDisabled) {
		t.Fatalf("停用应用应返回 ErrAppDisabled，实际为 %v", err)
	}
}
