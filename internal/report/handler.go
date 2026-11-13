package report

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/hatedMe/node-logs/internal/auth"
	"github.com/hatedMe/node-logs/internal/httpx"
)

// Handler 是报表 HTTP 入口；必须串联在认证与应用属主中间件之后。
type Handler struct {
	Service *Service
	// Timeout 是单次查询超时；<=0 时默认为 5 秒。
	Timeout time.Duration
}

func (h *Handler) timeout() time.Duration {
	if h.Timeout <= 0 {
		return 5 * time.Second
	}
	return h.Timeout
}

func (h *Handler) appID(w http.ResponseWriter, r *http.Request) (string, context.Context, context.CancelFunc) {
	app := auth.AppFromContext(r.Context())
	if app == nil {
		httpx.WriteError(w, http.StatusUnauthorized, "缺少登录 token")
		return "", nil, func() {}
	}
	ctx, cancel := context.WithTimeout(r.Context(), h.timeout())
	return app.AppID, ctx, cancel
}

// Overview 处理 GET /api/v1/reports/overview。
func (h *Handler) Overview(w http.ResponseWriter, r *http.Request) {
	appID, ctx, cancel := h.appID(w, r)
	if appID == "" {
		return
	}
	defer cancel()
	q := r.URL.Query()
	res, err := h.Service.Overview(ctx, appID, q.Get("from"), q.Get("to"))
	if err != nil {
		writeReportError(w, err)
		return
	}
	httpx.WriteSuccess(w, res)
}

// Trend 处理 GET /api/v1/reports/trend。
func (h *Handler) Trend(w http.ResponseWriter, r *http.Request) {
	appID, ctx, cancel := h.appID(w, r)
	if appID == "" {
		return
	}
	defer cancel()
	q := r.URL.Query()
	res, err := h.Service.Trend(ctx, appID, q.Get("from"), q.Get("to"))
	if err != nil {
		writeReportError(w, err)
		return
	}
	httpx.WriteSuccess(w, res)
}

// Pages 处理 GET /api/v1/reports/pages。
func (h *Handler) Pages(w http.ResponseWriter, r *http.Request) {
	appID, ctx, cancel := h.appID(w, r)
	if appID == "" {
		return
	}
	defer cancel()
	q := r.URL.Query()
	page, err := atoiOr(q.Get("page"), 1)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "page 非法")
		return
	}
	pageSize, err := atoiOr(q.Get("page_size"), 20)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "page_size 非法")
		return
	}
	res, err := h.Service.Pages(ctx, appID, q.Get("from"), q.Get("to"), q.Get("sort"), page, pageSize)
	if err != nil {
		writeReportError(w, err)
		return
	}
	httpx.WriteSuccess(w, res)
}

// Devices 处理 GET /api/v1/reports/devices。
func (h *Handler) Devices(w http.ResponseWriter, r *http.Request) {
	appID, ctx, cancel := h.appID(w, r)
	if appID == "" {
		return
	}
	defer cancel()
	q := r.URL.Query()
	res, err := h.Service.Devices(ctx, appID, q.Get("from"), q.Get("to"))
	if err != nil {
		writeReportError(w, err)
		return
	}
	httpx.WriteSuccess(w, res)
}

// Sources 处理 GET /api/v1/reports/sources；来源维度暂未采集，稳定返回空数组。
func (h *Handler) Sources(w http.ResponseWriter, r *http.Request) {
	if auth.AppFromContext(r.Context()) == nil {
		httpx.WriteError(w, http.StatusUnauthorized, "缺少登录 token")
		return
	}
	httpx.WriteSuccess(w, []any{})
}

func atoiOr(raw string, def int) (int, error) {
	if raw == "" {
		return def, nil
	}
	return strconv.Atoi(raw)
}

func writeReportError(w http.ResponseWriter, err error) {
	var rerr *Error
	if errors.As(err, &rerr) {
		httpx.WriteError(w, rerr.Code, rerr.Msg)
		return
	}
	httpx.WriteError(w, http.StatusInternalServerError, "服务器内部错误")
}
