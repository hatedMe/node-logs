package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hatedMe/node-logs/internal/auth"
)

func testHandler(t *testing.T) *Handler {
	t.Helper()
	return &Handler{Service: testService(t)}
}

func postJSON(t *testing.T, h http.HandlerFunc, path string, payload any) (int, map[string]any) {
	t.Helper()
	var buf bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&buf).Encode(payload); err != nil {
			t.Fatalf("编码请求体失败: %v", err)
		}
	}
	req := httptest.NewRequest(http.MethodPost, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h(rec, req)
	res := rec.Result()
	defer res.Body.Close()
	var body map[string]any
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("响应应为合法 JSON: %v", err)
	}
	return res.StatusCode, body
}

func TestHandlerRegister(t *testing.T) {
	h := testHandler(t)

	status, body := postJSON(t, h.Register, "/api/v1/auth/register", RegisterRequest{
		Name: "demo", Category: "web", Username: "alice", Password: "s3cret", Domain: "demo.com",
	})
	if status != http.StatusOK {
		t.Fatalf("注册应返回 200，实际为 %d：%v", status, body)
	}
	data, _ := body["data"].(map[string]any)
	if data["app_id"] != "800800" {
		t.Errorf("注册响应应包含 app_id=800800，实际为 %v", body)
	}

	status, _ = postJSON(t, h.Register, "/api/v1/auth/register", RegisterRequest{
		Name: "demo", Category: "web", Username: "bob",
	})
	if status != http.StatusBadRequest {
		t.Fatalf("缺少密码应返回 400，实际为 %d", status)
	}
}

func TestHandlerRegisterDuplicate(t *testing.T) {
	h := testHandler(t)
	req := RegisterRequest{Name: "n", Category: "web", Username: "dup", Password: "p"}
	if status, _ := postJSON(t, h.Register, "/api/v1/auth/register", req); status != http.StatusOK {
		t.Fatalf("首次注册应返回 200，实际为 %d", status)
	}
	status, body := postJSON(t, h.Register, "/api/v1/auth/register", req)
	if status != http.StatusConflict {
		t.Fatalf("重复注册应返回 409，实际为 %d：%v", status, body)
	}
}

func TestHandlerLogin(t *testing.T) {
	h := testHandler(t)
	if status, _ := postJSON(t, h.Register, "/api/v1/auth/register", RegisterRequest{
		Name: "n", Category: "web", Username: "alice", Password: "s3cret",
	}); status != http.StatusOK {
		t.Fatalf("预置注册失败，状态 %d", status)
	}

	status, body := postJSON(t, h.Login, "/api/v1/auth/login", LoginRequest{Username: "alice", Password: "s3cret"})
	if status != http.StatusOK {
		t.Fatalf("登录应返回 200，实际为 %d：%v", status, body)
	}
	data, _ := body["data"].(map[string]any)
	if data["token"] == "" || data["username"] != "alice" || data["expiresIn"] == nil {
		t.Errorf("登录响应应包含 token/username/expiresIn，实际为 %v", body)
	}

	status, _ = postJSON(t, h.Login, "/api/v1/auth/login", LoginRequest{Username: "alice", Password: "wrong"})
	if status != http.StatusUnauthorized {
		t.Fatalf("错误密码应返回 401，实际为 %d", status)
	}
}

func TestHandlerCreateApp(t *testing.T) {
	h := testHandler(t)

	payload, _ := json.Marshal(CreateAppRequest{Name: "extra", Category: "h5"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/apps", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.CreateApp(rec, req.WithContext(auth.ContextWithUsername(req.Context(), "alice")))

	res := rec.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("创建应用应返回 200，实际为 %d", res.StatusCode)
	}
	var body struct {
		Data struct {
			AppID     string `json:"app_id"`
			SecretKey string `json:"secret_key"`
		} `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("响应应为合法 JSON: %v", err)
	}
	if body.Data.AppID == "" || body.Data.SecretKey == "" {
		t.Errorf("创建应用应返回 app_id 与 secret_key，实际为 %+v", body.Data)
	}
}

func TestHandlerCreateAppWithoutAuth(t *testing.T) {
	h := testHandler(t)

	payload, _ := json.Marshal(CreateAppRequest{Name: "extra"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/apps", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.CreateApp(rec, req)

	if rec.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("未认证应返回 401，实际为 %d", rec.Result().StatusCode)
	}
}
