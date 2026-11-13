package store

import (
	"context"
	"testing"
	"time"
)

func TestListApps(t *testing.T) {
	s := testStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.CreateApp(ctx, App{AppID: "800800", SecretKey: "sk", Name: "a"}); err != nil {
		t.Fatalf("创建应用失败: %v", err)
	}
	if err := s.CreateApp(ctx, App{AppID: "800801", SecretKey: "sk", Name: "b"}); err != nil {
		t.Fatalf("创建应用失败: %v", err)
	}

	apps, err := s.ListApps(ctx)
	if err != nil {
		t.Fatalf("ListApps 失败: %v", err)
	}
	if len(apps) != 2 {
		t.Fatalf("ListApps 应返回 2 个应用，实际 %d 个", len(apps))
	}
	ids := map[string]bool{apps[0].AppID: true, apps[1].AppID: true}
	if !ids["800800"] || !ids["800801"] {
		t.Errorf("ListApps 返回的应用 ID 错误: %v", apps)
	}
}
