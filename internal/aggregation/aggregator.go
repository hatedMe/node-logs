package aggregation

import (
	"context"
	"sort"
	"time"

	"github.com/mileusna/useragent"

	"github.com/hatedMe/node-logs/internal/session"
	"github.com/hatedMe/node-logs/internal/store"
)

// Store 是聚合落库依赖的存储子集；*store.Tx 已实现全部方法。
type Store interface {
	session.Store
	InsertVisitorIfNew(ctx context.Context, key store.VisitorKey) (bool, error)
	InsertIPIfNew(ctx context.Context, key store.IPKey) (bool, error)
	InsertPageVisitorIfNew(ctx context.Context, key store.PageVisitorKey) (bool, error)
	InsertPageSessionIfNew(ctx context.Context, key store.PageSessionKey) (bool, error)
	UpsertDailyStats(ctx context.Context, delta store.StatsDelta) error
	UpsertPageStats(ctx context.Context, appID string, date time.Time, pages []store.PageDelta) error
	UpsertDeviceStats(ctx context.Context, appID string, date time.Time, devices []store.DeviceDelta) error
}

// Aggregator 按统计规则构建聚合增量并落库。
// 输入事件必须已按 event_id 去重（Worker 在调用前过滤），否则 PV 会重复计数。
type Aggregator struct {
	SessionTimeout time.Duration
}

type groupKey struct {
	appID string
	day   time.Time
}

func statDay(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}

// BuildDelta 是纯函数：按应用 + 自然日分组，计数 PV、页面 PV 与设备 PV。
// UV/IP/会话等需跨批次去重的指标由 Apply 补齐。
func (a *Aggregator) BuildDelta(events []store.RawEvent) []store.StatsDelta {
	groups := map[groupKey]*store.StatsDelta{}
	pageIdx := map[groupKey]map[string]int{}
	devIdx := map[groupKey]map[string]int{}
	var order []groupKey

	group := func(e store.RawEvent) *store.StatsDelta {
		k := groupKey{appID: e.AppID, day: statDay(e.EventTime)}
		g := groups[k]
		if g == nil {
			g = &store.StatsDelta{AppID: e.AppID, Date: statDay(e.EventTime)}
			groups[k] = g
			pageIdx[k] = map[string]int{}
			devIdx[k] = map[string]int{}
			order = append(order, k)
		}
		return g
	}

	for _, e := range events {
		g := group(e)
		g.PV++
		k := groupKey{appID: e.AppID, day: statDay(e.EventTime)}

		if i, ok := pageIdx[k][e.URL]; ok {
			g.Pages[i].PV++
		} else {
			pageIdx[k][e.URL] = len(g.Pages)
			g.Pages = append(g.Pages, store.PageDelta{URL: e.URL, PV: 1})
		}

		info := eventUA(e)
		for _, dd := range []store.DeviceDelta{
			{Dimension: DimensionBrowser, Value: info.Browser, PV: 1},
			{Dimension: DimensionSystem, Value: info.System, PV: 1},
			{Dimension: DimensionDevice, Value: info.Device, PV: 1},
		} {
			dk := dd.Dimension + "\x00" + dd.Value
			if i, ok := devIdx[k][dk]; ok {
				g.Devices[i].PV++
			} else {
				devIdx[k][dk] = len(g.Devices)
				g.Devices = append(g.Devices, dd)
			}
		}
	}

	deltas := make([]store.StatsDelta, 0, len(order))
	for _, k := range order {
		deltas = append(deltas, *groups[k])
	}
	return deltas
}

// Apply 对新事件做去重、会话归因并落库；调用方应保证事件已按时间可排序。
func (a *Aggregator) Apply(ctx context.Context, db Store, events []store.RawEvent) error {
	_, err := a.ApplyWithSessions(ctx, db, events)
	return err
}

