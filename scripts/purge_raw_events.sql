-- 清理超出保留期的原始事件；聚合表（daily_*）与会话表不受影响。
-- 用法：
--   psql "$DATABASE_URL" -v retention_days=30 -f scripts/purge_raw_events.sql
-- 建议以 cron 每日执行，retention_days 与 RETENTION_DAYS 环境变量保持一致。
DELETE FROM raw_events
WHERE event_time < now() - make_interval(days => :'retention_days'::int);
