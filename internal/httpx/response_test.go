package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()

	WriteError(rec, http.StatusBadRequest, "缺少 pvid 参数")

	res := rec.Result()
	defer res.Body.Close()

	if ct := res.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type 应为 application/json，实际为 %q", ct)
	}
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("状态码应为 400，实际为 %d", res.StatusCode)
	}

	var body struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("错误响应应为合法 JSON: %v", err)
	}
	if body.Code != http.StatusBadRequest {
		t.Errorf("code 应为 400，实际为 %d", body.Code)
	}
	if body.Message != "缺少 pvid 参数" {
		t.Errorf("message 解析错误，实际为 %q", body.Message)
	}
	if string(body.Data) != "null" {
		t.Errorf("data 应为 null，实际为 %s", string(body.Data))
	}
}

func TestWriteSuccess(t *testing.T) {
	rec := httptest.NewRecorder()

	WriteSuccess(rec, map[string]string{"app_id": "800800"})

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("状态码应为 200，实际为 %d", res.StatusCode)
	}

	var body struct {
		Code    int               `json:"code"`
		Message string            `json:"message"`
		Data    map[string]string `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("成功响应应为合法 JSON: %v", err)
	}
	if body.Code != 200 || body.Message != "ok" {
		t.Errorf("成功包络错误: code=%d message=%q", body.Code, body.Message)
	}
	if body.Data["app_id"] != "800800" {
		t.Errorf("data 透传错误: %v", body.Data)
	}
}

func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()

	WriteJSON(rec, http.StatusCreated, map[string]int{"n": 1})

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		t.Errorf("状态码应为 201，实际为 %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type 应为 application/json，实际为 %q", ct)
	}
}
