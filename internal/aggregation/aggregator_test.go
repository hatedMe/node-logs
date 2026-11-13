package aggregation

import (
	"context"
	"testing"
	"time"

	"github.com/hatedMe/node-logs/internal/store"
)

const chromeUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
const iphoneUA = "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1"

func testEvent(appID, eventID, visitorID, url string, at time.Time) store.RawEvent {
	return store.RawEvent{
		EventID: eventID, AppID: appID, VisitorID: visitorID, URL: url,
		EventTime: at, ReceivedAt: at, UA: chromeUA, IPHash: "ip-" + visitorID,
	}
}

// fakeStore 是内存去重实现，用于无 DB 单测。
type fakeStore struct {
	visitors     map[store.VisitorKey]bool
	ips          map[store.IPKey]bool
	pageVisitors map[store.PageVisitorKey]bool
	pageSessions map[store.PageSessionKey]bool
	sessions     map[string]*store.Session
	daily        []store.StatsDelta
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		visitors: map[store.VisitorKey]bool{}, ips: map[store.IPKey]bool{},
		pageVisitors: map[store.PageVisitorKey]bool{}, pageSessions: map[store.PageSessionKey]bool{},
		sessions: map[string]*store.Session{},
	}
}

func dateKey(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}

func (f *fakeStore) InsertVisitorIfNew(_ context.Context, key store.VisitorKey) (bool, error) {
	key.Date = dateKey(key.Date)
	if f.visitors[key] {
		return false, nil
	}
	f.visitors[key] = true
	return true, nil
}

func (f *fakeStore) InsertIPIfNew(_ context.Context, key store.IPKey) (bool, error) {
	key.Date = dateKey(key.Date)
	if f.ips[key] {
		return false, nil
	}
	f.ips[key] = true
	return true, nil
}

func (f *fakeStore) InsertPageVisitorIfNew(_ context.Context, key store.PageVisitorKey) (bool, error) {
	key.Date = dateKey(key.Date)
	if f.pageVisitors[key] {
		return false, nil
	}
	f.pageVisitors[key] = true
	return true, nil
}

func (f *fakeStore) InsertPageSessionIfNew(_ context.Context, key store.PageSessionKey) (bool, error) {
	key.Date = dateKey(key.Date)
	if f.pageSessions[key] {
		return false, nil
	}
	f.pageSessions[key] = true
	return true, nil
}

func (f *fakeStore) LockVisitor(_ context.Context, _, _ string) error {
	return nil
}

func (f *fakeStore) FindOpenSession(_ context.Context, appID, visitorID string, since time.Time) (*store.Session, error) {
	for _, s := range f.sessions {
		if s.AppID == appID && s.VisitorID == visitorID && !s.EndedAt.Before(since) {
			cp := *s
			return &cp, nil
		}
	}
	return nil, nil
}

func (f *fakeStore) InsertSession(_ context.Context, s store.Session) error {
	cp := s
	f.sessions[s.SessionID] = &cp
	return nil
}

func (f *fakeStore) ExtendSession(_ context.Context, sessionID string, endedAt time.Time, exitURL string) (int64, error) {
	s := f.sessions[sessionID]
	s.EndedAt = endedAt
	s.ExitURL = exitURL
	s.EventCount++
	return s.EventCount, nil
}

func (f *fakeStore) UpsertDailyStats(_ context.Context, delta store.StatsDelta) error {
	f.daily = append(f.daily, delta)
	return nil
}

func (f *fakeStore) UpsertPageStats(_ context.Context, _ string, _ time.Time, _ []store.PageDelta) error {
	return nil
}

func (f *fakeStore) UpsertDeviceStats(_ context.Context, _ string, _ time.Time, _ []store.DeviceDelta) error {
	return nil
}

func findDelta(deltas []store.StatsDelta, appID string, date time.Time) *store.StatsDelta {
	for i := range deltas {
		if deltas[i].AppID == appID && dateKey(deltas[i].Date).Equal(dateKey(date)) {
			return &deltas[i]
		}
	}
	return nil
}

