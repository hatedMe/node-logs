package testdb

import (
	"context"
	"time"

	"github.com/hatedMe/node-logs/internal/aggregation"
	"github.com/hatedMe/node-logs/internal/store"
)

// FakeStore 是内存存储实现，供 worker 等包做无 DB 单测。
// 它实现 worker.Store（aggregation.Store + 原始事件 + 应用查询）。
type FakeStore struct {
	Apps map[string]store.App
	Raw  map[string]store.RawEvent

	visitors     map[store.VisitorKey]bool
	ips          map[store.IPKey]bool
	pageVisitors map[store.PageVisitorKey]bool
	pageSessions map[store.PageSessionKey]bool
	sessions     map[string]*store.Session

	Daily   map[DayKey]*store.StatsDelta
	Pages   map[PageKey]*store.PageDelta
	Devices map[DeviceKey]int64
	// Locks 记录加锁顺序，供并发顺序断言。
	Locks []string
}

// DayKey 标识日汇总行。
type DayKey struct {
	AppID string
	Day   time.Time
}

// PageKey 标识页面汇总行。
type PageKey struct {
	AppID string
	Day   time.Time
	URL   string
}

// DeviceKey 标识设备汇总行。
type DeviceKey struct {
	AppID     string
	Day       time.Time
	Dimension string
	Value     string
}

// NewFakeStore 创建空内存存储。
func NewFakeStore() *FakeStore {
	return &FakeStore{
		Apps: map[string]store.App{}, Raw: map[string]store.RawEvent{},
		visitors: map[store.VisitorKey]bool{}, ips: map[store.IPKey]bool{},
		pageVisitors: map[store.PageVisitorKey]bool{}, pageSessions: map[store.PageSessionKey]bool{},
		sessions: map[string]*store.Session{},
		Daily:    map[DayKey]*store.StatsDelta{},
		Pages:    map[PageKey]*store.PageDelta{},
		Devices:  map[DeviceKey]int64{},
	}
}

// 确保 FakeStore 满足聚合与会话存储接口。
var (
	_ aggregation.Store = (*FakeStore)(nil)
)

func fakeDay(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}

// GetAppByID 查询应用；不存在时返回 store.ErrNotFound。
func (f *FakeStore) GetAppByID(_ context.Context, appID string) (*store.App, error) {
	app, ok := f.Apps[appID]
	if !ok {
		return nil, store.ErrNotFound
	}
	cp := app
	return &cp, nil
}

// InsertRawEventsReturning 写入新事件并返回新插入的 event_id。
func (f *FakeStore) InsertRawEventsReturning(_ context.Context, events []store.RawEvent) ([]string, error) {
	var inserted []string
	for _, e := range events {
		if _, ok := f.Raw[e.EventID]; ok {
			continue
		}
		f.Raw[e.EventID] = e
		inserted = append(inserted, e.EventID)
	}
	return inserted, nil
}

// UpdateRawSessionIDs 回填内存事件的会话 ID。
func (f *FakeStore) UpdateRawSessionIDs(_ context.Context, pairs map[string]string) error {
	for id, sid := range pairs {
		if e, ok := f.Raw[id]; ok {
			e.SessionID = sid
			f.Raw[id] = e
		}
	}
	return nil
}

// InsertVisitorIfNew 插入 UV 去重键。
func (f *FakeStore) InsertVisitorIfNew(_ context.Context, key store.VisitorKey) (bool, error) {
	key.Date = fakeDay(key.Date)
	if f.visitors[key] {
		return false, nil
	}
	f.visitors[key] = true
	return true, nil
}

// InsertIPIfNew 插入独立 IP 去重键。
func (f *FakeStore) InsertIPIfNew(_ context.Context, key store.IPKey) (bool, error) {
	key.Date = fakeDay(key.Date)
	if f.ips[key] {
		return false, nil
	}
	f.ips[key] = true
	return true, nil
}

// InsertPageVisitorIfNew 插入页面 UV 去重键。
func (f *FakeStore) InsertPageVisitorIfNew(_ context.Context, key store.PageVisitorKey) (bool, error) {
	key.Date = fakeDay(key.Date)
	if f.pageVisitors[key] {
		return false, nil
	}
	f.pageVisitors[key] = true
	return true, nil
}

// InsertPageSessionIfNew 插入页面会话去重键。
func (f *FakeStore) InsertPageSessionIfNew(_ context.Context, key store.PageSessionKey) (bool, error) {
	key.Date = fakeDay(key.Date)
	if f.pageSessions[key] {
		return false, nil
	}
	f.pageSessions[key] = true
	return true, nil
}

// LockVisitor 记录访客锁（内存实现无需真锁）。
func (f *FakeStore) LockVisitor(_ context.Context, appID, visitorID string) error {
	f.Locks = append(f.Locks, appID+"\x00"+visitorID)
	return nil
}

// FindOpenSession 查找未超时会话。
func (f *FakeStore) FindOpenSession(_ context.Context, appID, visitorID string, since time.Time) (*store.Session, error) {
	for _, s := range f.sessions {
		if s.AppID == appID && s.VisitorID == visitorID && !s.EndedAt.Before(since) {
			cp := *s
			return &cp, nil
		}
	}
	return nil, nil
}

// InsertSession 创建会话。
func (f *FakeStore) InsertSession(_ context.Context, s store.Session) error {
	cp := s
	f.sessions[s.SessionID] = &cp
	return nil
}

// ExtendSession 延长会话并返回延长后的事件数。
func (f *FakeStore) ExtendSession(_ context.Context, sessionID string, endedAt time.Time, exitURL string) (int64, error) {
	s := f.sessions[sessionID]
	s.EndedAt = endedAt
	s.ExitURL = exitURL
	s.EventCount++
	return s.EventCount, nil
}

// UpsertDailyStats 累加日汇总。
func (f *FakeStore) UpsertDailyStats(_ context.Context, delta store.StatsDelta) error {
	k := DayKey{AppID: delta.AppID, Day: fakeDay(delta.Date)}
	d := f.Daily[k]
	if d == nil {
		d = &store.StatsDelta{AppID: delta.AppID, Date: fakeDay(delta.Date)}
		f.Daily[k] = d
	}
	d.PV += delta.PV
	d.UV += delta.UV
	d.IPCount += delta.IPCount
	d.Sessions += delta.Sessions
	d.Bounces += delta.Bounces
	if d.Bounces < 0 {
		d.Bounces = 0
	}
	return nil
}

// UpsertPageStats 累加页面汇总。
func (f *FakeStore) UpsertPageStats(_ context.Context, appID string, date time.Time, pages []store.PageDelta) error {
	for _, p := range pages {
		k := PageKey{AppID: appID, Day: fakeDay(date), URL: p.URL}
		d := f.Pages[k]
		if d == nil {
			d = &store.PageDelta{URL: p.URL}
			f.Pages[k] = d
		}
		d.PV += p.PV
		d.UV += p.UV
		d.Sessions += p.Sessions
	}
	return nil
}

// UpsertDeviceStats 累加设备汇总。
func (f *FakeStore) UpsertDeviceStats(_ context.Context, appID string, date time.Time, devices []store.DeviceDelta) error {
	for _, dd := range devices {
		k := DeviceKey{AppID: appID, Day: fakeDay(date), Dimension: dd.Dimension, Value: dd.Value}
		f.Devices[k] += dd.PV
	}
	return nil
}
