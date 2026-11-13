// Package queue 封装 Redis Stream 队列：采集端写入与消费端解析。
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/hatedMe/node-logs/internal/store"
)

// StreamProducer 向固定 Stream 写入事件。
type StreamProducer struct {
	client  *redis.Client
	stream  string
	timeout time.Duration
}

// NewStreamProducer 创建生产者并 Ping 验证连通性。
func NewStreamProducer(addr, password string, db int, stream string, timeout time.Duration) (*StreamProducer, error) {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	client := redis.NewClient(&redis.Options{Addr: addr, Password: password, DB: db})
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("queue: 连接 Redis 失败: %w", err)
	}
	return &StreamProducer{client: client, stream: stream, timeout: timeout}, nil
}

// Publish 将事件 XADD 到 Stream，返回 Stream 记录 ID。
func (p *StreamProducer) Publish(ctx context.Context, event store.RawEvent) (string, error) {
	values, err := EncodeEvent(event)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()
	id, err := p.client.XAdd(ctx, &redis.XAddArgs{Stream: p.stream, Values: values}).Result()
	if err != nil {
		return "", fmt.Errorf("queue: 写入 Stream 失败: %w", err)
	}
	return id, nil
}

// Len 返回队列长度，供监控与积压观察。
func (p *StreamProducer) Len(ctx context.Context) (int64, error) {
	return p.client.XLen(ctx, p.stream).Result()
}

// PoolStats 返回连接池状态。
func (p *StreamProducer) PoolStats() *redis.PoolStats {
	stats := p.client.PoolStats()
	return stats
}

// Ping 检查 Redis 连通性，供就绪探测使用。
func (p *StreamProducer) Ping(ctx context.Context) error {
	return p.client.Ping(ctx).Err()
}

// Close 关闭底层连接。
func (p *StreamProducer) Close() error {
	return p.client.Close()
}

// EncodeEvent 将统一事件序列化为 Stream 记录字段；与 ParseEvent 互逆。
func EncodeEvent(e store.RawEvent) (map[string]any, error) {
	extra := e.Extra
	if extra == nil {
		extra = map[string]string{}
	}
	raw, err := json.Marshal(extra)
	if err != nil {
		return nil, fmt.Errorf("queue: 序列化扩展字段失败: %w", err)
	}
	return map[string]any{
		"event_id":        e.EventID,
		"app_id":          e.AppID,
		"visitor_id":      e.VisitorID,
		"session_id":      e.SessionID,
		"url":             e.URL,
		"arg":             e.Arg,
		"version":         e.Version,
		"screen":          e.Screen,
		"lang":            e.Lang,
		"tz":              e.TZ,
		"domain":          e.Domain,
		"event_time_ms":   strconv.FormatInt(e.EventTime.UnixMilli(), 10),
		"received_at_ms":  strconv.FormatInt(e.ReceivedAt.UnixMilli(), 10),
		"ua":              e.UA,
		"ip_hash":         e.IPHash,
		"browser_name":    e.BrowserName,
		"browser_version": e.BrowserVer,
		"system":          e.System,
		"system_version":  e.SystemVer,
		"device_type":     e.DeviceType,
		"extra":           string(raw),
	}, nil
}

// ParseEvent 将 Stream 记录反序列化为统一事件；字段缺失或非法时返回错误。
func ParseEvent(values map[string]any) (store.RawEvent, error) {
	var e store.RawEvent
	str := func(key string) string {
		switch v := values[key].(type) {
		case string:
			return v
		case []byte:
			return string(v)
		default:
			return ""
		}
	}

	e.EventID = str("event_id")
	e.AppID = str("app_id")
	if e.EventID == "" {
		return store.RawEvent{}, fmt.Errorf("queue: Stream 记录缺少 event_id")
	}
	if e.AppID == "" {
		return store.RawEvent{}, fmt.Errorf("queue: Stream 记录缺少 app_id")
	}
	e.VisitorID = str("visitor_id")
	e.SessionID = str("session_id")
	e.URL = str("url")
	e.Arg = str("arg")
	e.Version = str("version")
	e.Screen = str("screen")
	e.Lang = str("lang")
	e.TZ = str("tz")
	e.Domain = str("domain")
	e.UA = str("ua")
	e.IPHash = str("ip_hash")
	e.BrowserName = str("browser_name")
	e.BrowserVer = str("browser_version")
	e.System = str("system")
	e.SystemVer = str("system_version")
	e.DeviceType = str("device_type")

	var err error
	if raw := str("event_time_ms"); raw != "" {
		if e.EventTime, err = parseMillis(raw); err != nil {
			return store.RawEvent{}, fmt.Errorf("queue: 非法 event_time_ms=%q: %w", raw, err)
		}
	}
	if raw := str("received_at_ms"); raw != "" {
		if e.ReceivedAt, err = parseMillis(raw); err != nil {
			return store.RawEvent{}, fmt.Errorf("queue: 非法 received_at_ms=%q: %w", raw, err)
		}
	}
	if raw := str("extra"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &e.Extra); err != nil {
			return store.RawEvent{}, fmt.Errorf("queue: 非法 extra JSON: %w", err)
		}
	}
	if e.Extra == nil {
		e.Extra = map[string]string{}
	}
	return e, nil
}

func parseMillis(raw string) (time.Time, error) {
	ms, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.UnixMilli(ms).UTC(), nil
}