func TestBuildDeltaGroupsByAppAndDate(t *testing.T) {
	a := &Aggregator{SessionTimeout: 30 * time.Minute}
	day1 := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC)
	events := []store.RawEvent{
		testEvent("800800", "e1", "v1", "/a", day1),
		testEvent("800800", "e2", "v1", "/b", day1),
		testEvent("800800", "e3", "v1", "/a", day2),
		testEvent("800801", "e4", "v1", "/a", day1),
	}

	deltas := a.BuildDelta(events)

	if len(deltas) != 3 {
		t.Fatalf("应按应用+日期分组为 3 组，实际 %d 组", len(deltas))
	}
	d := findDelta(deltas, "800800", day1)
	if d == nil || d.PV != 2 {
		t.Errorf("800800/day1 PV 应为 2: %+v", d)
	}
	if len(d.Pages) != 2 {
		t.Fatalf("800800/day1 应有 2 个页面分组，实际 %+v", d.Pages)
	}
	d2 := findDelta(deltas, "800800", day2)
	if d2 == nil || d2.PV != 1 {
		t.Errorf("800800/day2 PV 应为 1: %+v", d2)
	}
	d3 := findDelta(deltas, "800801", day1)
	if d3 == nil || d3.PV != 1 {
		t.Errorf("800801/day1 PV 应为 1: %+v", d3)
	}
}

func TestBuildDeltaDevices(t *testing.T) {
	a := &Aggregator{SessionTimeout: 30 * time.Minute}
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	iphone := testEvent("800800", "e2", "v2", "/a", at)
	iphone.UA = iphoneUA
	deltas := a.BuildDelta([]store.RawEvent{testEvent("800800", "e1", "v1", "/a", at), iphone})

	if len(deltas) != 1 {
		t.Fatalf("应为 1 组，实际 %d 组", len(deltas))
	}
	devices := map[string]map[string]int64{}
	for _, dd := range deltas[0].Devices {
		if devices[dd.Dimension] == nil {
			devices[dd.Dimension] = map[string]int64{}
		}
		devices[dd.Dimension][dd.Value] = dd.PV
	}
	if devices[DimensionBrowser]["Chrome"] != 1 {
		t.Errorf("browser Chrome 应为 1: %v", devices)
	}
	if devices[DimensionSystem]["Windows"] != 1 {
		t.Errorf("system Windows 应为 1: %v", devices)
	}
	if devices[DimensionDevice]["desktop"] != 1 || devices[DimensionDevice]["mobile"] != 1 {
		t.Errorf("device 维度错误: %v", devices)
	}
}

func TestParseUA(t *testing.T) {
	info := ParseUA(chromeUA)
	if info.Browser != "Chrome" || info.System != "Windows" || info.Device != "desktop" {
		t.Errorf("Chrome UA 解析错误: %+v", info)
	}
	if info.BrowserVer == "" {
		t.Errorf("Chrome UA 应解析出浏览器版本: %+v", info)
	}
	if ParseUA(iphoneUA).Device != "mobile" {
		t.Errorf("iPhone UA 设备应为 mobile: %+v", ParseUA(iphoneUA))
	}
	empty := ParseUA("")
	if empty.Browser != "unknown" || empty.System != "unknown" || empty.Device != "unknown" {
		t.Errorf("空 UA 应为 unknown: %+v", empty)
	}
}

func TestApplyDedupsAcrossCalls(t *testing.T) {
	a := &Aggregator{SessionTimeout: 30 * time.Minute}
	db := newFakeStore()
	ctx := context.Background()
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)

	first := []store.RawEvent{
		testEvent("800800", "e1", "v1", "/a", at),
		testEvent("800800", "e2", "v1", "/b", at.Add(time.Minute)),
	}
	if err := a.Apply(ctx, db, first); err != nil {
		t.Fatalf("Apply 失败: %v", err)
	}
	// 同访客新事件（不同 event_id）：PV 增加，UV 不增加。
	second := []store.RawEvent{testEvent("800800", "e3", "v1", "/c", at.Add(2*time.Minute))}
	if err := a.Apply(ctx, db, second); err != nil {
		t.Fatalf("Apply 失败: %v", err)
	}

	if len(db.daily) != 2 {
		t.Fatalf("应落库 2 条日增量，实际 %d 条", len(db.daily))
	}
	if db.daily[0].PV != 2 || db.daily[0].UV != 1 {
		t.Errorf("首批应 PV=2 UV=1: %+v", db.daily[0])
	}
	if db.daily[1].PV != 1 || db.daily[1].UV != 0 {
		t.Errorf("次批应 PV=1 UV=0: %+v", db.daily[1])
	}
}

