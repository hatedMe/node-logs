package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// ErrNotFound 表示查询目标不存在。
var ErrNotFound = errors.New("store: not found")

const createAppSQL = `INSERT INTO apps (app_id, secret_key, name, category, domain, username, status)
	VALUES ($1,$2,$3,$4,$5,$6,COALESCE(NULLIF($7,''),'active'))`

func createApp(ctx context.Context, db DBTX, app App) error {
	_, err := db.Exec(ctx, createAppSQL,
		app.AppID, app.SecretKey, app.Name, app.Category, app.Domain, app.Username, app.Status)
	return err
}

// CreateApp 创建应用。
func (s *Store) CreateApp(ctx context.Context, app App) error {
	return createApp(ctx, s.pool, app)
}

// CreateApp 创建应用。
func (t *Tx) CreateApp(ctx context.Context, app App) error {
	return createApp(ctx, t.tx, app)
}

const getAppSQL = `SELECT app_id, secret_key, name, category, domain, username, status, created_at
	FROM apps WHERE app_id = $1`

func scanApp(row pgx.Row) (*App, error) {
	var app App
	if err := row.Scan(&app.AppID, &app.SecretKey, &app.Name, &app.Category,
		&app.Domain, &app.Username, &app.Status, &app.CreatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &app, nil
}

// GetAppByID 按 ID 查询应用；不存在时返回 ErrNotFound。
func (s *Store) GetAppByID(ctx context.Context, appID string) (*App, error) {
	return scanApp(s.pool.QueryRow(ctx, getAppSQL, appID))
}

// GetAppByID 按 ID 查询应用；不存在时返回 ErrNotFound。
func (t *Tx) GetAppByID(ctx context.Context, appID string) (*App, error) {
	return scanApp(t.tx.QueryRow(ctx, getAppSQL, appID))
}

// SetAppStatus 更新应用状态；应用不存在时返回 ErrNotFound。
func (s *Store) SetAppStatus(ctx context.Context, appID, status string) error {
	tag, err := s.pool.Exec(ctx, `UPDATE apps SET status = $2 WHERE app_id = $1`, appID, status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ListApps 返回全部应用，供采集端缓存预热与定时刷新。
func (s *Store) ListApps(ctx context.Context) ([]App, error) {
	rows, err := s.pool.Query(ctx, `SELECT app_id, secret_key, name, category, domain, username, status, created_at
		FROM apps ORDER BY app_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var apps []App
	for rows.Next() {
		var app App
		if err := rows.Scan(&app.AppID, &app.SecretKey, &app.Name, &app.Category,
			&app.Domain, &app.Username, &app.Status, &app.CreatedAt); err != nil {
			return nil, err
		}
		apps = append(apps, app)
	}
	return apps, rows.Err()
}

const createUserSQL = `INSERT INTO users (username, password_hash)
	VALUES ($1,$2) RETURNING id, created_at`

// CreateUser 创建用户并返回完整记录。
func (s *Store) CreateUser(ctx context.Context, username, passwordHash string) (*User, error) {
	var u User
	u.Username = username
	u.PasswordHash = passwordHash
	if err := s.pool.QueryRow(ctx, createUserSQL, username, passwordHash).Scan(&u.ID, &u.CreatedAt); err != nil {
		return nil, err
	}
	return &u, nil
}

// CreateUser 在事务内创建用户并返回完整记录。
func (t *Tx) CreateUser(ctx context.Context, username, passwordHash string) (*User, error) {
	var u User
	u.Username = username
	u.PasswordHash = passwordHash
	if err := t.tx.QueryRow(ctx, createUserSQL, username, passwordHash).Scan(&u.ID, &u.CreatedAt); err != nil {
		return nil, err
	}
	return &u, nil
}

// MaxAppID 返回当前最大的数字应用 ID；空库时返回 800799。
func (s *Store) MaxAppID(ctx context.Context) (int64, error) {
	return maxAppID(ctx, s.pool)
}

// MaxAppID 返回当前最大的数字应用 ID；空库时返回 800799。
func (t *Tx) MaxAppID(ctx context.Context) (int64, error) {
	return maxAppID(ctx, t.tx)
}

func maxAppID(ctx context.Context, db DBTX) (int64, error) {
	var max int64
	err := db.QueryRow(ctx, `SELECT COALESCE(MAX(app_id::bigint), 800799) FROM apps WHERE app_id ~ '^[0-9]+$'`).
		Scan(&max)
	return max, err
}

const getUserSQL = `SELECT id, username, password_hash, created_at FROM users WHERE username = $1`

// GetUserByUsername 按用户名查询用户；不存在时返回 ErrNotFound。
func (s *Store) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	var u User
	if err := s.pool.QueryRow(ctx, getUserSQL, username).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

// UpdateUserPassword 更新用户密码哈希；用户不存在时返回 ErrNotFound。
func (s *Store) UpdateUserPassword(ctx context.Context, username, passwordHash string) error {
	tag, err := s.pool.Exec(ctx, `UPDATE users SET password_hash = $2 WHERE username = $1`, username, passwordHash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
