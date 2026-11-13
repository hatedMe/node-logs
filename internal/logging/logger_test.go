package logging

import (
	"context"
	"log/slog"
	"testing"
)

func TestNewDefaultsToInfo(t *testing.T) {
	logger, err := New("")
	if err != nil {
		t.Fatalf("New(\"\") 返回意外错误: %v", err)
	}
	if logger == nil {
		t.Fatal("New(\"\") 不应返回 nil logger")
	}
	if !logger.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("默认级别应启用 Info")
	}
	if logger.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("默认级别不应启用 Debug")
	}
}

func TestNewParsesLevels(t *testing.T) {
	cases := map[string]slog.Level{
		"debug": slog.LevelDebug,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
	}
	for name, want := range cases {
		t.Run(name, func(t *testing.T) {
			logger, err := New(name)
			if err != nil {
				t.Fatalf("New(%q) 返回意外错误: %v", name, err)
			}
			if !logger.Enabled(context.Background(), want) {
				t.Errorf("New(%q) 应启用 %v 级别", name, want)
			}
		})
	}
}

func TestNewInvalidLevel(t *testing.T) {
	if _, err := New("verbose-please"); err == nil {
		t.Fatal("New() 在非法级别时应返回错误，实际为 nil")
	}
}