func TestApplySameIPMultipleVisitors(t *testing.T) {
	a := &Aggregator{SessionTimeout: 30 * time.Minute}
	db := newFakeStore()
	ctx := context.Background()
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)

	e1 := testEvent("800800", "e1", "v1", "/a", at)
	e2 := testEvent("800800", "e2", "v2", "/a", at)
	e2.IPHash = e1.IPHash // 同 IP 不同访客
	if err := a.Apply(ctx, db, []store.RawEvent{e1, e2}); err != nil {
		t.Fatalf("Apply 失败: %v", err)
	}
	if len(db.daily) != 1 {
		t.Fatalf("应落库 1 条日增量，实际 %d 条", len(db.daily))
	}
	if db.daily[0].UV != 2 || db.daily[0].IPCount != 1 {
		t.Errorf("同 IP 多访客应 UV=2 IP=1: %+v", db.daily[0])
	}
}

func TestApplySessionBounce(t *testing.T) {
	a := &Aggregator{SessionTimeout: 30 * time.Minute}
	db := newFakeStore()
	ctx := context.Background()
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)

	if err := a.Apply(ctx, db, []store.RawEvent{testEvent("800800", "e1", "v1", "/a", at)}); err != nil {
		t.Fatalf("Apply 失败: %v", err)
	}
	if db.daily[0].Sessions != 1 || db.daily[0].Bounces != 1 {
		t.Fatalf("单事件会话应 sessions=1 bounces=1: %+v", db.daily[0])
	}
	if err := a.Apply(ctx, db, []store.RawEvent{testEvent("800800", "e2", "v1", "/b", at.Add(5*time.Minute))}); err != nil {
		t.Fatalf("Apply 失败: %v", err)
	}
	if db.daily[1].Sessions != 0 || db.daily[1].Bounces != -1 {
		t.Fatalf("第二事件应 sessions=0 bounces=-1: %+v", db.daily[1])
	}
}

func TestApplyCrossMidnight(t *testing.T) {
	a := &Aggregator{SessionTimeout: 30 * time.Minute}
	db := newFakeStore()
	ctx := context.Background()

	e1 := testEvent("800800", "e1", "v1", "/a", time.Date(2026, 11, 12, 23, 59, 0, 0, time.UTC))
	e2 := testEvent("800800", "e2", "v1", "/b", time.Date(2026, 11, 13, 0, 1, 0, 0, time.UTC))
	if err := a.Apply(ctx, db, []store.RawEvent{e1, e2}); err != nil {
		t.Fatalf("Apply 失败: %v", err)
	}
	day1 := findDelta(db.daily, "800800", e1.EventTime)
	day2 := findDelta(db.daily, "800800", e2.EventTime)
	if day1 == nil || day2 == nil {
		t.Fatalf("跨天事件应落到两天: %+v", db.daily)
	}
	// 会话计在开始日；第二事件的跳出回扣同样归因到开始日。
	if day1.Sessions != 1 || day1.Bounces != 0 {
		t.Errorf("开始日应 sessions=1 bounces=0: %+v", day1)
	}
	if day2.Sessions != 0 || day2.Bounces != 0 {
		t.Errorf("次日应 sessions=0 bounces=0: %+v", day2)
	}
	if day1.UV != 1 || day2.UV != 1 {
		t.Errorf("跨天 UV 应分别计数: %+v %+v", day1, day2)
	}
}