// ApplyWithSessions 与 Apply 相同，并额外返回 event_id 到服务端会话 ID 的映射，
// 供调用方回填原始事件的 session_id。
func (a *Aggregator) ApplyWithSessions(ctx context.Context, db Store, events []store.RawEvent) (map[string]string, error) {
	mapping := map[string]string{}
	if len(events) == 0 {
		return mapping, nil
	}
	ordered := make([]store.RawEvent, len(events))
	copy(ordered, events)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].EventTime.Before(ordered[j].EventTime) })

	deltas := a.BuildDelta(ordered)
	// 注意：此处只存索引不存指针。ensure 可能触发 deltas 扩容重分配，
	// 届时旧指针全部失效，而索引始终有效。
	byGroup := map[groupKey]int{}
	for i := range deltas {
		d := &deltas[i]
		byGroup[groupKey{appID: d.AppID, day: statDay(d.Date)}] = i
	}
	pageOf := map[groupKey]map[string]int{}
	for i := range deltas {
		d := &deltas[i]
		k := groupKey{appID: d.AppID, day: statDay(d.Date)}
		m := map[string]int{}
		for j := range d.Pages {
			m[deltas[i].Pages[j].URL] = j
		}
		pageOf[k] = m
	}
	// 跳出回扣可能归因到批内无事件的日期（跨天会话），按需创建空增量。
	ensure := func(appID string, day time.Time) int {
		k := groupKey{appID: appID, day: day}
		if i, ok := byGroup[k]; ok {
			return i
		}
		deltas = append(deltas, store.StatsDelta{AppID: appID, Date: day})
		i := len(deltas) - 1
		byGroup[k] = i
		pageOf[k] = map[string]int{}
		return i
	}

	tracker := session.NewTracker(a.SessionTimeout)
	for _, e := range ordered {
		day := statDay(e.EventTime)
		gi := byGroup[groupKey{appID: e.AppID, day: day}]

		if isNew, err := db.InsertVisitorIfNew(ctx, store.VisitorKey{AppID: e.AppID, Date: e.EventTime, VisitorID: e.VisitorID}); err != nil {
			return nil, err
		} else if isNew {
			deltas[gi].UV++
		}
		if isNew, err := db.InsertIPIfNew(ctx, store.IPKey{AppID: e.AppID, Date: e.EventTime, IPHash: e.IPHash}); err != nil {
			return nil, err
		} else if isNew {
			deltas[gi].IPCount++
		}

		asg, err := tracker.Assign(ctx, db, e.AppID, e.VisitorID, e.EventTime, e.URL)
		if err != nil {
			return nil, err
		}
		mapping[e.EventID] = asg.SessionID
		deltas[gi].Sessions += asg.SessionsDelta
		bi := ensure(e.AppID, statDay(asg.BounceDate))
		deltas[bi].Bounces += asg.BounceDelta

		if pi, ok := pageOf[groupKey{appID: e.AppID, day: day}][e.URL]; ok {
			if isNew, err := db.InsertPageVisitorIfNew(ctx, store.PageVisitorKey{AppID: e.AppID, Date: e.EventTime, URL: e.URL, VisitorID: e.VisitorID}); err != nil {
				return nil, err
			} else if isNew {
				deltas[gi].Pages[pi].UV++
			}
			if isNew, err := db.InsertPageSessionIfNew(ctx, store.PageSessionKey{AppID: e.AppID, Date: e.EventTime, URL: e.URL, SessionID: asg.SessionID}); err != nil {
				return nil, err
			} else if isNew {
				deltas[gi].Pages[pi].Sessions++
			}
		}
	}

	for _, d := range deltas {
		if err := db.UpsertDailyStats(ctx, d); err != nil {
			return nil, err
		}
		if err := db.UpsertPageStats(ctx, d.AppID, d.Date, d.Pages); err != nil {
			return nil, err
		}
		if err := db.UpsertDeviceStats(ctx, d.AppID, d.Date, d.Devices); err != nil {
			return nil, err
		}
	}
	return mapping, nil
}

// UAInfo 是 UA 解析结果。
type UAInfo struct {
	Browser    string
	BrowserVer string
	System     string
	SystemVer  string
	Device     string
}

// ParseUA 解析 UA 字符串；空或无法识别时返回 unknown。
func ParseUA(raw string) UAInfo {
	if raw == "" {
		return UAInfo{Browser: "unknown", System: "unknown", Device: "unknown"}
	}
	ua := useragent.Parse(raw)
	info := UAInfo{
		Browser:    ua.Name,
		BrowserVer: ua.Version,
		System:     ua.OS,
		SystemVer:  ua.OSVersion,
		Device:     "desktop",
	}
	if info.Browser == "" {
		info.Browser = "unknown"
	}
	if info.System == "" {
		info.System = "unknown"
	}
	switch {
	case ua.Bot:
		info.Device = "bot"
	case ua.Mobile:
		info.Device = "mobile"
	case ua.Tablet:
		info.Device = "tablet"
	}
	return info
}

// eventUA 优先使用事件上已填充的解析结果，否则现场解析。
func eventUA(e store.RawEvent) UAInfo {
	if e.BrowserName != "" {
		return UAInfo{
			Browser:    e.BrowserName,
			BrowserVer: e.BrowserVer,
			System:     e.System,
			SystemVer:  e.SystemVer,
			Device:     e.DeviceType,
		}
	}
	return ParseUA(e.UA)
}
