// Package report 实现只读聚合表的报表查询。
package report

import (
	"context"
	"time"

	"github.com/hatedMe/node-logs/internal/store"
)

// Querier 是报表查询依赖的存储子集；*store.Store 已实现。
type Querier interface {
	GetOverview(ctx context.Context, appID string, from, to time.Time) (store.OverviewRow, error)
	GetTrend(ctx context.Context, appID string, from, to time.Time) ([]store.TrendRow, error)
	GetPages(ctx context.Context, appID string, from, to time.Time, sort string, limit, offset int) ([]store.PageRow, int, error)
	GetDevices(ctx context.Context, appID string, from, to time.Time) ([]store.DeviceRow, error)
}

// Error 是携带 HTTP 状态码的报表错误。
type Error struct {
	Code int
	Msg  string
}

func (e *Error) Error() string { return e.Msg }

// Service 实现报表查询业务。
type Service struct {
	DB           Querier
	MaxRangeDays int
}

// Overview 是总览响应。
type Overview struct {
	AppID      string   `json:"app_id"`
	From       string   `json:"from"`
	To         string   `json:"to"`
	PV         int64    `json:"pv"`
	UV         int64    `json:"uv"`
	IPCount    int64    `json:"ip_count"`
	Sessions   int64    `json:"sessions"`
	Bounces    int64    `json:"bounces"`
	BounceRate *float64 `json:"bounce_rate"`
}

// TrendPoint 是趋势响应点。
type TrendPoint struct {
	Date       string   `json:"date"`
	PV         int64    `json:"pv"`
	UV         int64    `json:"uv"`
	IPCount    int64    `json:"ip_count"`
	Sessions   int64    `json:"sessions"`
	BounceRate *float64 `json:"bounce_rate"`
}

// PageItem 是页面排行项。
type PageItem struct {
	URL      string `json:"url"`
	PV       int64  `json:"pv"`
	UV       int64  `json:"uv"`
	Sessions int64  `json:"sessions"`
}

// PageResult 是分页页面排行。
type PageResult struct {
	Total    int        `json:"total"`
	Page     int        `json:"page"`
	PageSize int        `json:"page_size"`
	Items    []PageItem `json:"items"`
}

// DeviceItem 是设备维度项。
type DeviceItem struct {
	Value string `json:"value"`
	PV    int64  `json:"pv"`
}

// DeviceResult 是设备统计。
type DeviceResult struct {
	Browser []DeviceItem `json:"browser"`
	System  []DeviceItem `json:"system"`
	Device  []DeviceItem `json:"device"`
}

func (s *Service) maxRange() int {
	if s.MaxRangeDays <= 0 {
		return 90
	}
	return s.MaxRangeDays
}

func (s *Service) parseRange(fromStr, toStr string) (time.Time, time.Time, error) {
	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		return time.Time{}, time.Time{}, &Error{Code: 400, Msg: "from 日期格式非法（应为 YYYY-MM-DD）"}
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		return time.Time{}, time.Time{}, &Error{Code: 400, Msg: "to 日期格式非法（应为 YYYY-MM-DD）"}
	}
	if from.After(to) {
		return time.Time{}, time.Time{}, &Error{Code: 400, Msg: "from 不得晚于 to"}
	}
	if int(to.Sub(from).Hours()/24)+1 > s.maxRange() {
		return time.Time{}, time.Time{}, &Error{Code: 400, Msg: "日期范围超出上限"}
	}
	return from, to, nil
}

func bounceRate(bounces, sessions int64) *float64 {
	if sessions <= 0 {
		return nil
	}
	r := float64(bounces) / float64(sessions)
	return &r
}

// Overview 查询总览指标。
func (s *Service) Overview(ctx context.Context, appID, fromStr, toStr string) (*Overview, error) {
	from, to, err := s.parseRange(fromStr, toStr)
	if err != nil {
		return nil, err
	}
	row, err := s.DB.GetOverview(ctx, appID, from, to)
	if err != nil {
		return nil, err
	}
	return &Overview{
		AppID: appID, From: fromStr, To: toStr,
		PV: row.PV, UV: row.UV, IPCount: row.IPCount,
		Sessions: row.Sessions, Bounces: row.Bounces,
		BounceRate: bounceRate(row.Bounces, row.Sessions),
	}, nil
}

// Trend 查询按日趋势。
func (s *Service) Trend(ctx context.Context, appID, fromStr, toStr string) ([]TrendPoint, error) {
	from, to, err := s.parseRange(fromStr, toStr)
	if err != nil {
		return nil, err
	}
	rows, err := s.DB.GetTrend(ctx, appID, from, to)
	if err != nil {
		return nil, err
	}
	out := make([]TrendPoint, 0, len(rows))
	for _, r := range rows {
		out = append(out, TrendPoint{
			Date: r.Date.UTC().Format("2006-01-02"),
			PV:   r.PV, UV: r.UV, IPCount: r.IPCount, Sessions: r.Sessions,
			BounceRate: bounceRate(r.Bounces, r.Sessions),
		})
	}
	return out, nil
}

// Pages 查询页面排行。
func (s *Service) Pages(ctx context.Context, appID, fromStr, toStr, sort string, page, pageSize int) (*PageResult, error) {
	from, to, err := s.parseRange(fromStr, toStr)
	if err != nil {
		return nil, err
	}
	if sort == "" {
		sort = "pv"
	}
	if sort != "pv" && sort != "uv" && sort != "sessions" {
		return nil, &Error{Code: 400, Msg: "sort 仅支持 pv、uv、sessions"}
	}
	if page < 1 || pageSize < 1 || pageSize > 100 {
		return nil, &Error{Code: 400, Msg: "分页参数非法（page>=1，1<=page_size<=100）"}
	}
	rows, total, err := s.DB.GetPages(ctx, appID, from, to, sort, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, err
	}
	items := make([]PageItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, PageItem{URL: r.URL, PV: r.PV, UV: r.UV, Sessions: r.Sessions})
	}
	return &PageResult{Total: total, Page: page, PageSize: pageSize, Items: items}, nil
}

// Devices 查询设备维度统计。
func (s *Service) Devices(ctx context.Context, appID, fromStr, toStr string) (*DeviceResult, error) {
	from, to, err := s.parseRange(fromStr, toStr)
	if err != nil {
		return nil, err
	}
	rows, err := s.DB.GetDevices(ctx, appID, from, to)
	if err != nil {
		return nil, err
	}
	res := &DeviceResult{Browser: []DeviceItem{}, System: []DeviceItem{}, Device: []DeviceItem{}}
	for _, r := range rows {
		item := DeviceItem{Value: r.Value, PV: r.PV}
		switch r.Dimension {
		case "browser":
			res.Browser = append(res.Browser, item)
		case "system":
			res.System = append(res.System, item)
		case "device":
			res.Device = append(res.Device, item)
		}
	}
	return res, nil
}
