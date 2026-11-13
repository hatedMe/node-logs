package collector

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func testNormalizer() *EventNormalizer {
	return &EventNormalizer{
		IPSalt:            "test-salt",
		TrustProxyHeaders: false,
		Now:               func() time.Time { return time.Date(2026, 11, 12, 10, 0, 1, 0, time.UTC) },
	}
}

func TestNormalizeGETFullParams(t *testing.T) {
	n := testNormalizer()
	url := "/pingd/front/web?dm=example.com&pvid=800800&uvmark=visitor-1" +
		"&url=%2Fhome&arg=%3Fa%3D1&v=1.0.0&scr=1920x1080&lg=zh-cn&tz=-8" +
		"&d=1794477610000&ty=0&random=abc123&event_id=evt-1"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.RemoteAddr = "1.2.3.4:5678"
	req.Header.Set("User-Agent", "test-agent")

	event, err := n.Normalize(req)
	if err != nil {
		t.Fatalf("Normalize() 返回意外错误: %v", err)
	}

	if event.EventID != "evt-1" {
		t.Errorf("EventID 应透传 evt-1，实际为 %q", event.EventID)
	}
	if event.AppID != "800800" || event.VisitorID != "visitor-1" {
		t.Errorf("应用/访客解析错误: %+v", event)
	}
	if event.URL != "/home" || event.Arg != "?a=1" {
		t.Errorf("页面参数解析错误: url=%q arg=%q", event.URL, event.Arg)
	}
	if event.Version != "1.0.0" || event.Screen != "1920x1080" || event.Lang != "zh-cn" || event.TZ != "-8" {
		t.Errorf("环境参数解析错误: %+v", event)
	}
	if event.Domain != "example.com" {
		t.Errorf("Domain 应为 example.com，实际为 %q", event.Domain)
	}
	wantTime := time.Date(2026, 11, 12, 10, 0, 10, 0, time.UTC)
	if !event.EventTime.Equal(wantTime) {
		t.Errorf("EventTime 应为 %v，实际为 %v", wantTime, event.EventTime)
	}
	if !event.ReceivedAt.Equal(time.Date(2026, 11, 12, 10, 0, 1, 0, time.UTC)) {
		t.Errorf("ReceivedAt 应为服务端时间，实际为 %v", event.ReceivedAt)
	}
	if event.UA != "test-agent" {
		t.Errorf("UA 应透传，实际为 %q", event.UA)
	}
	if event.IPHash == "" || len(event.IPHash) != 64 {
		t.Errorf("IPHash 应为 64 位十六进制，实际为 %q", event.IPHash)
	}
	if event.Extra["ty"] != "0" || event.Extra["random"] != "abc123" {
		t.Errorf("遗留扩展参数应进入 Extra，实际为 %v", event.Extra)
	}
}

func TestNormalizeGeneratesIDs(t *testing.T) {
	n := testNormalizer()
	req := httptest.NewRequest(http.MethodGet, "/pingd/front/web?pvid=800800&url=%2Fa", nil)
	req.RemoteAddr = "1.2.3.4:5678"

	event, err := n.Normalize(req)
	if err != nil {
		t.Fatalf("Normalize() 返回意外错误: %v", err)
	}
	if event.EventID == "" {
		t.Error("缺失 event_id 时应服务端生成")
	}
	if event.VisitorID == "" {
		t.Error("缺失 uvmark 时应服务端生成访客标识")
	}
	if !event.EventTime.Equal(event.ReceivedAt) {
		t.Errorf("缺失 d 时 EventTime 应等于 ReceivedAt：%v vs %v", event.EventTime, event.ReceivedAt)
	}
}

func TestNormalizeNullVisitorID(t *testing.T) {
	// 旧版 SDK 首访 bug：localStorage 写入后仍返回 null，上报 uvmark=null。
	n := testNormalizer()
	for _, raw := range []string{"null", "undefined", "  "} {
		req := httptest.NewRequest(http.MethodGet, "/pingd/front/web?pvid=800800&uvmark="+url.QueryEscape(raw), nil)
		req.RemoteAddr = "1.2.3.4:5678"
		event, err := n.Normalize(req)
		if err != nil {
			t.Fatalf("uvmark=%q 时 Normalize() 返回意外错误: %v", raw, err)
		}
		if event.VisitorID == "" || event.VisitorID == "null" || event.VisitorID == "undefined" {
			t.Errorf("uvmark=%q 应被替换为生成的访客标识，实际为 %q", raw, event.VisitorID)
		}
	}
}

