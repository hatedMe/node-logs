// Package app 实现应用管理与注册登录业务。
package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/hatedMe/node-logs/internal/auth"
	"github.com/hatedMe/node-logs/internal/store"
)

// Error 是携带 HTTP 状态码的应用错误。
type Error struct {
	Code int
	Msg  string
}

func (e *Error) Error() string { return e.Msg }

// RegisterRequest 对应旧注册接口的业务字段。
type RegisterRequest struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Username string `json:"username"`
	Password string `json:"password"`
	Domain   string `json:"domain"`
}

// LoginRequest 是登录请求。
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResult 是登录结果，与旧接口字段一致。
type LoginResult struct {
	Token     string `json:"token"`
	Username  string `json:"username"`
	ExpiresIn int64  `json:"expiresIn"`
}

// CreateAppRequest 是创建应用请求。
type CreateAppRequest struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Domain   string `json:"domain"`
}

// appStore 是 Service 依赖的存储子集；*store.Store 已实现。
type appStore interface {
	WithTx(ctx context.Context, fn func(*store.Tx) error) error
	GetAppByID(ctx context.Context, appID string) (*store.App, error)
	GetUserByUsername(ctx context.Context, username string) (*store.User, error)
	UpdateUserPassword(ctx context.Context, username, passwordHash string) error
}

// Service 实现注册、登录与应用管理。
type Service struct {
	Store     appStore
	JWTSecret string
	JWTTTL    time.Duration
}

// errAppIDConflict 表示应用 ID 分配冲突，可重试。
var errAppIDConflict = errors.New("app: 应用 ID 冲突")

// Register 创建用户与首个应用；用户名已存在时返回 409。
func (s *Service) Register(ctx context.Context, req RegisterRequest) (*store.App, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Category = strings.TrimSpace(req.Category)
	req.Username = strings.TrimSpace(req.Username)
	if req.Name == "" || req.Category == "" || req.Username == "" || req.Password == "" {
		return nil, &Error{Code: 400, Msg: "缺少必传参数：name、category、username、password"}
	}
	hash, err := auth.Hash(req.Password)
	if err != nil {
		return nil, err
	}

	var app *store.App
	for range 3 {
		var done *store.App
		err := s.Store.WithTx(ctx, func(tx *store.Tx) error {
			if _, err := tx.CreateUser(ctx, req.Username, hash); err != nil {
				if isUniqueViolation(err, "users_username_key") {
					return &Error{Code: 409, Msg: "用户名已存在"}
				}
				return err
			}
			created, err := createAppInTx(ctx, tx, req.Username, req.Name, req.Category, req.Domain)
			if err != nil {
				return err
			}
			done = created
			return nil
		})
		if err == nil {
			app = done
			break
		}
		var appErr *Error
		if errors.As(err, &appErr) {
			return nil, err
		}
		if errors.Is(err, errAppIDConflict) {
			continue
		}
		return nil, err
	}
	if app == nil {
		return nil, errors.New("app: 分配应用 ID 冲突，请重试")
	}
	return app, nil
}

// Login 校验账号密码并签发 JWT；失败时统一返回 401，不区分用户不存在与密码错误。
func (s *Service) Login(ctx context.Context, req LoginRequest) (*LoginResult, error) {
	if s.JWTSecret == "" {
		return nil, &Error{Code: 500, Msg: "服务未配置 JWT 密钥"}
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		return nil, &Error{Code: 400, Msg: "缺少用户名或密码"}
	}
	user, err := s.Store.GetUserByUsername(ctx, req.Username)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, &Error{Code: 401, Msg: "账号或密码错误"}
		}
		return nil, err
	}
	if err := auth.Verify(user.PasswordHash, req.Password); err != nil {
		// 兼容旧库导入的 MD5 用户：验证通过则透明升级为 bcrypt。
		if !s.tryUpgradeLegacyPassword(ctx, user, req.Password) {
			return nil, &Error{Code: 401, Msg: "账号或密码错误"}
		}
	}
	token, err := auth.Sign(s.JWTSecret, user.Username, s.JWTTTL)
	if err != nil {
		return nil, err
	}
	return &LoginResult{Token: token, Username: user.Username, ExpiresIn: int64(s.JWTTTL / time.Second)}, nil
}

// tryUpgradeLegacyPassword 对 MD5 老用户验密，成功则升级为 bcrypt 并返回 true。
// 升级写库失败不阻塞登录（下次登录重试），但验密失败必须返回 false。
func (s *Service) tryUpgradeLegacyPassword(ctx context.Context, user *store.User, password string) bool {
	if !auth.IsMD5Hash(user.PasswordHash) {
		return false
	}
	if auth.MD5Upper(password) != strings.ToUpper(user.PasswordHash) {
		return false
	}
	if hash, err := auth.Hash(password); err == nil {
		_ = s.Store.UpdateUserPassword(ctx, user.Username, hash)
	}
	return true
}

// CreateApp 为属主创建新应用并返回 app_id 与 secret_key。
func (s *Service) CreateApp(ctx context.Context, owner string, req CreateAppRequest) (*store.App, error) {
	if owner == "" {
		return nil, &Error{Code: 401, Msg: "缺少登录用户"}
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return nil, &Error{Code: 400, Msg: "缺少必传参数：name"}
	}
	var app *store.App
	for range 3 {
		var done *store.App
		err := s.Store.WithTx(ctx, func(tx *store.Tx) error {
			created, err := createAppInTx(ctx, tx, owner, req.Name, req.Category, req.Domain)
			if err != nil {
				return err
			}
			done = created
			return nil
		})
		if err == nil {
			app = done
			break
		}
		if errors.Is(err, errAppIDConflict) {
			continue
		}
		return nil, err
	}
	if app == nil {
		return nil, errors.New("app: 分配应用 ID 冲突，请重试")
	}
	return app, nil
}

func createAppInTx(ctx context.Context, tx *store.Tx, owner, name, category, domain string) (*store.App, error) {
	maxID, err := tx.MaxAppID(ctx)
	if err != nil {
		return nil, err
	}
	secret, err := randomSecret()
	if err != nil {
		return nil, err
	}
	app := store.App{
		AppID:     strconv.FormatInt(maxID+1, 10),
		SecretKey: secret,
		Name:      name,
		Category:  category,
		Domain:    domain,
		Username:  owner,
		Status:    "active",
	}
	if err := tx.CreateApp(ctx, app); err != nil {
		if isUniqueViolation(err, "apps_pkey") {
			return nil, errAppIDConflict
		}
		return nil, err
	}
	return &app, nil
}

func randomSecret() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == "23505" && pgErr.ConstraintName == constraint
}
