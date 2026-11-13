package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBTX 是 *pgxpool.Pool 与 pgx.Tx 的公共子集，使同一套 SQL 同时支持
// 直接执行与事务执行。
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
}

// Store 是 PostgreSQL 连接池的薄封装。
type Store struct {
	pool *pgxpool.Pool
}

// New 创建连接池。
func New(ctx context.Context, dsn string) (*Store, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	return &Store{pool: pool}, nil
}

// Close 关闭连接池。
func (s *Store) Close() {
	s.pool.Close()
}

// Ping 检查数据库连通性，供就绪探测使用。
func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// PoolStats 返回连接池已用与总数，供指标抓取使用。
func (s *Store) PoolStats() (acquired, total int32) {
	stat := s.pool.Stat()
	return stat.AcquiredConns(), stat.TotalConns()
}

// Migrate 应用全部待执行迁移。
func (s *Store) Migrate(ctx context.Context) error {
	return Migrate(ctx, s.pool)
}

// Tx 是事务内的存储访问器。
type Tx struct {
	tx pgx.Tx
}

// WithTx 在单个事务中执行业务函数；函数返回错误时回滚。
func (s *Store) WithTx(ctx context.Context, fn func(*Tx) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(&Tx{tx: tx}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// statDate 将任意时刻归一化为 UTC 自然日，供 stat_date 列使用。
func statDate(t time.Time) pgtype.Date {
	u := t.UTC()
	return pgtype.Date{
		Time:  time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC),
		Valid: true,
	}
}

const insertRawEventSQL = `INSERT INTO raw_events
	(event_id, app_id, visitor_id, session_id, url, arg, version, screen, lang, tz, domain,
	 event_time, received_at, ua, ip_hash, browser_name, browser_version,
	 system, system_version, device_type, extra)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21::jsonb)
	ON CONFLICT (event_id) DO NOTHING
	RETURNING event_id`

func rawEventArgs(e RawEvent) ([]any, error) {
	extra := e.Extra
	if extra == nil {
		extra = map[string]string{}
	}
	raw, err := json.Marshal(extra)
	if err != nil {
		return nil, err
	}
	return []any{
		e.EventID, e.AppID, e.VisitorID, e.SessionID, e.URL, e.Arg, e.Version,
		e.Screen, e.Lang, e.TZ, e.Domain, e.EventTime, e.ReceivedAt, e.UA,
		e.IPHash, e.BrowserName, e.BrowserVer, e.System, e.SystemVer,
		e.DeviceType, string(raw),
	}, nil
}

func insertRawEventsReturning(ctx context.Context, db DBTX, events []RawEvent) ([]string, error) {
	if len(events) == 0 {
		return nil, nil
	}
	b := &pgx.Batch{}
	for _, e := range events {
		args, err := rawEventArgs(e)
		if err != nil {
			return nil, err
		}
		b.Queue(insertRawEventSQL, args...)
	}
	br := db.SendBatch(ctx, b)
	defer br.Close()

	var inserted []string
	for range events {
		var id string
		rows, err := br.Query()
		if err != nil {
			return nil, err
		}
		if rows.Next() {
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return nil, err
			}
			inserted = append(inserted, id)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	return inserted, nil
}

// InsertRawEvents 批量写入原始事件；重复 event_id 静默跳过。
func (s *Store) InsertRawEvents(ctx context.Context, events []RawEvent) error {
	_, err := insertRawEventsReturning(ctx, s.pool, events)
	return err
}

// InsertRawEventsReturning 批量写入并返回实际插入的 event_id。
func (s *Store) InsertRawEventsReturning(ctx context.Context, events []RawEvent) ([]string, error) {
	return insertRawEventsReturning(ctx, s.pool, events)
}

// PurgeRawEvents 删除早于保留天数的原始事件，返回删除行数。
// retentionDays 为负时拒绝执行（调用方用 <=0 表示禁用清理）。
func (s *Store) PurgeRawEvents(ctx context.Context, retentionDays int) (int64, error) {
	if retentionDays < 0 {
		return 0, errors.New("store: 保留天数不能为负")
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)
	tag, err := s.pool.Exec(ctx, `DELETE FROM raw_events WHERE event_time < $1`, cutoff)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// InsertRawEvents 批量写入原始事件；重复 event_id 静默跳过。
func (t *Tx) InsertRawEvents(ctx context.Context, events []RawEvent) error {
	_, err := insertRawEventsReturning(ctx, t.tx, events)
	return err
}

// InsertRawEventsReturning 批量写入并返回实际插入的 event_id。
func (t *Tx) InsertRawEventsReturning(ctx context.Context, events []RawEvent) ([]string, error) {
	return insertRawEventsReturning(ctx, t.tx, events)
}

const updateRawSessionIDsSQL = `UPDATE raw_events AS r SET session_id = v.sid
	FROM (SELECT unnest($1::text[]) AS eid, unnest($2::text[]) AS sid) AS v
	WHERE r.event_id = v.eid`

func updateRawSessionIDs(ctx context.Context, db DBTX, pairs map[string]string) error {
	if len(pairs) == 0 {
		return nil
	}
	ids := make([]string, 0, len(pairs))
	sids := make([]string, 0, len(pairs))
	for id, sid := range pairs {
		ids = append(ids, id)
		sids = append(sids, sid)
	}
	_, err := db.Exec(ctx, updateRawSessionIDsSQL, ids, sids)
	return err
}

// UpdateRawSessionIDs 批量回填原始事件的服务端会话 ID。
func (s *Store) UpdateRawSessionIDs(ctx context.Context, pairs map[string]string) error {
	return updateRawSessionIDs(ctx, s.pool, pairs)
}

// UpdateRawSessionIDs 批量回填原始事件的服务端会话 ID。
func (t *Tx) UpdateRawSessionIDs(ctx context.Context, pairs map[string]string) error {
	return updateRawSessionIDs(ctx, t.tx, pairs)
}

func insertIfNew(ctx context.Context, db DBTX, sql string, args ...any) (bool, error) {
	var one int
	err := db.QueryRow(ctx, sql, args...).Scan(&one)
	if err != nil {
		// ON CONFLICT DO NOTHING 命中时 RETURNING 无行，属正常"非新"结果。
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return one == 1, nil
}

// InsertVisitorIfNew 插入 UV 去重键；返回是否为当日新访客。
func (s *Store) InsertVisitorIfNew(ctx context.Context, key VisitorKey) (bool, error) {
	return insertIfNew(ctx, s.pool,
		`INSERT INTO daily_visitors (app_id, stat_date, visitor_id) VALUES ($1,$2,$3)
		 ON CONFLICT DO NOTHING RETURNING 1`,
		key.AppID, statDate(key.Date), key.VisitorID)
}

// InsertVisitorIfNew 插入 UV 去重键；返回是否为当日新访客。
func (t *Tx) InsertVisitorIfNew(ctx context.Context, key VisitorKey) (bool, error) {
	return insertIfNew(ctx, t.tx,
		`INSERT INTO daily_visitors (app_id, stat_date, visitor_id) VALUES ($1,$2,$3)
		 ON CONFLICT DO NOTHING RETURNING 1`,
		key.AppID, statDate(key.Date), key.VisitorID)
}

// InsertIPIfNew 插入独立 IP 去重键；返回是否为当日新 IP。
func (s *Store) InsertIPIfNew(ctx context.Context, key IPKey) (bool, error) {
	return insertIfNew(ctx, s.pool,
		`INSERT INTO daily_ips (app_id, stat_date, ip_hash) VALUES ($1,$2,$3)
		 ON CONFLICT DO NOTHING RETURNING 1`,
		key.AppID, statDate(key.Date), key.IPHash)
}

// InsertIPIfNew 插入独立 IP 去重键；返回是否为当日新 IP。
func (t *Tx) InsertIPIfNew(ctx context.Context, key IPKey) (bool, error) {
	return insertIfNew(ctx, t.tx,
		`INSERT INTO daily_ips (app_id, stat_date, ip_hash) VALUES ($1,$2,$3)
		 ON CONFLICT DO NOTHING RETURNING 1`,
		key.AppID, statDate(key.Date), key.IPHash)
}

// InsertPageVisitorIfNew 插入页面 UV 去重键；返回是否为当日该页面新访客。
func (s *Store) InsertPageVisitorIfNew(ctx context.Context, key PageVisitorKey) (bool, error) {
	return insertIfNew(ctx, s.pool,
		`INSERT INTO daily_page_visitors (app_id, stat_date, url, visitor_id) VALUES ($1,$2,$3,$4)
		 ON CONFLICT DO NOTHING RETURNING 1`,
		key.AppID, statDate(key.Date), key.URL, key.VisitorID)
}

// InsertPageVisitorIfNew 插入页面 UV 去重键；返回是否为当日该页面新访客。
func (t *Tx) InsertPageVisitorIfNew(ctx context.Context, key PageVisitorKey) (bool, error) {
	return insertIfNew(ctx, t.tx,
		`INSERT INTO daily_page_visitors (app_id, stat_date, url, visitor_id) VALUES ($1,$2,$3,$4)
		 ON CONFLICT DO NOTHING RETURNING 1`,
		key.AppID, statDate(key.Date), key.URL, key.VisitorID)
}

// InsertPageSessionIfNew 插入页面会话去重键；返回是否为当日该页面新会话。
func (s *Store) InsertPageSessionIfNew(ctx context.Context, key PageSessionKey) (bool, error) {
	return insertIfNew(ctx, s.pool,
		`INSERT INTO daily_page_sessions (app_id, stat_date, url, session_id) VALUES ($1,$2,$3,$4)
		 ON CONFLICT DO NOTHING RETURNING 1`,
		key.AppID, statDate(key.Date), key.URL, key.SessionID)
}

// InsertPageSessionIfNew 插入页面会话去重键；返回是否为当日该页面新会话。
func (t *Tx) InsertPageSessionIfNew(ctx context.Context, key PageSessionKey) (bool, error) {
	return insertIfNew(ctx, t.tx,
		`INSERT INTO daily_page_sessions (app_id, stat_date, url, session_id) VALUES ($1,$2,$3,$4)
		 ON CONFLICT DO NOTHING RETURNING 1`,
		key.AppID, statDate(key.Date), key.URL, key.SessionID)
}

const upsertDailyStatsSQL = `INSERT INTO daily_stats (app_id, stat_date, pv, uv, ip_count, sessions, bounces)
	VALUES ($1,$2,$3,$4,$5,$6,$7)
	ON CONFLICT (app_id, stat_date) DO UPDATE SET
		pv = daily_stats.pv + EXCLUDED.pv,
		uv = daily_stats.uv + EXCLUDED.uv,
		ip_count = daily_stats.ip_count + EXCLUDED.ip_count,
		sessions = daily_stats.sessions + EXCLUDED.sessions,
		bounces = GREATEST(0, daily_stats.bounces + EXCLUDED.bounces)`

func upsertDailyStats(ctx context.Context, db DBTX, delta StatsDelta) error {
	_, err := db.Exec(ctx, upsertDailyStatsSQL,
		delta.AppID, statDate(delta.Date),
		delta.PV, delta.UV, delta.IPCount, delta.Sessions, delta.Bounces)
	return err
}

// UpsertDailyStats 按增量更新日汇总。
func (s *Store) UpsertDailyStats(ctx context.Context, delta StatsDelta) error {
	return upsertDailyStats(ctx, s.pool, delta)
}

// UpsertDailyStats 按增量更新日汇总。
func (t *Tx) UpsertDailyStats(ctx context.Context, delta StatsDelta) error {
	return upsertDailyStats(ctx, t.tx, delta)
}

const upsertPageStatsSQL = `INSERT INTO daily_page_stats (app_id, stat_date, url, pv, uv, sessions)
	VALUES ($1,$2,$3,$4,$5,$6)
	ON CONFLICT (app_id, stat_date, url) DO UPDATE SET
		pv = daily_page_stats.pv + EXCLUDED.pv,
		uv = daily_page_stats.uv + EXCLUDED.uv,
		sessions = daily_page_stats.sessions + EXCLUDED.sessions`

func upsertPageStats(ctx context.Context, db DBTX, appID string, date time.Time, pages []PageDelta) error {
	for _, p := range pages {
		if _, err := db.Exec(ctx, upsertPageStatsSQL,
			appID, statDate(date), p.URL, p.PV, p.UV, p.Sessions); err != nil {
			return err
		}
	}
	return nil
}

// UpsertPageStats 按增量更新页面汇总。
func (s *Store) UpsertPageStats(ctx context.Context, appID string, date time.Time, pages []PageDelta) error {
	return upsertPageStats(ctx, s.pool, appID, date, pages)
}

// UpsertPageStats 按增量更新页面汇总。
func (t *Tx) UpsertPageStats(ctx context.Context, appID string, date time.Time, pages []PageDelta) error {
	return upsertPageStats(ctx, t.tx, appID, date, pages)
}

const upsertDeviceStatsSQL = `INSERT INTO daily_device_stats (app_id, stat_date, dimension, value, pv)
	VALUES ($1,$2,$3,$4,$5)
	ON CONFLICT (app_id, stat_date, dimension, value) DO UPDATE SET
		pv = daily_device_stats.pv + EXCLUDED.pv`

func upsertDeviceStats(ctx context.Context, db DBTX, appID string, date time.Time, devices []DeviceDelta) error {
	for _, d := range devices {
		if _, err := db.Exec(ctx, upsertDeviceStatsSQL,
			appID, statDate(date), d.Dimension, d.Value, d.PV); err != nil {
			return err
		}
	}
	return nil
}

// UpsertDeviceStats 按增量更新设备维度汇总。
func (s *Store) UpsertDeviceStats(ctx context.Context, appID string, date time.Time, devices []DeviceDelta) error {
	return upsertDeviceStats(ctx, s.pool, appID, date, devices)
}

// UpsertDeviceStats 按增量更新设备维度汇总。
func (t *Tx) UpsertDeviceStats(ctx context.Context, appID string, date time.Time, devices []DeviceDelta) error {
	return upsertDeviceStats(ctx, t.tx, appID, date, devices)
}

const findOpenSessionSQL = `SELECT session_id, app_id, visitor_id, started_at, ended_at,
		event_count, entry_url, exit_url
	FROM sessions
	WHERE app_id = $1 AND visitor_id = $2 AND ended_at >= $3
	ORDER BY ended_at DESC
	LIMIT 1`

func scanSession(row pgx.Row) (*Session, error) {
	var s Session
	if err := row.Scan(&s.SessionID, &s.AppID, &s.VisitorID, &s.StartedAt, &s.EndedAt,
		&s.EventCount, &s.EntryURL, &s.ExitURL); err != nil {
		return nil, err
	}
	return &s, nil
}

func findOpenSession(ctx context.Context, db DBTX, appID, visitorID string, since time.Time) (*Session, error) {
	s, err := scanSession(db.QueryRow(ctx, findOpenSessionSQL, appID, visitorID, since))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return s, nil
}

// LockVisitor 获取访客会话临界区的事务级咨询锁（可重入，随事务结束释放），
// 串行化多 worker 对同访客的查找-创建，防止重复建会话与重复统计。
// 仅事务内有效；池直连调用返回错误而非静默无锁。
func (t *Tx) LockVisitor(ctx context.Context, appID, visitorID string) error {
	// 嵌套哈希避免分隔符拼接的碰撞与 NUL 非法问题。
	_, err := t.tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, hashtextextended($2, 0)))`, appID, visitorID)
	return err
}

// LockVisitor 在池直连上无事务语义，明确拒绝。
func (s *Store) LockVisitor(_ context.Context, _, _ string) error {
	return errors.New("store: LockVisitor 仅事务内有效")
}

// FindOpenSession 查找访客在 since 之后仍活跃的最新会话；无会话时返回 (nil, nil)。
func (s *Store) FindOpenSession(ctx context.Context, appID, visitorID string, since time.Time) (*Session, error) {
	return findOpenSession(ctx, s.pool, appID, visitorID, since)
}

// FindOpenSession 查找访客在 since 之后仍活跃的最新会话；无会话时返回 (nil, nil)。
func (t *Tx) FindOpenSession(ctx context.Context, appID, visitorID string, since time.Time) (*Session, error) {
	return findOpenSession(ctx, t.tx, appID, visitorID, since)
}

const insertSessionSQL = `INSERT INTO sessions
	(session_id, app_id, visitor_id, started_at, ended_at, event_count, entry_url, exit_url)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`

func insertSession(ctx context.Context, db DBTX, s Session) error {
	_, err := db.Exec(ctx, insertSessionSQL,
		s.SessionID, s.AppID, s.VisitorID, s.StartedAt, s.EndedAt,
		s.EventCount, s.EntryURL, s.ExitURL)
	return err
}

// InsertSession 创建会话。
func (s *Store) InsertSession(ctx context.Context, sess Session) error {
	return insertSession(ctx, s.pool, sess)
}

// InsertSession 创建会话。
func (t *Tx) InsertSession(ctx context.Context, sess Session) error {
	return insertSession(ctx, t.tx, sess)
}

const extendSessionSQL = `UPDATE sessions
	SET ended_at = $2, exit_url = $3, event_count = event_count + 1
	WHERE session_id = $1
	RETURNING event_count`

func extendSession(ctx context.Context, db DBTX, sessionID string, endedAt time.Time, exitURL string) (int64, error) {
	var count int64
	if err := db.QueryRow(ctx, extendSessionSQL, sessionID, endedAt, exitURL).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// ExtendSession 延长会话并返回延长后的事件数。
func (s *Store) ExtendSession(ctx context.Context, sessionID string, endedAt time.Time, exitURL string) (int64, error) {
	return extendSession(ctx, s.pool, sessionID, endedAt, exitURL)
}

// ExtendSession 延长会话并返回延长后的事件数。
func (t *Tx) ExtendSession(ctx context.Context, sessionID string, endedAt time.Time, exitURL string) (int64, error) {
	return extendSession(ctx, t.tx, sessionID, endedAt, exitURL)
}
