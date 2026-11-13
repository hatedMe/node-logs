package app

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/hatedMe/node-logs/internal/auth"
	"github.com/hatedMe/node-logs/internal/httpx"
)

// Handler 是应用管理与认证的 HTTP 入口。
type Handler struct {
	Service *Service
}

// Register 处理 POST /api/v1/auth/register。
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	app, err := h.Service.Register(r.Context(), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteSuccess(w, map[string]string{"app_id": app.AppID})
}

// Login 处理 POST /api/v1/auth/login。
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	result, err := h.Service.Login(r.Context(), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteSuccess(w, result)
}

// CreateApp 处理 POST /api/v1/apps；需要认证中间件注入用户名。
func (h *Handler) CreateApp(w http.ResponseWriter, r *http.Request) {
	username := auth.UsernameFromContext(r.Context())
	if username == "" {
		httpx.WriteError(w, http.StatusUnauthorized, "缺少登录 token")
		return
	}
	var req CreateAppRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	app, err := h.Service.CreateApp(r.Context(), username, req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteSuccess(w, map[string]string{
		"app_id":     app.AppID,
		"secret_key": app.SecretKey,
		"name":       app.Name,
		"status":     app.Status,
	})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "请求体不是合法 JSON")
		return false
	}
	return true
}

func writeServiceError(w http.ResponseWriter, err error) {
	var appErr *Error
	if errors.As(err, &appErr) {
		httpx.WriteError(w, appErr.Code, appErr.Msg)
		return
	}
	httpx.WriteError(w, http.StatusInternalServerError, "服务器内部错误")
}
