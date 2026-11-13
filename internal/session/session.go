// Package session 实现会话归因：同一访客在超时窗口内的连续访问归为一个会话。
package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/hatedMe/node-logs/internal/store"
)

// Store 是会话持久化子集；*store.Store 与 *store.Tx 均已实现。
// 注意：LockVisitor 仅在事务内有效（*store.Tx），多 worker 并发安全依赖于此。
type Store interface {
	FindOpenSession(ctx context.Context, appID, visitorID string, since time.Time) (*store.Session, error)
	InsertSession(ctx context.Context, s store.Session) error
	ExtendSession(ctx context.Context, sessionID string, endedAt time.Time, exitURL string) (int64, error)
	// LockVisitor 锁定访客的会话创建临界区，串行化并发建会话。
	LockVisitor(ctx context.Context, appID, visitorID string) error
}

// Assignment 是单个事件的会话归因结果。
type Assignment struct {
	SessionID string
	IsNew     bool
	// SessionsDelta 计入事件所在日期；BounceDelta 计入 BounceDate（会话开始日期）。
	SessionsDelta int64
	BounceDelta   int64
	BounceDate    time.Time
}

// Tracker 跟踪批内未超时会话，将事件归因到新老会话。
type Tracker struct {
	Timeout time.Duration
	open    map[string]*openSession
}

type openSession struct {
	sessionID string
	startedAt time.Time
	endedAt   time.Time
}

// NewTracker 创建会话追踪器，timeout 内无事件即结束会话。
func NewTracker(timeout time.Duration) *Tracker {
	return &Tracker{Timeout: timeout, open: map[string]*openSession{}}
}

// Assign 将事件归因到会话，必要时创建或延长持久化会话。
// 调用方必须按事件时间递增顺序调用。
func (t *Tracker) Assign(ctx context.Context, db Store, appID, visitorID string, eventTime time.Time, url string) (Assignment, error) {
	key := appID + "\x00" + visitorID
	if o, ok := t.open[key]; ok && !eventTime.After(o.endedAt.Add(t.Timeout)) {
		return t.extend(ctx, db, o, eventTime, url)
	}
	// 慢路径先加访客锁：查找与创建之间的窗口若无锁，多 worker 会为同访客
	// 创建多个会话并重复统计。锁可重入，调用方预锁定后此处为无操作。
	if err := db.LockVisitor(ctx, appID, visitorID); err != nil {
		return Assignment{}, err
	}
	sess, err := db.FindOpenSession(ctx, appID, visitorID, eventTime.Add(-t.Timeout))
	if err != nil {
		return Assignment{}, err
	}
	if sess != nil {
		o := &openSession{sessionID: sess.SessionID, startedAt: sess.StartedAt, endedAt: sess.EndedAt}
		t.open[key] = o
		// 持久化会话保证 ended_at >= since，即窗口内，直接延长。
		return t.extend(ctx, db, o, eventTime, url)
	}

	id, err := generateSessionID()
	if err != nil {
		return Assignment{}, err
	}
	s := store.Session{
		SessionID: id, AppID: appID, VisitorID: visitorID,
		StartedAt: eventTime, EndedAt: eventTime, EventCount: 1,
		EntryURL: url, ExitURL: url,
	}
	if err := db.InsertSession(ctx, s); err != nil {
		return Assignment{}, err
	}
	t.open[key] = &openSession{sessionID: id, startedAt: eventTime, endedAt: eventTime}
	return Assignment{
		SessionID: id, IsNew: true,
		SessionsDelta: 1, BounceDelta: 1, BounceDate: eventTime,
	}, nil
}

func (t *Tracker) extend(ctx context.Context, db Store, o *openSession, eventTime time.Time, url string) (Assignment, error) {
	newCount, err := db.ExtendSession(ctx, o.sessionID, eventTime, url)
	if err != nil {
		return Assignment{}, err
	}
	o.endedAt = eventTime
	a := Assignment{SessionID: o.sessionID, BounceDate: o.startedAt}
	// 会话从单事件变为多事件时，扣减创建时计的跳出。
	if newCount == 2 {
		a.BounceDelta = -1
	}
	return a, nil
}

func generateSessionID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
