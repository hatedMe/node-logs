package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hatedMe/node-logs/internal/auth"
	"github.com/hatedMe/node-logs/internal/store"
	"github.com/hatedMe/node-logs/internal/testdb"
)

// stubStore 模拟存储；校验测试不应触达存储层。
type stubStore struct{}

func (stubStore) WithTx(_ context.Context, _ func(*store.Tx) error) error {
	return errors.New("unexpected store call")
}
func (stubStore) GetAppByID(_ context.Context, _ string) (*store.App, error) {
	return nil, errors.New("unexpected store call")
}
func (stubStore) GetUserByUsername(_ context.Context, _ string) (*store.User, error) {
	return nil, errors.New("unexpected store call")
}
func (stubStore) UpdateUserPassword(_ context.Context, _, _ string) error {
	return errors.New("unexpected store call")
}

func testService(t *testing.T) *Service {
	t.Helper()
	dsn := testdb.Prepare(t, "nodelogs_test_app")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := store.New(ctx, dsn)
	if err != nil {
		t.Fatalf("创建 Store 失败: %v", err)
	}
	t.Cleanup(db.Close)
	return &Service{Store: db, JWTSecret: "test-secret", JWTTTL: 2 * time.Hour}
}

func TestRegisterValidation(t *testing.T) {
	svc := &Service{Store: stubStore{}, JWTSecret: "s", JWTTTL: time.Hour}
	cases := map[string]RegisterRequest{
		"缺少 name":     {Category: "web", Username: "u", Password: "p"},
		"缺少 category": {Name: "n", Username: "u", Password: "p"},
		"缺少 username": {Name: "n", Category: "web", Password: "p"},
		"缺少 password": {Name: "n", Category: "web", Username: "u"},
	}
	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.Register(context.Background(), req)
			var appErr *Error
			if !errors.As(err, &appErr) || appErr.Code != 400 {
				t.Errorf("%s 应返回 400 错误，实际为 %v", name, err)
			}
		})
	}
}

func TestRegisterOK(t *testing.T) {
	svc := testService(t)
	ctx := context.Background()

	app, err := svc.Register(ctx, RegisterRequest{
		Name: "demo", Category: "web", Username: "alice", Password: "s3cret", Domain: "demo.com",
	})
	if err != nil {
		t.Fatalf("注册失败: %v", err)
	}
	if app.AppID != "800800" {
		t.Errorf("首个应用 ID 应为 800800，实际为 %q", app.AppID)
	}
	if len(app.SecretKey) != 64 {
		t.Errorf("secret_key 应为 64 位十六进制，实际为 %q", app.SecretKey)
	}
	if app.Domain != "demo.com" || app.Username != "alice" {
		t.Errorf("应用字段错误: %+v", app)
	}

	second, err := svc.Register(ctx, RegisterRequest{
		Name: "demo2", Category: "web", Username: "bob", Password: "s3cret",
	})
	if err != nil {
		t.Fatalf("第二次注册失败: %v", err)
	}
	if second.AppID != "800801" {
		t.Errorf("第二个应用 ID 应为 800801，实际为 %q", second.AppID)
	}

	// 密码不得明文落库。
	u, err := svc.Store.GetUserByUsername(ctx, "alice")
	if err != nil {
		t.Fatalf("读取用户失败: %v", err)
	}
	if u.PasswordHash == "s3cret" {
		t.Error("密码不得明文落库")
	}
	if err := auth.Verify(u.PasswordHash, "s3cret"); err != nil {
		t.Errorf("落库哈希应可验证，实际错误: %v", err)
	}
}

func TestRegisterDuplicateUsername(t *testing.T) {
	svc := testService(t)
	ctx := context.Background()

	req := RegisterRequest{Name: "n", Category: "web", Username: "dup", Password: "p"}
	if _, err := svc.Register(ctx, req); err != nil {
		t.Fatalf("首次注册失败: %v", err)
	}
	_, err := svc.Register(ctx, req)
	var appErr *Error
	if !errors.As(err, &appErr) || appErr.Code != 409 {
		t.Fatalf("重复用户名应返回 409，实际为 %v", err)
	}
}

func TestLoginEmptySecret(t *testing.T) {
	// 空密钥必须在触达存储层之前拒绝（stub 存储任何调用都会失败）。
	svc := &Service{Store: stubStore{}, JWTSecret: "", JWTTTL: time.Hour}
	_, err := svc.Login(context.Background(), LoginRequest{Username: "alice", Password: "p"})
	var appErr *Error
	if !errors.As(err, &appErr) || appErr.Code != 500 {
		t.Fatalf("空 JWT 密钥应返回 500 错误，实际为 %v", err)
	}
}