func TestApplyPageUV(t *testing.T) {
	a := &Aggregator{SessionTimeout: 30 * time.Minute}
	db := newFakeStore()
	ctx := context.Background()
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)

	events := []store.RawEvent{
		testEvent("800800", "e1", "v1", "/a", at),
		testEvent("800800", "e2", "v1", "/a", at.Add(time.Minute)),
		testEvent("800800", "e3", "v1", "/b", at.Add(2*time.Minute)),
	}
	if err := a.Apply(ctx, db, events); err != nil {
		t.Fatalf("Apply 失败: %v", err)
	}
	if len(db.daily) != 1 {
		t.Fatalf("应落库 1 条日增量，实际 %d 条", len(db.daily))
	}
	pages := map[string]store.PageDelta{}
	for _, p := range db.daily[0].Pages {
		pages[p.URL] = p
	}
	if pages["/a"].PV != 2 || pages["/a"].UV != 1 {
		t.Errorf("/a 应 PV=2 UV=1: %+v", pages["/a"])
	}
	if pages["/b"].PV != 1 || pages["/b"].UV != 1 {
		t.Errorf("/b 应 PV=1 UV=1: %+v", pages["/b"])
	}
	if pages["/a"].Sessions != 1 || pages["/b"].Sessions != 1 {
		t.Errorf("同会话页面 sessions 应各为 1: %+v %+v", pages["/a"], pages["/b"])
	}
}

func TestApplyBounceToDatelessGroup(t *testing.T) {
	a := &Aggregator{SessionTimeout: 30 * time.Minute}
	db := newFakeStore()
	ctx := context.Background()
	// 持久化单事件会话开始于 day1 23:59；本批只有 day2 事件。
	start := time.Date(2026, 11, 12, 23, 59, 0, 0, time.UTC)
	db.sessions["s1"] = &store.Session{
		SessionID: "s1", AppID: "800800", VisitorID: "v1",
		StartedAt: start, EndedAt: start, EventCount: 1, EntryURL: "/a", ExitURL: "/a",
	}
	e := testEvent("800800", "e2", "v1", "/b", time.Date(2026, 11, 13, 0, 1, 0, 0, time.UTC))
	e3 := testEvent("800800", "e3", "v2", "/c", time.Date(2026, 11, 13, 0, 2, 0, 0, time.UTC))
	if err := a.Apply(ctx, db, []store.RawEvent{e, e3}); err != nil {
		t.Fatalf("Apply 失败: %v", err)
	}
	day1 := findDelta(db.daily, "800800", start)
	if day1 == nil || day1.Bounces != -1 || day1.Sessions != 0 {
		t.Fatalf("跳出回扣应归因到开始日且不新增会话: %+v", day1)
	}
	day2 := findDelta(db.daily, "800800", e.EventTime)
	if day2 == nil || day2.PV != 2 || day2.UV != 2 {
		t.Fatalf("次日增量错误: %+v", day2)
	}
	if len(day2.Pages) != 2 || day2.Pages[0].UV != 1 || day2.Pages[1].UV != 1 {
		t.Fatalf("次日页面增量错误: %+v", day2.Pages)
	}
}

func TestApplyWithSessionsReturnsMapping(t *testing.T) {
	a := &Aggregator{SessionTimeout: 30 * time.Minute}
	db := newFakeStore()
	ctx := context.Background()
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)

	events := []store.RawEvent{
		testEvent("800800", "e1", "v1", "/a", at),
		testEvent("800800", "e2", "v1", "/b", at.Add(time.Minute)),
		testEvent("800800", "e3", "v2", "/a", at),
	}
	mapping, err := a.ApplyWithSessions(ctx, db, events)
	if err != nil {
		t.Fatalf("ApplyWithSessions 失败: %v", err)
	}
	if len(mapping) != 3 {
		t.Fatalf("应返回 3 条事件会话映射，实际 %d 条", len(mapping))
	}
	if mapping["e1"] == "" || mapping["e1"] != mapping["e2"] {
		t.Errorf("同访客事件应映射到同一会话: %v", mapping)
	}
	if mapping["e3"] == "" || mapping["e3"] == mapping["e1"] {
		t.Errorf("不同访客应映射到不同会话: %v", mapping)
	}
}

func TestApplyEmpty(t *testing.T) {
	a := &Aggregator{SessionTimeout: 30 * time.Minute}
	if err := a.Apply(context.Background(), newFakeStore(), nil); err != nil {
		t.Errorf("空批次应直接成功: %v", err)
	}
}
