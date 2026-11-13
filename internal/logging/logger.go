// Package logging 初始化结构化日志。
package logging

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// New 按级别名称创建 JSON 结构化日志器；空字符串默认为 info。
func New(level string) (*slog.Logger, error) {
	lv := slog.LevelInfo
	if strings.TrimSpace(level) != "" {
		if err := lv.UnmarshalText([]byte(strings.ToLower(strings.TrimSpace(level)))); err != nil {
			return nil, fmt.Errorf("logging: 非法日志级别 %q: %w", level, err)
		}
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lv})), nil
}