func TestLoginOK(t *testing.T) {
	svc := testService(t)
	ctx := context.Background()

	if _, err := svc.Register(ctx, RegisterRequest{Name: "n", Category: "web", Username: "alice", Password: "s3cret"}); err != nil {
		t.Fatalf("注册失败: %v", err)
	}
	result, err := svc.Login(ctx, LoginRequest{Username: "alice", Password: "s3cret"})
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}
	if result.Username != "alice" || result.Token == "" {
		t.Errorf("登录结果错误: %+v", result)
	}
	if result.ExpiresIn != 7200 {
		t.Errorf("ExpiresIn 应为 7200，实际为 %d", result.ExpiresIn)
	}
	claims, err := auth.VerifyToken(result.Token, "test-secret")
	if err != nil || claims.Username != "alice" {
		t.Errorf("登录 token 应可验证: (%v, %v)", claims, err)
	}
}

func TestLoginLegacyMD5Upgrade(t *testing.T) {
	svc := testService(t)
	ctx := context.Background()

	// 模拟从旧库导入的 MD5 用户。
	db, ok := svc.Store.(*store.Store)
	if !ok {
		t.Fatal("测试需要真实 Store")
	}
	if _, err := db.CreateUser(ctx, "olduser", auth.MD5Upper("oldpass")); err != nil {
		t.Fatalf("预置 MD5 用户失败: %v", err)
	}

	result, err := svc.Login(ctx, LoginRequest{Username: "olduser", Password: "oldpass"})
	if err != nil {
		t.Fatalf("MD5 老用户应可登录，实际错误: %v", err)
	}
	if result.Token == "" {
		t.Error("MD5 登录应签发 token")
	}

	// 登录后哈希应透明升级为 bcrypt。
	u, err := db.GetUserByUsername(ctx, "olduser")
	if err != nil {
		t.Fatalf("读取用户失败: %v", err)
	}
	if auth.IsMD5Hash(u.PasswordHash) {
		t.Errorf("登录后哈希应升级为 bcrypt，实际仍为 MD5: %q", u.PasswordHash)
	}
	if err := auth.Verify(u.PasswordHash, "oldpass"); err != nil {
		t.Errorf("升级后哈希应可验证: %v", err)
	}

	// 升级后再次登录走 bcrypt 路径。
	if _, err := svc.Login(ctx, LoginRequest{Username: "olduser", Password: "oldpass"}); err != nil {
		t.Errorf("升级后登录失败: %v", err)
	}
	// 错误密码仍拒绝。
	if _, err := svc.Login(ctx, LoginRequest{Username: "olduser", Password: "wrong"}); err == nil {
		t.Error("错误密码应拒绝，实际通过")
	}
}

func TestLoginRejects(t *testing.T) {
	svc := testService(t)
	ctx := context.Background()

	if _, err := svc.Register(ctx, RegisterRequest{Name: "n", Category: "web", Username: "alice", Password: "right"}); err != nil {
		t.Fatalf("注册失败: %v", err)
	}
	cases := map[string]LoginRequest{
		"错误密码": {Username: "alice", Password: "wrong"},
		"未知用户": {Username: "ghost", Password: "whatever"},
		"缺少密码": {Username: "alice"},
		"缺少用户": {Password: "x"},
	}
	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.Login(ctx, req)
			var appErr *Error
			if !errors.As(err, &appErr) {
				t.Fatalf("%s 应返回应用错误，实际为 %v", name, err)
			}
			if appErr.Code != 401 && appErr.Code != 400 {
				t.Errorf("%s 应返回 400/401，实际为 %d", name, appErr.Code)
			}
		})
	}
}

func TestCreateApp(t *testing.T) {
	svc := testService(t)
	ctx := context.Background()

	app, err := svc.CreateApp(ctx, "alice", CreateAppRequest{Name: "extra", Category: "h5", Domain: "x.com"})
	if err != nil {
		t.Fatalf("创建应用失败: %v", err)
	}
	if app.AppID != "800800" || app.Username != "alice" || app.Status != "active" {
		t.Errorf("创建应用结果错误: %+v", app)
	}
	if app.SecretKey == "" {
		t.Error("创建应用应返回 secret_key")
	}

	_, err = svc.CreateApp(ctx, "alice", CreateAppRequest{})
	var appErr *Error
	if !errors.As(err, &appErr) || appErr.Code != 400 {
		t.Errorf("缺少 name 应返回 400，实际为 %v", err)
	}
}
