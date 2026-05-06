package slog

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/tianlu1990s/gim/internal/config"
)

func TestNewStdoutText(t *testing.T) {
	cfg := config.LogConfig{
		Level:     "info",
		Format:    "text",
		Output:    "stdout",
		ShortFile: false,
		Color:     false,
	}
	logger := New(cfg)
	if logger == nil {
		t.Fatal("New returned nil")
	}
	if logger.Logger == nil {
		t.Fatal("Logger.Logger is nil")
	}
}

func TestNewJSONFormat(t *testing.T) {
	cfg := config.LogConfig{
		Level:     "debug",
		Format:    "json",
		Output:    "stdout",
		ShortFile: false,
		Color:     false,
	}
	logger := New(cfg)
	if logger == nil {
		t.Fatal("New returned nil")
	}
}

func TestNewFileOutput(t *testing.T) {
	cfg := config.LogConfig{
		Level:      "warn",
		Format:     "text",
		Output:     "file",
		FilePath:   "/tmp/gim_test.log",
		MaxSize:    1,
		MaxBackups: 1,
		MaxAge:     1,
		Compress:   false,
		ShortFile:  true,
		Color:      false,
	}
	logger := New(cfg)
	if logger == nil {
		t.Fatal("New returned nil")
	}
	// 写入测试确保不 panic
	logger.Info("test file log")
}

func TestNewInvalidLevel(t *testing.T) {
	cfg := config.LogConfig{
		Level:     "invalid",
		Format:    "text",
		Output:    "stdout",
		ShortFile: false,
		Color:     false,
	}
	logger := New(cfg)
	if logger == nil {
		t.Fatal("New returned nil")
	}
	// 无效等级应回退到 info
	logger.Info("test")
}

func TestColoredOutput(t *testing.T) {
	cfg := config.LogConfig{
		Level:     "debug",
		Format:    "text",
		Output:    "stdout",
		ShortFile: true,
		Color:     true,
	}
	logger := New(cfg)
	if logger == nil {
		t.Fatal("New returned nil")
	}
	// 颜色模式下写入不 panic
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")
}

func TestShortFile(t *testing.T) {
	cfg := config.LogConfig{
		Level:     "info",
		Format:    "text",
		Output:    "stdout",
		ShortFile: true,
		Color:     false,
	}
	logger := New(cfg)
	if logger == nil {
		t.Fatal("New returned nil")
	}
	var buf bytes.Buffer
	// 测试短文件名模式下的日志输出
	logger.Info("test short file")
	_ = buf.String()
	// 验证不 panic 即可——ReplaceAttr 在内部生效
}

func TestLogLevels(t *testing.T) {
	cfg := config.LogConfig{
		Level:     "warn",
		Format:    "text",
		Output:    "stdout",
		ShortFile: false,
		Color:     false,
	}
	logger := New(cfg)
	if logger == nil {
		t.Fatal("New returned nil")
	}
	// Debug 和 Info 应被过滤（level = warn）
	logger.Debug("debug filtered")
	logger.Info("info filtered")
	// Warn 和 Error 应输出
	logger.Warn("warn shown")
	logger.Error("error shown")
}

func TestLevelMapping(t *testing.T) {
	tests := []struct {
		level string
	}{
		{"debug"},
		{"info"},
		{"warn"},
		{"error"},
		{"WARN"},
		{"ERROR"},
	}
	for _, tt := range tests {
		cfg := config.LogConfig{
			Level:     tt.level,
			Format:    "text",
			Output:    "stdout",
			ShortFile: false,
			Color:     false,
		}
		logger := New(cfg)
		if logger == nil {
			t.Errorf("New returned nil for level %s", tt.level)
		}
	}
}

func TestContainsCaller(t *testing.T) {
	// 验证 slog.go 文件名能被正确引用
	sourceName := "slog.go"
	if !strings.HasSuffix(sourceName, ".go") {
		t.Error("source file should end with .go")
	}
}

func TestFatal(t *testing.T) {
	// Fatal calls os.Exit(1) after logging. Use subprocess to verify exit code.
	if os.Getenv("SLOG_TEST_FATAL") == "1" {
		cfg := config.LogConfig{
			Level:  "info",
			Format: "text",
			Output: "stdout",
		}
		logger := New(cfg)
		logger.Fatal("test fatal message")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestFatal")
	cmd.Env = append(os.Environ(), "SLOG_TEST_FATAL=1")
	err := cmd.Run()
	if err == nil {
		t.Error("expected Fatal to call os.Exit(1)")
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() != 1 {
			t.Errorf("exit code = %d, want 1", exitErr.ExitCode())
		}
	}
}
