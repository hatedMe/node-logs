package collector

import (
	"strings"
	"testing"
	"time"

	"github.com/hatedMe/node-logs/internal/store"
)

func validEvent() store.RawEvent {
	return store.RawEvent{
		EventID:    "evt-1",
		AppID:      "800800",
		VisitorID:  "v1",
		URL:        "/home",
		EventTime:  time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC),
		ReceivedAt: time.Date(2026, 11, 12, 10, 0, 1, 0, time.UTC),
		IPHash:     strings.Repeat("a", 64),
	}
}

func TestValidateOK(t *testing.T) {
	v := Validator{MaxURLLength: 2048}
	if err := v.Validate(validEvent()); err != nil {
		t.Errorf("合法事件应通过校验，实际错误: %v", err)
	}
}

func TestValidateMissingFields(t *testing.T) {
	v := Validator{MaxURLLength: 2048}
	cases := map[string]func(*store.RawEvent){
		"缺少 event_id":   func(e *store.RawEvent) { e.EventID = "" },
		"缺少 app_id":     func(e *store.RawEvent) { e.AppID = "" },
		"缺少 visitor_id": func(e *store.RawEvent) { e.VisitorID = "" },
		"缺少 event_time": func(e *store.RawEvent) { e.EventTime = time.Time{} },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			e := validEvent()
			mutate(&e)
			if err := v.Validate(e); err == nil {
				t.Errorf("%s 时应返回错误，实际为 nil", name)
			}
		})
	}
}

func TestValidateURLTooLong(t *testing.T) {
	v := Validator{MaxURLLength: 2048}
	e := validEvent()
	e.URL = strings.Repeat("a", 2049)
	err := v.Validate(e)
	if err == nil {
		t.Fatal("超长 URL 应返回错误，实际为 nil")
	}
	if !strings.Contains(err.Error(), "url") {
		t.Errorf("错误信息应提及 url，实际为 %q", err.Error())
	}
}

func TestValidateFieldCaps(t *testing.T) {
	v := Validator{MaxURLLength: 2048}
	cases := map[string]func(*store.RawEvent){
		"arg 超长":        func(e *store.RawEvent) { e.Arg = strings.Repeat("a", 4097) },
		"version 超长":    func(e *store.RawEvent) { e.Version = strings.Repeat("a", 257) },
		"event_id 超长":   func(e *store.RawEvent) { e.EventID = strings.Repeat("a", 129) },
		"app_id 超长":     func(e *store.RawEvent) { e.AppID = strings.Repeat("a", 129) },
		"visitor_id 超长": func(e *store.RawEvent) { e.VisitorID = strings.Repeat("a", 129) },
		"session_id 超长": func(e *store.RawEvent) { e.SessionID = strings.Repeat("a", 129) },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			e := validEvent()
			mutate(&e)
			if err := v.Validate(e); err == nil {
				t.Errorf("%s 时应返回错误，实际为 nil", name)
			}
		})
	}
}
