package store

import (
	"context"
	"time"
)

// OverviewRow 是总览聚合行。
type OverviewRow struct {
	PV       int64
	UV       int64
	IPCount  int64
	Sessions int64
	Bounces  int64
}

// TrendRow 是单日趋势行。
type TrendRow struct {
	Date     time.Time
	PV       int64
	UV       int64
	IPCount  int64
	Sessions int64
	Bounces  int64
}

// PageRow 是页面排行行。
type PageRow struct {
	URL      string
	PV       int64
	UV       int64
	Sessions int64
}

// DeviceRow 是设备维度行。
type DeviceRow struct {
	Dimension string
	Value     string
	PV        int64
}

// GetOverview 汇总日期闭区间内的总览指标；无数据时返回零值。
func (s *Store) GetOverview(ctx context.Context, appID string, from, to time.Time) (OverviewRow, error) {
	var ov OverviewRow
	err := s.pool.QueryRow(ctx, `SELECT COALESCE(SUM(pv),0), COALESCE(SUM(uv),0),
		COALESCE(SUM(ip_count),0), COALESCE(SUM(sessions),0), COALESCE(SUM(bounces),0)
		FROM daily_stats WHERE app_id = $1 AND stat_date BETWEEN $2 AND $3`,
		appID, statDate(from), statDate(to)).
		Scan(&ov.PV, &ov.UV, &ov.IPCount, &ov.Sessions, &ov.Bounces)
	return ov, err
}

// GetTrend 返回日期闭区间内按日升序的趋势。
func (s *Store) GetTrend(ctx context.Context, appID string, from, to time.Time) ([]TrendRow, error) {
	rows, err := s.pool.Query(ctx, `SELECT stat_date, pv, uv, ip_count, sessions, bounces
		FROM daily_stats WHERE app_id = $1 AND stat_date BETWEEN $2 AND $3 ORDER BY stat_date`,
		appID, statDate(from), statDate(to))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TrendRow
	for rows.Next() {
		var r TrendRow
		if err := rows.Scan(&r.Date, &r.PV, &r.UV, &r.IPCount, &r.Sessions, &r.Bounces); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetPages 返回页面排行与去重页面总数，sort 仅允许 pv/uv/sessions。
func (s *Store) GetPages(ctx context.Context, appID string, from, to time.Time, sort string, limit, offset int) ([]PageRow, int, error) {
	order := "SUM(pv)"
	switch sort {
	case "", "pv":
		order = "SUM(pv)"
	case "uv":
		order = "SUM(uv)"
	case "sessions":
		order = "SUM(sessions)"
	default:
		order = "SUM(pv)"
	}
	rows, err := s.pool.Query(ctx, `SELECT url, SUM(pv), SUM(uv), SUM(sessions)
		FROM daily_page_stats WHERE app_id = $1 AND stat_date BETWEEN $2 AND $3
		GROUP BY url ORDER BY `+order+` DESC, url LIMIT $4 OFFSET $5`,
		appID, statDate(from), statDate(to), limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []PageRow
	for rows.Next() {
		var r PageRow
		if err := rows.Scan(&r.URL, &r.PV, &r.UV, &r.Sessions); err != nil {
			return nil, 0, err
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int
	err = s.pool.QueryRow(ctx, `SELECT COUNT(DISTINCT url) FROM daily_page_stats
		WHERE app_id = $1 AND stat_date BETWEEN $2 AND $3`,
		appID, statDate(from), statDate(to)).Scan(&total)
	return out, total, err
}

// GetDevices 返回设备维度汇总行。
func (s *Store) GetDevices(ctx context.Context, appID string, from, to time.Time) ([]DeviceRow, error) {
	rows, err := s.pool.Query(ctx, `SELECT dimension, value, SUM(pv)
		FROM daily_device_stats WHERE app_id = $1 AND stat_date BETWEEN $2 AND $3
		GROUP BY dimension, value ORDER BY dimension, SUM(pv) DESC`,
		appID, statDate(from), statDate(to))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DeviceRow
	for rows.Next() {
		var r DeviceRow
		if err := rows.Scan(&r.Dimension, &r.Value, &r.PV); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
