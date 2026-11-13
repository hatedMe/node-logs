package collector

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/hatedMe/node-logs/internal/httpx"
	"github.com/hatedMe/node-logs/internal/store"
)

// ErrUnknownApp 表示应用未注册。
var ErrUnknownApp = errors.New("collector: 未知应用")

// ErrAppDisabled 表示应用已停用。
var ErrAppDisabled = errors.New("collector: 应用已停用")

// AppValidator 校验应用可用性。
type AppValidator interface {
	ValidateApp(ctx context.Context, appID string) error
}

// Publisher 将事件写入队列。
type Publisher interface {
	Publish(ctx context.Context, event store.RawEvent) (string, error)
}

// CollectorHandler 是采集 HTTP 入口：校验 → 应用检查 → 写队列 → 204。
// 热路径只访问内存缓存与 Redis，不访问 PostgreSQL。
type CollectorHandler struct {
	Normalizer *EventNormalizer
	Validator  Validator
	Apps       AppValidator
	Publisher  Publisher
	// PublishTimeout 是单次队列写入超时；<=0 时默认为 2 秒。
	PublishTimeout time.Duration
}

func (h *CollectorHandler) publishTimeout() time.Duration {
	if h.PublishTimeout <= 0 {
		return 2 * time.Second
	}
	return h.PublishTimeout
}

// maxCollectBodyBytes 是采集请求体上限（1MB），防止超大请求体耗尽内存。
const maxCollectBodyBytes = 1 << 20

// Collect 支持 GET 查询参数与 POST（表单/JSON/sendBeacon）。
func (h *CollectorHandler) Collect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "仅支持 GET 与 POST")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxCollectBodyBytes)
	event, err := h.Normalizer.Normalize(r)
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			httpx.WriteError(w, http.StatusBadRequest, "请求体超过上限（1MB）")
			return
		}
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.Validator.Validate(event); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.Apps.ValidateApp(r.Context(), event.AppID); err != nil {
		switch {
		case errors.Is(err, ErrAppDisabled):
			httpx.WriteError(w, http.StatusForbidden, err.Error())
		case errors.Is(err, ErrUnknownApp):
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
		default:
			httpx.WriteError(w, http.StatusServiceUnavailable, "应用校验暂不可用")
		}
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.publishTimeout())
	defer cancel()
	if _, err := h.Publisher.Publish(ctx, event); err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "队列写入失败")
		return
	}

	if r.URL.Query().Get("fmt") == "gif" {
		w.Header().Set("Content-Type", "image/gif")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(transparentGIF)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// transparentGIF 是 1x1 透明 GIF，供图片 Beacon 兼容场景使用。
var transparentGIF = []byte{
	0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00, 0x01, 0x00,
	0x80, 0x00, 0x00, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x21,
	0xF9, 0x04, 0x01, 0x00, 0x00, 0x00, 0x00, 0x2C, 0x00, 0x00,
	0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02, 0x44,
	0x01, 0x00, 0x3B,
}

// AppLister 是应用全量的读取源；*store.Store 已实现该接口。
type AppLister interface {
	ListApps(ctx context.Context) ([]store.App, error)
}

// CachingAppValidator 缓存应用状态，TTL 内不访问数据库。
// 缓存未命中且冷却过期时会同步刷新一次，使新注册应用尽快可用。
// 刷新失败时拒绝写入（fail-closed），绝不用旧缓存放行已停用或无法确认的应用；
// 失败带冷却退避，避免冲击故障中的数据库。
type CachingAppValidator struct {
	mu       sync.RWMutex
	lister   AppLister
	ttl      time.Duration
	apps     map[string]string
	loaded   bool
	loadedAt time.Time
	// MissRefreshCooldown 是未命中触发刷新的冷却时间；<=0 时默认为 5 秒。
	MissRefreshCooldown time.Duration
	lastMissRefresh     time.Time
	// RefreshErrorCooldown 是刷新失败的冷却时间；<=0 时默认为 10 秒。
	RefreshErrorCooldown time.Duration
	lastErrAt            time.Time
	lastErr              error
}

// defaultMissRefreshCooldown 是未命中刷新的默认冷却时间。
const defaultMissRefreshCooldown = 5 * time.Second

// defaultRefreshErrorCooldown 是刷新失败的默认冷却时间。
const defaultRefreshErrorCooldown = 10 * time.Second

// NewCachingAppValidator 创建缓存校验器。
func NewCachingAppValidator(lister AppLister, ttl time.Duration) *CachingAppValidator {
	return &CachingAppValidator{lister: lister, ttl: ttl, apps: map[string]string{}}
}

func decideApp(status string, ok bool) error {
	if !ok {
		return ErrUnknownApp
	}
	if status != "active" {
		return ErrAppDisabled
	}
	return nil
}

// ValidateApp 校验应用存在且处于 active 状态。
// 状态无法确认（缓存过期且刷新失败）时返回错误，调用方应拒绝写入。
func (v *CachingAppValidator) ValidateApp(ctx context.Context, appID string) error {
	v.mu.RLock()
	hit, err := v.cached(appID)
	v.mu.RUnlock()
	if hit {
		return err
	}

	v.mu.Lock()
	defer v.mu.Unlock()
	if hit, err := v.cached(appID); hit {
		return err
	}
	apps, err := v.lister.ListApps(ctx)
	if err != nil {
		v.lastErrAt = time.Now()
		v.lastErr = err
		return err
	}
	next := make(map[string]string, len(apps))
	for _, app := range apps {
		next[app.AppID] = app.Status
	}
	v.apps = next
	v.loaded = true
	v.loadedAt = time.Now()
	v.lastMissRefresh = time.Now()
	v.lastErrAt = time.Time{}
	v.lastErr = nil
	status, ok := v.apps[appID]
	return decideApp(status, ok)
}

// cached 用缓存状态裁决请求；hit=false 表示需要刷新。
// 调用方必须持有读锁或写锁。
func (v *CachingAppValidator) cached(appID string) (bool, error) {
	if v.loaded && time.Since(v.loadedAt) < v.ttl {
		status, ok := v.apps[appID]
		if ok {
			return true, decideApp(status, ok)
		}
		if time.Since(v.lastMissRefresh) < v.missCooldown() {
			return true, ErrUnknownApp
		}
		// 未命中且冷却过期：刷新（除非错误退避中）。
	}
	// 缓存过期或未加载：刷新（除非错误退避中）。
	if v.lastErr != nil && time.Since(v.lastErrAt) < v.errCooldown() {
		return true, v.lastErr
	}
	return false, nil
}

func (v *CachingAppValidator) missCooldown() time.Duration {
	if v.MissRefreshCooldown <= 0 {
		return defaultMissRefreshCooldown
	}
	return v.MissRefreshCooldown
}

func (v *CachingAppValidator) errCooldown() time.Duration {
	if v.RefreshErrorCooldown <= 0 {
		return defaultRefreshErrorCooldown
	}
	return v.RefreshErrorCooldown
}
