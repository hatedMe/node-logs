package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hatedMe/node-logs/internal/store"
)

func authedRequest(token string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/apps", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}

func TestAuthMiddlewareOK(t *testing.T) {
	token, err := Sign("test-secret", "alice", time.Hour)
	if err != nil {
		t.Fatalf("Sign 失败: %v", err)
	}
	var gotUser string
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotUser = UsernameFromContext(r.Context())
	})
	rec := httptest.NewRecorder()

	AuthMiddleware("test-secret", next).ServeHTTP(rec, authedRequest(token))

	if rec.Result().StatusCode != http.StatusOK {
		t.Fatalf("合法 token 应放行，实际状态 %d", rec.Result().StatusCode)
	}
	if gotUser != "alice" {
		t.Errorf("上下文用户名应为 alice，实际为 %q", gotUser)
	}
}

func TestAuthMiddlewareRejects(t *testing.T) {
	expired, err := Sign("test-secret", "alice", -time.Second)
	if err != nil {
		t.Fatalf("Sign 失败: %v", err)
	}
	cases := map[string]string{
		"缺少 token": "",
		"非法 token": "bad.token.value",
		"过期 token": expired,
	}
	for name, token := range cases {
		t.Run(name, func(t *testing.T) {
			called := false
			next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })
			rec := httptest.NewRecorder()

			AuthMiddleware("test-secret", next).ServeHTTP(rec, authedRequest(token))

			if rec.Result().StatusCode != http.StatusUnauthorized {
				t.Errorf("%s 应返回 401，实际为 %d", name, rec.Result().StatusCode)
			}
			if called {
				t.Errorf("%s 不应放行到下游", name)
			}
		})
	}
}

type stubAppGetter struct {
	app *store.App
	err error
}

func (s *stubAppGetter) GetAppByID(_ context.Context, _ string) (*store.App, error) {
	return s.app, s.err
}

func appRequest(appID, username string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/overview?app_id="+appID, nil)
	if username != "" {
		req = req.WithContext(context.WithValue(req.Context(), usernameKey{}, username))
	}
	return req
}

func TestRequireAppOwnerOK(t *testing.T) {
	getter := &stubAppGetter{app: &store.App{AppID: "800800", Username: "alice", Status: "active"}}
	var gotApp *store.App
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotApp = AppFromContext(r.Context())
	})
	rec := httptest.NewRecorder()

	RequireAppOwner(getter, next).ServeHTTP(rec, appRequest("800800", "alice"))

	if rec.Result().StatusCode != http.StatusOK {
		t.Fatalf("属主访问应放行，实际状态 %d", rec.Result().StatusCode)
	}
	if gotApp == nil || gotApp.AppID != "800800" {
		t.Errorf("上下文应用错误: %+v", gotApp)
	}
}

func TestRequireAppOwnerForbidden(t *testing.T) {
	getter := &stubAppGetter{app: &store.App{AppID: "800800", Username: "alice", Status: "active"}}
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	rec := httptest.NewRecorder()

	RequireAppOwner(getter, next).ServeHTTP(rec, appRequest("800800", "bob"))

	if rec.Result().StatusCode != http.StatusForbidden {
		t.Fatalf("跨用户访问应返回 403，实际为 %d", rec.Result().StatusCode)
	}
}

func TestRequireAppOwnerNotFound(t *testing.T) {
	getter := &stubAppGetter{err: store.ErrNotFound}
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	rec := httptest.NewRecorder()

	RequireAppOwner(getter, next).ServeHTTP(rec, appRequest("no-such", "alice"))

	if rec.Result().StatusCode != http.StatusNotFound {
		t.Fatalf("未知应用应返回 404，实际为 %d", rec.Result().StatusCode)
	}
}

func TestRequireAppOwnerMissingParam(t *testing.T) {
	getter := &stubAppGetter{app: &store.App{AppID: "800800", Username: "alice"}}
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	rec := httptest.NewRecorder()

	RequireAppOwner(getter, next).ServeHTTP(rec, appRequest("", "alice"))

	if rec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("缺失 app_id 应返回 400，实际为 %d", rec.Result().StatusCode)
	}
}

func TestRequireAppOwnerWithoutAuth(t *testing.T) {
	getter := &stubAppGetter{app: &store.App{AppID: "800800", Username: "alice"}}
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	rec := httptest.NewRecorder()

	RequireAppOwner(getter, next).ServeHTTP(rec, appRequest("800800", ""))

	if rec.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("未认证应返回 401，实际为 %d", rec.Result().StatusCode)
	}
}

func TestRequireAppOwnerStoreError(t *testing.T) {
	getter := &stubAppGetter{err: errors.New("pg down")}
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	rec := httptest.NewRecorder()

	RequireAppOwner(getter, next).ServeHTTP(rec, appRequest("800800", "alice"))

	if rec.Result().StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("存储异常应返回 503，实际为 %d", rec.Result().StatusCode)
	}
}
