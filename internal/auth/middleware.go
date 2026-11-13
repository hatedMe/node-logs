package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/hatedMe/node-logs/internal/httpx"
	"github.com/hatedMe/node-logs/internal/store"
)

type usernameKey struct{}
type appKey struct{}

// ContextWithUsername 向上下文注入用户名（供测试与中间件使用）。
func ContextWithUsername(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, usernameKey{}, username)
}

// UsernameFromContext 从上下文读取用户名；未认证时返回空字符串。
func UsernameFromContext(ctx context.Context) string {
	username, _ := ctx.Value(usernameKey{}).(string)
	return username
}

// AppFromContext 从上下文读取应用；不存在时返回 nil。
func AppFromContext(ctx context.Context) *store.App {
	app, _ := ctx.Value(appKey{}).(*store.App)
	return app
}

// AuthMiddleware 校验 JWT；通过后向上下文注入用户名。
func AuthMiddleware(secret string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" {
			httpx.WriteError(w, http.StatusUnauthorized, "缺少登录 token")
			return
		}
		claims, err := VerifyToken(token, secret)
		if err != nil {
			httpx.WriteError(w, http.StatusUnauthorized, "登录已过期或无效")
			return
		}
		next.ServeHTTP(w, r.WithContext(ContextWithUsername(r.Context(), claims.Username)))
	})
}

func bearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

// AppGetter 按 ID 读取应用；*store.Store 已实现该接口。
type AppGetter interface {
	GetAppByID(ctx context.Context, appID string) (*store.App, error)
}

// RequireAppOwner 校验应用存在且属主为当前登录用户；通过后向上下文注入应用。
// 必须串联在 AuthMiddleware 之后。
func RequireAppOwner(getter AppGetter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username := UsernameFromContext(r.Context())
		if username == "" {
			httpx.WriteError(w, http.StatusUnauthorized, "缺少登录 token")
			return
		}
		appID := r.URL.Query().Get("app_id")
		if appID == "" {
			httpx.WriteError(w, http.StatusBadRequest, "缺少 app_id 参数")
			return
		}
		app, err := getter.GetAppByID(r.Context(), appID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				httpx.WriteError(w, http.StatusNotFound, "应用不存在")
			} else {
				httpx.WriteError(w, http.StatusServiceUnavailable, "应用校验暂不可用")
			}
			return
		}
		if app.Username != username {
			httpx.WriteError(w, http.StatusForbidden, "无权访问该应用")
			return
		}
		ctx := context.WithValue(r.Context(), appKey{}, app)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