func TestNormalizeMissingPvid(t *testing.T) {
	n := testNormalizer()
	req := httptest.NewRequest(http.MethodGet, "/pingd/front/web?uvmark=v1", nil)
	req.RemoteAddr = "1.2.3.4:5678"

	if _, err := n.Normalize(req); err == nil {
		t.Fatal("缺失 pvid 时应返回错误，实际为 nil")
	} else if !strings.Contains(err.Error(), "pvid") {
		t.Errorf("错误信息应提及 pvid，实际为 %q", err.Error())
	}
}

func TestNormalizeBadEventTime(t *testing.T) {
	n := testNormalizer()
	cases := map[string]string{
		"非数字":  "abc",
		"负数":   "-5",
		"零":    "0",
		"未来过远": "4102444800000", // 2100-01-01
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/pingd/front/web?pvid=800800&d="+d, nil)
			req.RemoteAddr = "1.2.3.4:5678"
			if _, err := n.Normalize(req); err == nil {
				t.Errorf("d=%s 时应返回错误，实际为 nil", d)
			}
		})
	}
}

func TestNormalizeTruncatesUA(t *testing.T) {
	n := testNormalizer()
	req := httptest.NewRequest(http.MethodGet, "/pingd/front/web?pvid=800800", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	req.Header.Set("User-Agent", strings.Repeat("a", 5000))

	event, err := n.Normalize(req)
	if err != nil {
		t.Fatalf("Normalize() 返回意外错误: %v", err)
	}
	if len(event.UA) > 1024 {
		t.Errorf("超长 UA 应截断到 1024 以内，实际长度 %d", len(event.UA))
	}
}

func TestNormalizePOSTJSON(t *testing.T) {
	n := testNormalizer()
	body := `{"pvid":"800800","uvmark":"visitor-9","url":"/json","d":1794477610000,"event_id":"evt-json"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/collect", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "1.2.3.4:5678"

	event, err := n.Normalize(req)
	if err != nil {
		t.Fatalf("Normalize() 返回意外错误: %v", err)
	}
	if event.AppID != "800800" || event.VisitorID != "visitor-9" || event.URL != "/json" || event.EventID != "evt-json" {
		t.Errorf("POST JSON 解析错误: %+v", event)
	}
}

func TestClientIP(t *testing.T) {
	newReq := func() *http.Request {
		req := httptest.NewRequest(http.MethodGet, "/pingd/front/web?pvid=800800", nil)
		req.RemoteAddr = "1.2.3.4:5678"
		return req
	}

	t.Run("默认不信任代理头", func(t *testing.T) {
		req := newReq()
		req.Header.Set("X-Forwarded-For", "9.9.9.9")
		req.Header.Set("X-Real-IP", "8.8.8.8")
		if got := ClientIP(req, false); got != "1.2.3.4" {
			t.Errorf("默认应使用连接地址 1.2.3.4，实际为 %q", got)
		}
	})

	t.Run("开启信任后取 XFF 最左", func(t *testing.T) {
		req := newReq()
		req.Header.Set("X-Forwarded-For", "9.9.9.9, 10.0.0.1")
		if got := ClientIP(req, true); got != "9.9.9.9" {
			t.Errorf("应取 XFF 最左 9.9.9.9，实际为 %q", got)
		}
	})

	t.Run("开启信任后回退 X-Real-IP", func(t *testing.T) {
		req := newReq()
		req.Header.Set("X-Real-IP", "8.8.8.8")
		if got := ClientIP(req, true); got != "8.8.8.8" {
			t.Errorf("应回退 X-Real-IP，实际为 %q", got)
		}
	})

	t.Run("无端口地址", func(t *testing.T) {
		req := newReq()
		req.RemoteAddr = "1.2.3.4"
		if got := ClientIP(req, false); got != "1.2.3.4" {
			t.Errorf("应原样返回 1.2.3.4，实际为 %q", got)
		}
	})
}

func TestIPHashStableAndSalted(t *testing.T) {
	a := HashIP("1.2.3.4", "salt-a")
	b := HashIP("1.2.3.4", "salt-a")
	c := HashIP("1.2.3.4", "salt-b")
	d := HashIP("5.6.7.8", "salt-a")
	if a != b {
		t.Error("相同 IP+盐的哈希应稳定")
	}
	if a == c {
		t.Error("不同盐的哈希应不同")
	}
	if a == d {
		t.Error("不同 IP 的哈希应不同")
	}
	if len(a) != 64 {
		t.Errorf("哈希应为 64 位十六进制，实际为 %q", a)
	}
}
