package collector

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hatedMe/node-logs/internal/store"
)

type fakePublisher struct {
	events []store.RawEvent
	err    error
}

func (f *fakePublisher) Publish(_ context.Context, event store.RawEvent) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	f.events = append(f.events, event)
	return "stream-id-1", nil
}

type fakeApps struct {
	err error
}

func (f *fakeApps) ValidateApp(_ context.Context, _ string) error { return f.err }

func testHandler(pub *fakePublisher, apps *fakeApps) *CollectorHandler {
	return &CollectorHandler{
		Normalizer:     &EventNormalizer{IPSalt: "test-salt"},
		Validator:      Validator{MaxURLLength: 2048},
		Apps:           apps,
		Publisher:      pub,
		PublishTimeout: 2 * time.Second,
	}
}

func TestCollectGET204(t *testing.T) {
	pub := &fakePublisher{}
	h := testHandler(pub, &fakeApps{})

	req := httptest.NewRequest(http.MethodGet,
		"/pingd/front/web?pvid=800800&uvmark=v1&url=%2Fhome&v=1.0.0&scr=1920x1080&lg=zh-cn&tz=-8&d=1794477610000", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	req.Header.Set("User-Agent", "test-agent")
	rec := httptest.NewRecorder()

	h.Collect(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("采集成功应返回 204，实际为 %d", res.StatusCode)
	}
	if len(pub.events) != 1 {
		t.Fatalf("应发布 1 条事件，实际 %d 条", len(pub.events))
	}
	got := pub.events[0]
	if got.AppID != "800800" || got.VisitorID != "v1" || got.URL != "/home" {
		t.Errorf("发布事件字段错误: %+v", got)
	}
	if got.IPHash == "" || got.EventID == "" {
		t.Errorf("服务端字段未补充: %+v", got)
	}
	if got.ReceivedAt.IsZero() {
		t.Error("ReceivedAt 未补充")
	}
}

func TestCollectPOSTJSON(t *testing.T) {
	pub := &fakePublisher{}
	h := testHandler(pub, &fakeApps{})

	body := `{"pvid":"800800","uvmark":"v9","url":"/json","d":1794477610000}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/collect", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "1.2.3.4:5678"
	rec := httptest.NewRecorder()

	h.Collect(rec, req)

	if rec.Result().StatusCode != http.StatusNoContent {
		t.Fatalf("POST JSON 采集应返回 204，实际为 %d", rec.Result().StatusCode)
	}
	if len(pub.events) != 1 || pub.events[0].URL != "/json" {
		t.Errorf("POST JSON 事件发布错误: %+v", pub.events)
	}
}

func TestCollectMissingPvid400(t *testing.T) {
	pub := &fakePublisher{}
	h := testHandler(pub, &fakeApps{})

	req := httptest.NewRequest(http.MethodGet, "/pingd/front/web?uvmark=v1", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	rec := httptest.NewRecorder()

	h.Collect(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("缺失 pvid 应返回 400，实际为 %d", res.StatusCode)
	}
	var body struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("错误响应应为合法 JSON: %v", err)
	}
	if !strings.Contains(body.Message, "pvid") {
		t.Errorf("错误信息应提及 pvid，实际为 %q", body.Message)
	}
	if len(pub.events) != 0 {
		t.Error("非法请求不应发布事件")
	}
}

func TestCollectUnknownApp(t *testing.T) {
	pub := &fakePublisher{}
	h := testHandler(pub, &fakeApps{err: ErrUnknownApp})

	req := httptest.NewRequest(http.MethodGet, "/pingd/front/web?pvid=unknown&uvmark=v1", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	rec := httptest.NewRecorder()

	h.Collect(rec, req)

	if rec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("未注册应用应返回 400，实际为 %d", rec.Result().StatusCode)
	}
	if len(pub.events) != 0 {
		t.Error("未注册应用不应发布事件")
	}
}

func TestCollectDisabledApp(t *testing.T) {
	pub := &fakePublisher{}
	h := testHandler(pub, &fakeApps{err: ErrAppDisabled})

	req := httptest.NewRequest(http.MethodGet, "/pingd/front/web?pvid=800800&uvmark=v1", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	rec := httptest.NewRecorder()

	h.Collect(rec, req)

	if rec.Result().StatusCode != http.StatusForbidden {
		t.Fatalf("停用应用应返回 403，实际为 %d", rec.Result().StatusCode)
	}
}

func TestCollectAppCheckError(t *testing.T) {
	pub := &fakePublisher{}
	h := testHandler(pub, &fakeApps{err: errors.New("pg down")})

	req := httptest.NewRequest(http.MethodGet, "/pingd/front/web?pvid=800800&uvmark=v1", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	rec := httptest.NewRecorder()

	h.Collect(rec, req)

	if rec.Result().StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("应用校验异常应返回 503，实际为 %d", rec.Result().StatusCode)
	}
}

func TestCollectGif(t *testing.T) {
	pub := &fakePublisher{}
	h := testHandler(pub, &fakeApps{})

	req := httptest.NewRequest(http.MethodGet, "/pingd/front/web?pvid=800800&uvmark=v1&fmt=gif", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	rec := httptest.NewRecorder()

	h.Collect(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("fmt=gif 应返回 200，实际为 %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != "image/gif" {
		t.Errorf("Content-Type 应为 image/gif，实际为 %q", ct)
	}
}

func TestCollectBodyTooLarge(t *testing.T) {
	pub := &fakePublisher{}
	h := testHandler(pub, &fakeApps{})

	big := strings.Repeat("a", 2<<20)
	cases := map[string]struct {
		contentType string
		body        string
	}{
		"超大表单":    {"application/x-www-form-urlencoded", "pvid=800800&arg=" + big},
		"超大 JSON": {"application/json", `{"pvid":"800800","arg":"` + big + `"}`},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/collect", strings.NewReader(c.body))
			req.Header.Set("Content-Type", c.contentType)
			req.RemoteAddr = "1.2.3.4:5678"
			rec := httptest.NewRecorder()

			h.Collect(rec, req)

			res := rec.Result()
			defer res.Body.Close()
			if res.StatusCode != http.StatusBadRequest {
				t.Fatalf("%s 应返回 400，实际为 %d", name, res.StatusCode)
			}
			var body struct {
				Message string `json:"message"`
			}
			if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
				t.Fatalf("%s 错误响应应为合法 JSON: %v", name, err)
			}
			if !strings.Contains(body.Message, "超过上限") {
				t.Errorf("%s 错误信息应明确指出超过上限，实际为 %q", name, body.Message)
			}
		})
	}
	if len(pub.events) != 0 {
		t.Error("超大请求不应发布事件")
	}
}

func TestCollectMethodNotAllowed(t *testing.T) {
	pub := &fakePublisher{}
	h := testHandler(pub, &fakeApps{})

	req := httptest.NewRequest(http.MethodPut, "/pingd/front/web?pvid=800800", nil)
	rec := httptest.NewRecorder()

	h.Collect(rec, req)

	if rec.Result().StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("PUT 应返回 405，实际为 %d", rec.Result().StatusCode)
	}
}

func TestCollectPublishError(t *testing.T) {
	pub := &fakePublisher{err: errors.New("redis down")}
	h := testHandler(pub, &fakeApps{})

	req := httptest.NewRequest(http.MethodGet, "/pingd/front/web?pvid=800800&uvmark=v1", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	rec := httptest.NewRecorder()

	h.Collect(rec, req)

	if rec.Result().StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("队列写入失败应返回 503，实际为 %d", rec.Result().StatusCode)
	}
}

// stubLister 模拟应用列表源，统计刷新次数。
type stubLister struct {
	apps  []store.App
	err   error
	calls int
}

func (s *stubLister) ListApps(_ context.Context) ([]store.App, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	return s.apps, nil
}

func TestCachingAppValidator(t *testing.T) {
	lister := &stubLister{apps: []store.App{
		{AppID: "800800", Status: "active"},
		{AppID: "800801", Status: "disabled"},
	}}
	v := NewCachingAppValidator(lister, time.Minute)
	ctx := context.Background()

	if err := v.ValidateApp(ctx, "800800"); err != nil {
		t.Errorf("已知应用应通过，实际错误: %v", err)
	}
	if err := v.ValidateApp(ctx, "800800"); err != nil {
		t.Errorf("缓存命中应通过，实际错误: %v", err)
	}
	if lister.calls != 1 {
		t.Errorf("缓存有效期内只应刷新 1 次，实际 %d 次", lister.calls)
	}
	if err := v.ValidateApp(ctx, "unknown"); !errors.Is(err, ErrUnknownApp) {
		t.Errorf("未知应用应返回 ErrUnknownApp，实际为 %v", err)
	}
	if err := v.ValidateApp(ctx, "800801"); !errors.Is(err, ErrAppDisabled) {
		t.Errorf("停用应用应返回 ErrAppDisabled，实际为 %v", err)
	}
	// 缓存命中未知应用同样不应触发刷新。
	if lister.calls != 1 {
		t.Errorf("缓存命中时不应刷新，实际刷新 %d 次", lister.calls)
	}
}

func TestCachingAppValidatorMissRefresh(t *testing.T) {
	lister := &stubLister{apps: []store.App{{AppID: "800800", Status: "active"}}}
	v := NewCachingAppValidator(lister, time.Minute)
	v.MissRefreshCooldown = time.Millisecond
	ctx := context.Background()

	if err := v.ValidateApp(ctx, "800800"); err != nil {
		t.Fatalf("已知应用应通过: %v", err)
	}
	// 缓存新鲜且刚加载：未知应用直接拒绝，不刷新。
	if err := v.ValidateApp(ctx, "800801"); !errors.Is(err, ErrUnknownApp) {
		t.Fatalf("未知应用应返回 ErrUnknownApp，实际为 %v", err)
	}
	if lister.calls != 1 {
		t.Fatalf("冷却期内 miss 不应刷新，实际刷新 %d 次", lister.calls)
	}
	// 新注册应用出现 + 冷却过期：下次 miss 触发刷新并通过。
	lister.apps = append(lister.apps, store.App{AppID: "800801", Status: "active"})
	time.Sleep(2 * time.Millisecond)
	if err := v.ValidateApp(ctx, "800801"); err != nil {
		t.Fatalf("刷新后新应用应通过: %v", err)
	}
	if lister.calls != 2 {
		t.Fatalf("冷却过期后 miss 应刷新 1 次，实际 %d 次", lister.calls)
	}
	// 刷新后立即再次 miss：冷却期内不再刷新。
	if err := v.ValidateApp(ctx, "unknown"); !errors.Is(err, ErrUnknownApp) {
		t.Fatalf("未知应用应返回 ErrUnknownApp，实际为 %v", err)
	}
	if lister.calls != 2 {
		t.Fatalf("冷却期内 miss 不应刷新，实际 %d 次", lister.calls)
	}
}

func TestCachingAppValidatorFailsClosedOnError(t *testing.T) {
	lister := &stubLister{apps: []store.App{{AppID: "800800", Status: "active"}}}
	v := NewCachingAppValidator(lister, time.Millisecond)
	v.RefreshErrorCooldown = time.Millisecond
	ctx := context.Background()

	if err := v.ValidateApp(ctx, "800800"); err != nil {
		t.Fatalf("首次校验失败: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	lister.err = errors.New("pg down")
	// 刷新失败必须拒绝写入（fail-closed），不得用旧缓存放行。
	if err := v.ValidateApp(ctx, "800800"); err == nil {
		t.Fatal("刷新失败应返回错误拒绝写入，实际为 nil")
	}
	if lister.calls != 2 {
		t.Fatalf("应刷新 1 次，实际 %d 次", lister.calls)
	}
	// 冷却期内不再冲击数据库，直接返回缓存的错误。
	if err := v.ValidateApp(ctx, "800800"); err == nil {
		t.Fatal("冷却期内应继续返回错误，实际为 nil")
	}
	if lister.calls != 2 {
		t.Fatalf("错误冷却期内不应重试，实际 %d 次", lister.calls)
	}
	// 冷却过期后重试。
	time.Sleep(2 * time.Millisecond)
	if err := v.ValidateApp(ctx, "800800"); err == nil {
		t.Fatal("冷却过期后应重试并返回错误，实际为 nil")
	}
	if lister.calls != 3 {
		t.Fatalf("冷却过期后应重试，实际 %d 次", lister.calls)
	}
	// 恢复后正常服务。
	lister.err = nil
	time.Sleep(2 * time.Millisecond)
	if err := v.ValidateApp(ctx, "800800"); err != nil {
		t.Fatalf("恢复后应正常服务，实际错误: %v", err)
	}
}
