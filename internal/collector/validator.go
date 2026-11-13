package collector

import (
	"fmt"

	"github.com/hatedMe/node-logs/internal/store"
)

// 字段长度上限（字符数）。
const (
	maxArgLength     = 4096
	maxIDLength      = 128
	maxShortTextLong = 256
)

// Validator 校验归一化后的事件。不做归一化，只做拒绝判定。
type Validator struct {
	MaxURLLength int
}

func (v Validator) maxURL() int {
	if v.MaxURLLength <= 0 {
		return 2048
	}
	return v.MaxURLLength
}

// Validate 校验必填项与字段长度，非法时返回明确错误。
func (v Validator) Validate(e store.RawEvent) error {
	if e.EventID == "" {
		return fmt.Errorf("缺少 event_id")
	}
	if e.AppID == "" {
		return fmt.Errorf("缺少 app_id（pvid）")
	}
	if e.VisitorID == "" {
		return fmt.Errorf("缺少 visitor_id")
	}
	if e.EventTime.IsZero() {
		return fmt.Errorf("缺少 event_time")
	}
	for _, c := range []struct {
		name  string
		value string
		max   int
	}{
		{"url", e.URL, v.maxURL()},
		{"arg", e.Arg, maxArgLength},
		{"event_id", e.EventID, maxIDLength},
		{"app_id", e.AppID, maxIDLength},
		{"visitor_id", e.VisitorID, maxIDLength},
		{"session_id", e.SessionID, maxIDLength},
		{"v", e.Version, maxShortTextLong},
		{"scr", e.Screen, maxShortTextLong},
		{"lg", e.Lang, maxShortTextLong},
		{"tz", e.TZ, maxShortTextLong},
		{"dm", e.Domain, maxShortTextLong},
	} {
		if len([]rune(c.value)) > c.max {
			return fmt.Errorf("参数 %s 超长（上限 %d 字符）", c.name, c.max)
		}
	}
	return nil
}
