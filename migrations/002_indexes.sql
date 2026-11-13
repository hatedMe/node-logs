-- 002_indexes.sql：查询与清理所需的二级索引。

-- 原始事件按应用+时间范围查询。
CREATE INDEX IF NOT EXISTS idx_raw_events_app_time ON raw_events (app_id, event_time);

-- 原始事件按保留周期清理。
CREATE INDEX IF NOT EXISTS idx_raw_events_received ON raw_events (received_at);

-- 会话按访客查找未超时会话。
CREATE INDEX IF NOT EXISTS idx_sessions_visitor ON sessions (app_id, visitor_id, ended_at DESC);

-- 报表按日期范围扫描聚合表。
CREATE INDEX IF NOT EXISTS idx_daily_stats_date ON daily_stats (stat_date);

-- 页面排行按 PV 倒序。
CREATE INDEX IF NOT EXISTS idx_daily_page_stats_lookup ON daily_page_stats (app_id, stat_date, pv DESC);

-- 设备统计按维度查询。
CREATE INDEX IF NOT EXISTS idx_daily_device_lookup ON daily_device_stats (app_id, stat_date, dimension);
