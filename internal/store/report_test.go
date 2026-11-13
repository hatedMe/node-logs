package store

import (
	"context"
	"testing"
	"time"
)

func TestReportQueries(t *testing.T) {
	s := testStore(t)
	ensureApp(t, s, "800800")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	day1 := time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC)

	if err := s.UpsertDailyStats(ctx, StatsDelta{AppID: "800800", Date: day1, PV: 10, UV: 5, IPCount: 4, Sessions: 3, Bounces: 1}); err != nil {
		t.Fatalf("种子数据失败: %v", err)
	}
	if err := s.UpsertDailyStats(ctx, StatsDelta{AppID: "800800", Date: day2, PV: 7, UV: 2, IPCount: 2, Sessions: 1}); err != nil {
		t.Fatalf("种子数据失败: %v", err)
	}
	if err := s.UpsertPageStats(ctx, "800800", day1, []PageDelta{{URL: "/a", PV: 5, UV: 3, Sessions: 2}, {URL: "/b", PV: 5, UV: 2, Sessions: 1}}); err != nil {
		t.Fatalf("种子数据失败: %v", err)
	}
	if err := s.UpsertPageStats(ctx, "800800", day2, []PageDelta{{URL: "/a", PV: 7, UV: 2, Sessions: 1}}); err != nil {
		t.Fatalf("种子数据失败: %v", err)
	}
	devices := []DeviceDelta{
		{Dimension: "browser", Value: "Chrome", PV: 8},
		{Dimension: "browser", Value: "Firefox", PV: 2},
		{Dimension: "system", Value: "Windows", PV: 9},
		{Dimension: "device", Value: "desktop", PV: 10},
	}
	if err := s.UpsertDeviceStats(ctx, "800800", day1, devices); err != nil {
		t.Fatalf("种子数据失败: %v", err)
	}

	ov, err := s.GetOverview(ctx, "800800", day1, day2)
	if err != nil {
		t.Fatalf("GetOverview 失败: %v", err)
	}
	if ov.PV != 17 || ov.UV != 7 || ov.IPCount != 6 || ov.Sessions != 4 || ov.Bounces != 1 {
		t.Errorf("总览错误: %+v", ov)
	}
	ov1, err := s.GetOverview(ctx, "800800", day1, day1)
	if err != nil || ov1.PV != 10 {
		t.Errorf("单日总览错误: (%+v, %v)", ov1, err)
	}
	ovEmpty, err := s.GetOverview(ctx, "no-such", day1, day2)
	if err != nil || ovEmpty.PV != 0 {
		t.Errorf("未知应用应返回零值: (%+v, %v)", ovEmpty, err)
	}

	trend, err := s.GetTrend(ctx, "800800", day1, day2)
	if err != nil {
		t.Fatalf("GetTrend 失败: %v", err)
	}
	if len(trend) != 2 || trend[0].PV != 10 || trend[1].PV != 7 {
		t.Fatalf("趋势错误: %+v", trend)
	}
	if trend[0].Date.Day() != 12 || trend[1].Date.Day() != 13 {
		t.Errorf("趋势应按日期升序: %+v", trend)
	}

	pages, total, err := s.GetPages(ctx, "800800", day1, day2, "pv", 10, 0)
	if err != nil {
		t.Fatalf("GetPages 失败: %v", err)
	}
	if total != 2 || len(pages) != 2 {
		t.Fatalf("页面排行错误: total=%d pages=%+v", total, pages)
	}
	if pages[0].URL != "/a" || pages[0].PV != 12 || pages[0].UV != 5 {
		t.Errorf("页面 /a 汇总错误: %+v", pages[0])
	}
	paged, total, err := s.GetPages(ctx, "800800", day1, day2, "pv", 1, 1)
	if err != nil || total != 2 || len(paged) != 1 || paged[0].URL != "/b" {
		t.Errorf("页面分页错误: total=%d pages=%+v err=%v", total, paged, err)
	}
	byUV, _, err := s.GetPages(ctx, "800800", day1, day2, "uv", 10, 0)
	if err != nil || byUV[0].URL != "/a" {
		t.Errorf("按 UV 排序错误: %+v err=%v", byUV, err)
	}

	devRows, err := s.GetDevices(ctx, "800800", day1, day1)
	if err != nil {
		t.Fatalf("GetDevices 失败: %v", err)
	}
	if len(devRows) != 4 {
		t.Fatalf("设备行数应为 4，实际 %+v", devRows)
	}
}
