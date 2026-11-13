// Package collector 实现采集协议：旧埋点参数归一化、校验与发布。
package collector

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hatedMe/node-logs/internal/store"
)

// maxFutureSkew 允许事件时间超前服务端的上限，超出视为非法事件时间。
const maxFutureSkew = 24 * time.Hour

// maxUABytes UA 上限（截断而非拒绝，避免正常长 UA 被丢弃）。
const maxUABytes = 1024

// knownParams 是归一化消费的参数名；其余参数原样进入 Extra。
// 注意：ty / scl / random 等遗留参数有意不在此列，以便进入 Extra 存档。
var knownParams = map[string]bool{
	"dm": true, "pvid": true, "uvmark": true, "url": true, "arg": true,
	"v": true, "scr": true, "lg": true, "tz": true, "d": true,
	"event_id": true, "session_id": true, "fmt": true,
}

// EventNormalizer 将旧埋点参数转换为统一事件，并补充服务端字段。
type EventNormalizer struct {
	IPSalt            string
	TrustProxyHeaders bool
	// Now 返回服务端当前时间；测试可注入固定时钟。
	Now func() time.Time
}

func (n *EventNormalizer) now() time.Time {
	if n.Now != nil {
		return n.Now()
	}
	return time.Now()
}

// Normalize 解析 GET 查询参数、表单与 POST JSON，返回统一事件。
func (n *EventNormalizer) Normalize(r *http.Request) (store.RawEvent, error) {
	now := n.now()
	params, err := requestParams(r)
	if err != nil {
		return store.RawEvent{}, err
	}

	appID := strings.TrimSpace(params.Get("pvid"))
	if appID == "" {
		return store.RawEvent{}, fmt.Errorf("缺少必填参数 pvid")
	}

	eventTime := now
	if raw := strings.TrimSpace(params.Get("d")); raw != "" {
		ms, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || ms <= 0 {
			return store.RawEvent{}, fmt.Errorf("非法事件时间 d=%q", raw)
		}
		eventTime = time.UnixMilli(ms).UTC()
		if eventTime.After(now.Add(maxFutureSkew)) {
			return store.RawEvent{}, fmt.Errorf("事件时间超出允许范围 d=%q", raw)
		}
	}

	eventID := strings.TrimSpace(params.Get("event_id"))
	if eventID == "" {
		eventID, err = generateID()
		if err != nil {
			return store.RawEvent{}, fmt.Errorf("生成 event_id 失败: %w", err)
		}
	}

	visitorID := strings.TrimSpace(params.Get("uvmark"))
	if visitorID == "" || strings.EqualFold(visitorID, "null") || strings.EqualFold(visitorID, "undefined") {
		visitorID, err = generateID()
		if err != nil {
			return store.RawEvent{}, fmt.Errorf("生成 visitor_id 失败: %w", err)
		}
	}

	return store.RawEvent{
		EventID:    eventID,
		AppID:      appID,
		VisitorID:  visitorID,
		SessionID:  strings.TrimSpace(params.Get("session_id")),
		URL:        params.Get("url"),
		Arg:        params.Get("arg"),
		Version:    params.Get("v"),
		Screen:     params.Get("scr"),
		Lang:       params.Get("lg"),
		TZ:         params.Get("tz"),
		Domain:     params.Get("dm"),
		EventTime:  eventTime,
		ReceivedAt: now,
		UA:         truncateRunes(r.Header.Get("User-Agent"), maxUABytes),
		IPHash:     HashIP(ClientIP(r, n.TrustProxyHeaders), n.IPSalt),
		Extra:      extraParams(params),
	}, nil
}

// requestParams 合并 URL 查询、表单与 JSON 请求体；JSON 优先级最高。
func requestParams(r *http.Request) (url.Values, error) {
	// ParseForm 合并 URL 查询与 urlencoded 表单；JSON 体不受影响。
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("解析请求参数失败: %w", err)
	}
	params := r.Form
	if !isJSON(r) {
		return params, nil
	}
	decoder := json.NewDecoder(r.Body)
	decoder.UseNumber()
	var body map[string]any
	if err := decoder.Decode(&body); err != nil {
		return nil, fmt.Errorf("解析 JSON 请求体失败: %w", err)
	}
	for k, v := range body {
		params.Set(k, jsonScalar(v))
	}
	return params, nil
}

func isJSON(r *http.Request) bool {
	return strings.Contains(strings.ToLower(r.Header.Get("Content-Type")), "json")
}

func jsonScalar(v any) string {
	switch v := v.(type) {
	case nil:
		return ""
	case string:
		return v
	case json.Number:
		return v.String()
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case bool:
		return strconv.FormatBool(v)
	default:
		return fmt.Sprint(v)
	}
}

// extraParams 收集归一化未消费的参数，上限 32 项、每项 256 字符。
func extraParams(params url.Values) map[string]string {
	extra := map[string]string{}
	for k, vs := range params {
		if knownParams[k] || len(vs) == 0 {
			continue
		}
		if len(extra) >= 32 {
			break
		}
		extra[k] = truncateRunes(vs[0], 256)
	}
	return extra
}

func generateID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func truncateRunes(s string, max int) string {
	if len(s) <= max {
		return s
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}

// ClientIP 按配置解析客户端 IP。默认只信任连接地址，忽略全部代理头；
// 仅当 trustProxy 为 true 时按 X-Forwarded-For（最左）→ X-Real-IP → 连接地址回退。
func ClientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			if first := strings.TrimSpace(strings.Split(xff, ",")[0]); first != "" {
				return first
			}
		}
		if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
			return xri
		}
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// HashIP 用 HMAC-SHA256（盐为 key）对 IP 求哈希，返回 64 位十六进制。
// 明文 IP 只用于求哈希，不进入事件持久化字段。
func HashIP(ip, salt string) string {
	mac := hmac.New(sha256.New, []byte(salt))
	mac.Write([]byte(ip))
	return hex.EncodeToString(mac.Sum(nil))
}
