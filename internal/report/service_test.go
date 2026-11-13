package report

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hatedMe/node-logs/internal/store"
)

type stubQuerier struct {
	overview store.OverviewRow
	trend    []store.TrendRow
	pages    []store.PageRow
	total    int
	devices  []store.DeviceRow
	err      error
}

func (s *stubQuerier) GetOverview(_ context.Context, _ string, _, _ time.Time) (store.OverviewRow, error) {
	return s.overview, s.err
}
func (s *stubQuerier) GetTrend(_ context.Context, _ string, _, _ time.Time) ([]store.TrendRow, error) {
	return s.trend, s.err
}
func (s *stubQuerier) GetPages(_ context.Context, _ string, _, _ time.Time, _ string, _, _ int) ([]store.PageRow, int, error) {
	return s.pages, s.total, s.err
}
func (s *stubQuerier) GetDevices(_ context.Context, _ string, _, _ time.Time) ([]store.DeviceRow, error) {
	return s.devices, s.err
}

func testService(q *stubQuerier) *Service {
	return &Service{DB: q, MaxRangeDays: 90}
}

func TestOverviewOK(t *testing.T) {
	svc := testService(&stubQuerier{overview: store.OverviewRow{PV: 10, UV: 5, IPCount: 4, Sessions: 3, Bounces: 1}})
	ov, err := svc.Overview(context.Background(), "800800", "2026-11-01", "2026-11-12")
	if err != nil {
		t.Fatalf("Overview 失败: %v", err)
	}
	if ov.PV != 10 || ov.UV != 5 || ov.IPCount != 4 || ov.Sessions != 3 {
		t.Errorf("总览指标错误: %+v", ov)
	}
	if ov.BounceRate == nil || *ov.BounceRate < 0.333 || *ov.BounceRate > 0.334 {
		t.Errorf("跳出率应为 1/3: %+v", ov.BounceRate)
	}
}

func TestOverviewNoSessions(t *testing.T) {
	svc := testService(&stubQuerier{})
	ov, err := svc.Overview(context.Background(), "800800", "2026-11-01", "2026-11-12")
	if err != nil {
		t.Fatalf("Overview 失败: %v", err)
	}
	if ov.BounceRate != nil {
		t.Errorf("无会话时跳出率应为 null: %+v", ov.BounceRate)
	}
}

func TestDateValidation(t *testing.T) {
	svc := testService(&stubQuerier{})
	cases := map[string][2]string{
		"非法 from":    {"abc", "2026-11-12"},
		"非法 to":      {"2026-11-01", "2026-13-40"},
		"from 晚于 to": {"2026-11-12", "2026-11-01"},
		"范围超限":       {"2026-01-01", "2026-12-31"},
	}
	for name, dates := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.Overview(context.Background(), "800800", dates[0], dates[1])
			var rerr *Error
			if !errors.As(err, &rerr) || rerr.Code != 400 {
				t.Errorf("%s 应返回 400，实际为 %v", name, err)
			}
		})
	}
}

func TestTrendOK(t *testing.T) {
	svc := testService(&stubQuerier{trend: []store.TrendRow{
		{Date: time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC), PV: 10, Sessions: 2, Bounces: 1},
		{Date: time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC), PV: 7},
	}})
	points, err := svc.Trend(context.Background(), "800800", "2026-11-12", "2026-11-13")
	if err != nil {
		t.Fatalf("Trend 失败: %v", err)
	}
	if len(points) != 2 || points[0].Date != "2026-11-12" || points[1].Date != "2026-11-13" {
		t.Fatalf("趋势日期错误: %+v", points)
	}
	if points[0].BounceRate == nil || *points[0].BounceRate != 0.5 {
		t.Errorf("趋势跳出率错误: %+v", points[0])
	}
	if points[1].BounceRate != nil {
		t.Errorf("无会话日跳出率应为 null: %+v", points[1])
	}
}

func TestPagesOK(t *testing.T) {
	svc := testService(&stubQuerier{
		pages: []store.PageRow{{URL: "/a", PV: 12, UV: 5, Sessions: 3}},
		total: 2,
	})
	res, err := svc.Pages(context.Background(), "800800", "2026-11-01", "2026-11-12", "pv", 1, 20)
	if err != nil {
		t.Fatalf("Pages 失败: %v", err)
	}
	if res.Total != 2 || len(res.Items) != 1 || res.Items[0].URL != "/a" {
		t.Errorf("页面结果错误: %+v", res)
	}
	if res.Page != 1 || res.PageSize != 20 {
		t.Errorf("分页回显错误: %+v", res)
	}
}

func TestPagesValidation(t *testing.T) {
	svc := testService(&stubQuerier{})
	cases := map[string]struct {
		sort     string
		page     int
		pageSize int
	}{
		"非法排序":  {sort: "xxx", page: 1, pageSize: 20},
		"页码为零":  {sort: "pv", page: 0, pageSize: 20},
		"页大小为零": {sort: "pv", page: 1, pageSize: 0},
		"页大小超限": {sort: "pv", page: 1, pageSize: 101},
		"非法日期":  {sort: "pv", page: 1, pageSize: 20},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			from, to := "2026-11-01", "2026-11-12"
			if name == "非法日期" {
				from = "bad"
			}
			_, err := svc.Pages(context.Background(), "800800", from, to, c.sort, c.page, c.pageSize)
			var rerr *Error
			if !errors.As(err, &rerr) || rerr.Code != 400 {
				t.Errorf("%s 应返回 400，实际为 %v", name, err)
			}
		})
	}
}

func TestDevicesGrouping(t *testing.T) {
	svc := testService(&stubQuerier{devices: []store.DeviceRow{
		{Dimension: "browser", Value: "Chrome", PV: 8},
		{Dimension: "system", Value: "Windows", PV: 8},
		{Dimension: "device", Value: "desktop", PV: 8},
	}})
	res, err := svc.Devices(context.Background(), "800800", "2026-11-01", "2026-11-12")
	if err != nil {
		t.Fatalf("Devices 失败: %v", err)
	}
	if len(res.Browser) != 1 || res.Browser[0].Value != "Chrome" || res.Browser[0].PV != 8 {
		t.Errorf("浏览器分组错误: %+v", res)
	}
	if len(res.System) != 1 || len(res.Device) != 1 {
		t.Errorf("系统/设备分组错误: %+v", res)
	}
}

func TestDevicesEmpty(t *testing.T) {
	svc := testService(&stubQuerier{})
	res, err := svc.Devices(context.Background(), "800800", "2026-11-01", "2026-11-12")
	if err != nil {
		t.Fatalf("Devices 失败: %v", err)
	}
	if res.Browser == nil || res.System == nil || res.Device == nil {
		t.Errorf("空数据时各维度应返回空数组: %+v", res)
	}
}

func TestStoreErrorPassthrough(t *testing.T) {
	svc := testService(&stubQuerier{err: errors.New("pg down")})
	if _, err := svc.Overview(context.Background(), "800800", "2026-11-01", "2026-11-12"); err == nil {
		t.Error("存储错误应透传，实际为 nil")
	}
}
