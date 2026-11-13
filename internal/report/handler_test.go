package report

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hatedMe/node-logs/internal/auth"
	"github.com/hatedMe/node-logs/internal/store"
)

type stubAppGetter struct {
	app *store.App
	err error
}

func (s *stubAppGetter) GetAppByID(_ context.Context, _ string) (*store.App, error) {
	return s.app, s.err
}

func testChain(t *testing.T, q *stubQuerier, getter *stubAppGetter, h http.HandlerFunc) http.Handler {
	t.Helper()
	token, err := auth.Sign("test-secret", "alice", time.Hour)
	if err != nil {
		t.Fatalf("Sign 失败: %v", err)
	}
	_ = q
	authed := auth.AuthMiddleware("test-secret", auth.RequireAppOwner(getter, h))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Set("Authorization", "Bearer "+token)
		authed.ServeHTTP(w, r)
	})
}

func ownedGetter() *stubAppGetter {
	return &stubAppGetter{app: &store.App{AppID: "800800", Username: "alice", Status: "active"}}
}

func getJSON(t *testing.T, h http.Handler, path string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	res := rec.Result()
	defer res.Body.Close()
	var body map[string]any
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("响应应为合法 JSON: %v", err)
	}
	return res.StatusCode, body
}

func TestHandlerOverviewOK(t *testing.T) {
	q := &stubQuerier{overview: store.OverviewRow{PV: 10, UV: 5, IPCount: 4, Sessions: 2, Bounces: 1}}
	h := &Handler{Service: testService(q)}
	chain := testChain(t, q, ownedGetter(), h.Overview)

	status, body := getJSON(t, chain, "/api/v1/reports/overview?app_id=800800&from=2026-11-01&to=2026-11-12")
	if status != http.StatusOK {
		t.Fatalf("总览应返回 200，实际为 %d：%v", status, body)
	}
	data, _ := body["data"].(map[string]any)
	if data["pv"] != float64(10) || data["bounce_rate"] != 0.5 {
		t.Errorf("总览数据错误: %v", data)
	}
}

func TestHandlerForbidden(t *testing.T) {
	q := &stubQuerier{}
	h := &Handler{Service: testService(q)}
	getter := &stubAppGetter{app: &store.App{AppID: "800800", Username: "bob"}}
	chain := testChain(t, q, getter, h.Overview)

	status, _ := getJSON(t, chain, "/api/v1/reports/overview?app_id=800800&from=2026-11-01&to=2026-11-12")
	if status != http.StatusForbidden {
		t.Fatalf("跨用户访问应返回 403，实际为 %d", status)
	}
}

func TestHandlerMissingAppID(t *testing.T) {
	q := &stubQuerier{}
	h := &Handler{Service: testService(q)}
	chain := testChain(t, q, ownedGetter(), h.Overview)

	status, _ := getJSON(t, chain, "/api/v1/reports/overview?from=2026-11-01&to=2026-11-12")
	if status != http.StatusBadRequest {
		t.Fatalf("缺失 app_id 应返回 400，实际为 %d", status)
	}
}

func TestHandlerBadDate(t *testing.T) {
	q := &stubQuerier{}
	h := &Handler{Service: testService(q)}
	chain := testChain(t, q, ownedGetter(), h.Overview)

	status, _ := getJSON(t, chain, "/api/v1/reports/overview?app_id=800800&from=bad&to=2026-11-12")
	if status != http.StatusBadRequest {
		t.Fatalf("非法日期应返回 400，实际为 %d", status)
	}
}

func TestHandlerWithoutAuth(t *testing.T) {
	q := &stubQuerier{}
	h := &Handler{Service: testService(q)}
	chain := auth.AuthMiddleware("test-secret", auth.RequireAppOwner(ownedGetter(), http.HandlerFunc(h.Overview)))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/overview?app_id=800800", nil)
	rec := httptest.NewRecorder()
	chain.ServeHTTP(rec, req)
	if rec.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("未认证应返回 401，实际为 %d", rec.Result().StatusCode)
	}
}

func TestHandlerTrendOK(t *testing.T) {
	q := &stubQuerier{trend: []store.TrendRow{
		{Date: time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC), PV: 10},
	}}
	h := &Handler{Service: testService(q)}
	chain := testChain(t, q, ownedGetter(), h.Trend)

	status, body := getJSON(t, chain, "/api/v1/reports/trend?app_id=800800&from=2026-11-12&to=2026-11-12")
	if status != http.StatusOK {
		t.Fatalf("趋势应返回 200，实际为 %d：%v", status, body)
	}
	data, _ := body["data"].([]any)
	if len(data) != 1 {
		t.Errorf("趋势数据错误: %v", body)
	}
}

func TestHandlerPagesOK(t *testing.T) {
	q := &stubQuerier{pages: []store.PageRow{{URL: "/a", PV: 5}}, total: 1}
	h := &Handler{Service: testService(q)}
	chain := testChain(t, q, ownedGetter(), h.Pages)

	status, body := getJSON(t, chain, "/api/v1/reports/pages?app_id=800800&from=2026-11-01&to=2026-11-12&page=1&page_size=10&sort=pv")
	if status != http.StatusOK {
		t.Fatalf("页面排行应返回 200，实际为 %d：%v", status, body)
	}
	data, _ := body["data"].(map[string]any)
	if data["total"] != float64(1) {
		t.Errorf("页面总数错误: %v", data)
	}
}

func TestHandlerPagesBadParam(t *testing.T) {
	q := &stubQuerier{}
	h := &Handler{Service: testService(q)}
	chain := testChain(t, q, ownedGetter(), h.Pages)

	status, _ := getJSON(t, chain, "/api/v1/reports/pages?app_id=800800&from=2026-11-01&to=2026-11-12&page=abc")
	if status != http.StatusBadRequest {
		t.Fatalf("非法分页参数应返回 400，实际为 %d", status)
	}
}

func TestHandlerDevicesOK(t *testing.T) {
	q := &stubQuerier{devices: []store.DeviceRow{{Dimension: "browser", Value: "Chrome", PV: 8}}}
	h := &Handler{Service: testService(q)}
	chain := testChain(t, q, ownedGetter(), h.Devices)

	status, body := getJSON(t, chain, "/api/v1/reports/devices?app_id=800800&from=2026-11-01&to=2026-11-12")
	if status != http.StatusOK {
		t.Fatalf("设备统计应返回 200，实际为 %d：%v", status, body)
	}
}

func TestHandlerSourcesEmpty(t *testing.T) {
	q := &stubQuerier{}
	h := &Handler{Service: testService(q)}
	chain := testChain(t, q, ownedGetter(), h.Sources)

	status, body := getJSON(t, chain, "/api/v1/reports/sources?app_id=800800&from=2026-11-01&to=2026-11-12")
	if status != http.StatusOK {
		t.Fatalf("来源统计应返回 200，实际为 %d：%v", status, body)
	}
	data, _ := body["data"].([]any)
	if data == nil || len(data) != 0 {
		t.Errorf("来源暂无数据时应返回空数组: %v", body)
	}
}
