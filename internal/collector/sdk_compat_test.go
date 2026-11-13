package collector

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

// TestSDKGeneratedQueryAccepted 端到端：Node SDK 真实输出的查询串必须被 Go 采集器接受。
// 固件 testdata/sdk_query.txt 由 src/index.js 在模拟浏览器环境下生成（见生成命令注释），
// 保证"旧版 SDK 无需改参数即可上报"的验收标准。
//
// 生成固件：
//
//	node -e "global.window={location:{host:'example.com',pathname:'/home',search:'?a=1',origin:'https://example.com'},screen:{width:1920,height:1080},navigator:{language:'zh-CN'},localStorage:{getItem:()=>null,setItem:()=>{}},sessionStorage:{getItem:()=>null,setItem:()=>{}}};const sdk=require('./src/index.js');console.log(sdk.__test__.serialize(sdk.__test__.buildParams()))" > internal/collector/testdata/sdk_query.txt
func TestSDKGeneratedQueryAccepted(t *testing.T) {
	raw, err := os.ReadFile("testdata/sdk_query.txt")
	if err != nil {
		t.Fatalf("读取 SDK 查询固件失败: %v", err)
	}
	query := strings.TrimSpace(string(raw))
	if query == "" {
		t.Fatal("SDK 查询固件为空")
	}
	want, err := url.ParseQuery(query)
	if err != nil {
		t.Fatalf("固件不是合法查询串: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/pingd/front/web?"+query, nil)
	req.RemoteAddr = "203.0.113.7:4321"
	req.Header.Set("User-Agent", "sdk-compat-agent")
	pub := &fakePublisher{}
	h := testHandler(pub, &fakeApps{})
	rec := httptest.NewRecorder()
	h.Collect(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("SDK 查询应返回 204，实际 %d：%s", rec.Code, rec.Body.String())
	}
	if len(pub.events) != 1 {
		t.Fatalf("应发布 1 个事件，实际 %d 个", len(pub.events))
	}
	got := pub.events[0]
	if got.AppID != "800800" || got.URL != "/home" || got.Arg != "?a=1" {
		t.Errorf("旧参数丢失: app=%q url=%q arg=%q", got.AppID, got.URL, got.Arg)
	}
	if got.EventID != want.Get("event_id") || got.EventID == "" {
		t.Errorf("event_id 未透传: %q", got.EventID)
	}
	if got.SessionID != want.Get("session_id") || got.SessionID == "" {
		t.Errorf("session_id 未透传: %q", got.SessionID)
	}
	if got.VisitorID != want.Get("uvmark") || got.VisitorID == "" {
		t.Errorf("uvmark 未透传: %q", got.VisitorID)
	}
	if wantIP := HashIP("203.0.113.7", "test-salt"); got.IPHash != wantIP {
		t.Errorf("ip_hash 错误: got %q want %q", got.IPHash, wantIP)
	}
	if got.UA != "sdk-compat-agent" {
		t.Errorf("UA 未透传: %q", got.UA)
	}
	// 旧版遗留参数 ty/scl/random 进入 Extra 存档，不丢失。
	if got.Extra["ty"] != "0" {
		t.Errorf("Extra[ty] 应为 0，实际 %q", got.Extra["ty"])
	}
	if got.Extra["scl"] == "" || got.Extra["random"] == "" {
		t.Errorf("Extra 应保留 scl/random，实际 %v", got.Extra)
	}
}
