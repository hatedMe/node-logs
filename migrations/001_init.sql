-- 001_init.sql：核心表结构（主键与唯一约束）。
-- 补充说明：daily_page_visitors / daily_page_sessions / daily_device_stats
-- 为计划外增补，用于支撑"报表只读聚合表"（页面 UV/会话去重、设备统计）。

CREATE TABLE IF NOT EXISTS apps (
    app_id      TEXT NOT NULL PRIMARY KEY,
    secret_key  TEXT NOT NULL,
    name        TEXT NOT NULL DEFAULT '',
    category    TEXT NOT NULL DEFAULT '',
    domain      TEXT NOT NULL DEFAULT '',
    username    TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS raw_events (
    event_id        TEXT NOT NULL PRIMARY KEY,
    app_id          TEXT NOT NULL REFERENCES apps (app_id),
    visitor_id      TEXT NOT NULL,
    session_id      TEXT NOT NULL DEFAULT '',
    url             TEXT NOT NULL DEFAULT '',
    arg             TEXT NOT NULL DEFAULT '',
    version         TEXT NOT NULL DEFAULT '',
    screen          TEXT NOT NULL DEFAULT '',
    lang            TEXT NOT NULL DEFAULT '',
    tz              TEXT NOT NULL DEFAULT '',
    domain          TEXT NOT NULL DEFAULT '',
    event_time      TIMESTAMPTZ NOT NULL,
    received_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    ua              TEXT NOT NULL DEFAULT '',
    ip_hash         TEXT NOT NULL DEFAULT '',
    browser_name    TEXT NOT NULL DEFAULT '',
    browser_version TEXT NOT NULL DEFAULT '',
    system          TEXT NOT NULL DEFAULT '',
    system_version  TEXT NOT NULL DEFAULT '',
    device_type     TEXT NOT NULL DEFAULT '',
    extra           JSONB NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS daily_visitors (
    app_id     TEXT NOT NULL,
    stat_date  DATE NOT NULL,
    visitor_id TEXT NOT NULL,
    PRIMARY KEY (app_id, stat_date, visitor_id)
);

CREATE TABLE IF NOT EXISTS daily_ips (
    app_id    TEXT NOT NULL,
    stat_date DATE NOT NULL,
    ip_hash   TEXT NOT NULL,
    PRIMARY KEY (app_id, stat_date, ip_hash)
);

CREATE TABLE IF NOT EXISTS daily_stats (
    app_id    TEXT NOT NULL,
    stat_date DATE NOT NULL,
    pv        BIGINT NOT NULL DEFAULT 0,
    uv        BIGINT NOT NULL DEFAULT 0,
    ip_count  BIGINT NOT NULL DEFAULT 0,
    sessions  BIGINT NOT NULL DEFAULT 0,
    bounces   BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (app_id, stat_date)
);

CREATE TABLE IF NOT EXISTS daily_page_stats (
    app_id    TEXT NOT NULL,
    stat_date DATE NOT NULL,
    url       TEXT NOT NULL,
    pv        BIGINT NOT NULL DEFAULT 0,
    uv        BIGINT NOT NULL DEFAULT 0,
    sessions  BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (app_id, stat_date, url)
);

CREATE TABLE IF NOT EXISTS daily_page_visitors (
    app_id     TEXT NOT NULL,
    stat_date  DATE NOT NULL,
    url        TEXT NOT NULL,
    visitor_id TEXT NOT NULL,
    PRIMARY KEY (app_id, stat_date, url, visitor_id)
);

CREATE TABLE IF NOT EXISTS daily_page_sessions (
    app_id     TEXT NOT NULL,
    stat_date  DATE NOT NULL,
    url        TEXT NOT NULL,
    session_id TEXT NOT NULL,
    PRIMARY KEY (app_id, stat_date, url, session_id)
);

CREATE TABLE IF NOT EXISTS daily_device_stats (
    app_id    TEXT NOT NULL,
    stat_date DATE NOT NULL,
    dimension TEXT NOT NULL,
    value     TEXT NOT NULL,
    pv        BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (app_id, stat_date, dimension, value)
);

CREATE TABLE IF NOT EXISTS sessions (
    session_id  TEXT NOT NULL PRIMARY KEY,
    app_id      TEXT NOT NULL,
    visitor_id  TEXT NOT NULL,
    started_at  TIMESTAMPTZ NOT NULL,
    ended_at    TIMESTAMPTZ NOT NULL,
    event_count BIGINT NOT NULL DEFAULT 1,
    entry_url   TEXT NOT NULL DEFAULT '',
    exit_url    TEXT NOT NULL DEFAULT ''
);
