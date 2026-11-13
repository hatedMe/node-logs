// Package store 封装 PostgreSQL 访问：迁移、原始事件与聚合读写。
package store

import "time"

// RawEvent 是采集流水线统一的事件结构，对应 raw_events 表。
type RawEvent struct {
	EventID   string
	AppID     string
	VisitorID string
	SessionID string
	URL       string
	Arg       string
	Version   string
	Screen    string
	Lang      string
	TZ        string
	Domain    string

	EventTime  time.Time
	ReceivedAt time.Time

	UA          string
	IPHash      string
	BrowserName string
	BrowserVer  string
	System      string
	SystemVer   string
	DeviceType  string

	Extra map[string]string
}

// VisitorKey 标识"应用 + 自然日 + 访客"，用于 UV 去重。
type VisitorKey struct {
	AppID     string
	Date      time.Time
	VisitorID string
}

// IPKey 标识"应用 + 自然日 + IP 哈希"，用于独立 IP 去重。
type IPKey struct {
	AppID  string
	Date   time.Time
	IPHash string
}

// PageVisitorKey 标识"应用 + 自然日 + 页面 + 访客"，用于页面 UV 去重。
type PageVisitorKey struct {
	AppID     string
	Date      time.Time
	URL       string
	VisitorID string
}

// PageSessionKey 标识"应用 + 自然日 + 页面 + 会话"，用于页面会话去重。
type PageSessionKey struct {
	AppID     string
	Date      time.Time
	URL       string
	SessionID string
}

// StatsDelta 是单个批次产生的聚合增量，由 Aggregator 构建、Worker 落库。
type StatsDelta struct {
	AppID string
	Date  time.Time

	PV       int64
	UV       int64
	IPCount  int64
	Sessions int64
	Bounces  int64

	Pages   []PageDelta
	Devices []DeviceDelta
}

// PageDelta 是单个页面的聚合增量。
type PageDelta struct {
	URL      string
	PV       int64
	UV       int64
	Sessions int64
}

// DeviceDelta 是单个设备维度值的聚合增量。
// Dimension 取值：browser、system、device。
type DeviceDelta struct {
	Dimension string
	Value     string
	PV        int64
}

// App 对应 apps 表。
type App struct {
	AppID     string
	SecretKey string
	Name      string
	Category  string
	Domain    string
	Username  string
	Status    string
	CreatedAt time.Time
}

// User 对应 users 表。
type User struct {
	ID           int64
	Username     string
	PasswordHash string
	CreatedAt    time.Time
}

// Session 对应 sessions 表。
type Session struct {
	SessionID  string
	AppID      string
	VisitorID  string
	StartedAt  time.Time
	EndedAt    time.Time
	EventCount int64
	EntryURL   string
	ExitURL    string
}
