// Package worker 实现队列消费：批量读取、幂等落库、失败重试与死信。
package worker

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/hatedMe/node-logs/internal/aggregation"
	"github.com/hatedMe/node-logs/internal/store"
)

// Store 是批处理依赖的存储子集；*store.Tx 已实现全部方法。
type Store interface {
	aggregation.Store
	InsertRawEventsReturning(ctx context.Context, events []store.RawEvent) ([]string, error)
	UpdateRawSessionIDs(ctx context.Context, pairs map[string]string) error
	GetAppByID(ctx context.Context, appID string) (*store.App, error)
}

var _ Store = (*store.Tx)(nil)

// ProcessResult 是单批次处理结果。
type ProcessResult struct {
	Total    int
	Inserted int
	Dropped  int
	// Dupes 是批内重复 event_id 被折叠的数量（只保留首条）。
	Dupes int
}

// BatchProcessor 在单个可重试事务中批量写入原始事件与聚合。
type BatchProcessor struct {
	DB  *store.Store
	Agg *aggregation.Aggregator
}

// Process 处理一批事件：应用过滤 → UA 填充 → 幂等插入 → 聚合落库。
// 重复 event_id 的事件只参与计数零次；整个批次在一个事务中完成。
func (b *BatchProcessor) Process(ctx context.Context, events []store.RawEvent) (ProcessResult, error) {
	var res ProcessResult
	err := b.DB.WithTx(ctx, func(tx *store.Tx) error {
		r, err := b.processTx(ctx, tx, events)
		if err != nil {
			return err
		}
		res = r
		return nil
	})
	return res, err
}

func (b *BatchProcessor) processTx(ctx context.Context, db Store, events []store.RawEvent) (ProcessResult, error) {
	res := ProcessResult{Total: len(events)}
	if len(events) == 0 {
		return res, nil
	}
	// 批内按 event_id 去重（保留首条）：否则重复 ID 会同时命中插入返回集合，
	// 导致 PV 与明细重复计数。跨批次重复仍由 event_id 主键保证幂等。
	events = dedupeByEventID(events, &res)
	now := time.Now().UTC()

	kept := make([]store.RawEvent, 0, len(events))
	appStatus := map[string]string{}
	for _, e := range events {
		status, ok := appStatus[e.AppID]
		if !ok {
			app, err := db.GetAppByID(ctx, e.AppID)
			if err != nil {
				if errors.Is(err, store.ErrNotFound) {
					appStatus[e.AppID] = "unknown"
					res.Dropped++
					continue
				}
				return res, err
			}
			status = app.Status
			appStatus[e.AppID] = status
		}
		if status != "active" {
			res.Dropped++
			continue
		}
		if e.ReceivedAt.IsZero() {
			e.ReceivedAt = now
		}
		if e.EventTime.IsZero() {
			e.EventTime = e.ReceivedAt
		}
		info := aggregation.ParseUA(e.UA)
		e.BrowserName = info.Browser
		e.BrowserVer = info.BrowserVer
		e.System = info.System
		e.SystemVer = info.SystemVer
		e.DeviceType = info.Device
		kept = append(kept, e)
	}
	if len(kept) == 0 {
		return res, nil
	}
	// 预先按全局固定顺序锁定批内访客：与会话归因串行化多 worker 并发，
	// 固定顺序避免交叉死锁。锁为事务级，可重入，随事务结束释放。
	for _, vk := range distinctVisitorsSorted(kept) {
		if err := db.LockVisitor(ctx, vk.appID, vk.visitorID); err != nil {
			return res, err
		}
	}

	inserted, err := db.InsertRawEventsReturning(ctx, kept)
	if err != nil {
		return res, err
	}
	if len(inserted) == 0 {
		return res, nil
	}
	fresh := make([]store.RawEvent, 0, len(inserted))
	freshIDs := map[string]bool{}
	for _, id := range inserted {
		freshIDs[id] = true
	}
	for _, e := range kept {
		if freshIDs[e.EventID] {
			fresh = append(fresh, e)
		}
	}
	res.Inserted = len(fresh)
	mapping, err := b.Agg.ApplyWithSessions(ctx, db, fresh)
	if err != nil {
		return res, err
	}
	// 回填原始事件的服务端会话 ID，便于按会话追溯明细。
	if err := db.UpdateRawSessionIDs(ctx, mapping); err != nil {
		return res, err
	}
	return res, nil
}

// dedupeByEventID 按 event_id 折叠批内重复事件，只保留首条，并在 res 中记录折叠数。
func dedupeByEventID(events []store.RawEvent, res *ProcessResult) []store.RawEvent {
	seen := make(map[string]bool, len(events))
	unique := make([]store.RawEvent, 0, len(events))
	for _, e := range events {
		if seen[e.EventID] {
			res.Dupes++
			continue
		}
		seen[e.EventID] = true
		unique = append(unique, e)
	}
	return unique
}

type visitorKey struct {
	appID     string
	visitorID string
}

// distinctVisitorsSorted 返回批内去重并按全局固定顺序排列的访客键。
func distinctVisitorsSorted(events []store.RawEvent) []visitorKey {
	seen := map[visitorKey]bool{}
	var out []visitorKey
	for _, e := range events {
		k := visitorKey{appID: e.AppID, visitorID: e.VisitorID}
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].appID != out[j].appID {
			return out[i].appID < out[j].appID
		}
		return out[i].visitorID < out[j].visitorID
	})
	return out
}
