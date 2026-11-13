// Package httpx 提供统一的 HTTP JSON 响应格式。
package httpx

import (
	"encoding/json"
	"net/http"
)

// envelope 是全站统一的 JSON 包络，与旧 Node 接口的 {code,message,data} 语义一致。
type envelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// WriteJSON 以指定状态码写出任意 JSON 负载。
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// WriteSuccess 以 200 + 统一成功包络写出数据。
func WriteSuccess(w http.ResponseWriter, data any) {
	WriteJSON(w, http.StatusOK, envelope{Code: http.StatusOK, Message: "ok", Data: data})
}

// WriteError 以统一错误包络写出错误，包络 code 与 HTTP 状态码一致。
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, envelope{Code: status, Message: message, Data: nil})
}
