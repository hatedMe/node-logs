package store

import (
	"context"
	"testing"
	"time"
)

func TestInsertPageVisitorAndPageSessionIfNew(t *testing.T) {
	s := testStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	day := time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC)

	isNew, err := s.InsertPageVisitorIfNew(ctx, PageVisitorKey{AppID: "800800", Date: day, URL: "/a", VisitorID: "v1"})
	if err != nil || !isNew {
		t.Fatalf("首次页面访客应返回 new=true，实际 new=%v err=%v", isNew, err)
	}
	isNew, err = s.InsertPageVisitorIfNew(ctx, PageVisitorKey{AppID: "800800", Date: day, URL: "/a", VisitorID: "v1"})
	if err != nil || isNew {
		t.Fatalf("重复页面访客应返回 new=false，实际 new=%v err=%v", isNew, err)
	}
	// 同访客不同页面应分别计数。
	isNew, err = s.InsertPageVisitorIfNew(ctx, PageVisitorKey{AppID: "800800", Date: day, URL: "/b", VisitorID: "v1"})
	if err != nil || !isNew {
		t.Fatalf("同访客新页面应返回 new=true，实际 new=%v err=%v", isNew, err)
	}

	isNew, err = s.InsertPageSessionIfNew(ctx, PageSessionKey{AppID: "800800", Date: day, URL: "/a", SessionID: "s1"})
	if err != nil || !isNew {
		t.Fatalf("首次页面会话应返回 new=true，实际 new=%v err=%v", isNew, err)
	}
	isNew, err = s.InsertPageSessionIfNew(ctx, PageSessionKey{AppID: "800800", Date: day, URL: "/a", SessionID: "s1"})
	if err != nil || isNew {
		t.Fatalf("重复页面会话应返回 new=false，实际 new=%v err=%v", isNew, err)
	}
}

func TestUpsertPageAndDeviceStats(t *testing.T) {
	s := testStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	day := time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC)

	if err := s.UpsertPageStats(ctx, "800800", day, []PageDelta{{URL: "/a", PV: 3, UV: 2, Sessions: 1}}); err != nil {
		t.Fatalf("页面聚合更新失败: %v", err)
	}
	if err := s.UpsertPageStats(ctx, "800800", day, []PageDelta{{URL: "/a", PV: 2, UV: 1}}); err != nil {
		t.Fatalf("页面聚合增量更新失败: %v", err)
	}
	var pv, uv, sessions int64
	err := s.pool.QueryRow(ctx,
		`SELECT pv, uv, sessions FROM daily_page_stats WHERE app_id = $1 AND stat_date = $2 AND url = $3`,
		"800800", "2026-11-12", "/a").Scan(&pv, &uv, &sessions)
	if err != nil {
		t.Fatalf("读取页面聚合失败: %v", err)
	}
	if pv != 5 || uv != 3 || sessions != 1 {
		t.Errorf("页面聚合错误: pv=%d uv=%d sessions=%d", pv, uv, sessions)
	}

	devices := []DeviceDelta{
		{Dimension: "browser", Value: "Chrome", PV: 4},
		{Dimension: "system", Value: "Windows", PV: 4},
		{Dimension: "device", Value: "desktop", PV: 4},
	}
	if err := s.UpsertDeviceStats(ctx, "800800", day, devices); err != nil {
		t.Fatalf("设备聚合更新失败: %v", err)
	}
	if err := s.UpsertDeviceStats(ctx, "800800", day, devices); err != nil {
		t.Fatalf("设备聚合增量更新失败: %v", err)
	}
	var browserPV int64
	err = s.pool.QueryRow(ctx,
		`SELECT pv FROM daily_device_stats WHERE app_id = $1 AND stat_date = $2 AND dimension = $3 AND value = $4`,
		"800800", "2026-11-12", "browser", "Chrome").Scan(&browserPV)
	if err != nil {
		t.Fatalf("读取设备聚合失败: %v", err)
	}
	if browserPV != 8 {
		t.Errorf("设备聚合错误: browser Chrome pv=%d，应为 8", browserPV)
	}
}
