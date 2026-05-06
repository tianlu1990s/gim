package slog

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/tianlu1990s/gim/internal/config"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger 封装 Go 标准库 log/slog，提供结构化日志 + 彩色输出 + 日志轮转。
// 选择 slog 而非 Zap：Go 1.21+ 原生支持，零额外依赖，API 简洁够用。
type Logger struct {
	*slog.Logger
}

// New 根据配置创建日志实例。
// output=file 时自动创建日志目录，使用 lumberjack 按大小/天数轮转和压缩。
func New(cfg config.LogConfig) *Logger {
	var writer io.Writer

	if cfg.Output == "file" {
		// 自动创建日志目录，避免目录不存在导致 panic
		dir := filepath.Dir(cfg.FilePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			panic("Failed to create log directory: " + err.Error())
		}

		writer = &lumberjack.Logger{
			Filename:   cfg.FilePath,
			MaxSize:    cfg.MaxSize,    // MB
			MaxBackups: cfg.MaxBackups, // 保留旧文件数
			MaxAge:     cfg.MaxAge,     // 保留天数
			Compress:   cfg.Compress,   // 旧文件 gzip 压缩
		}
	} else {
		writer = os.Stdout
	}

	var level slog.Level
	switch strings.ToLower(cfg.Level) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	// 短文件名：只显示 main.go:123 而非 /home/admin/gim/cmd/gim/main.go:123
	if cfg.ShortFile {
		opts.AddSource = true
		opts.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.SourceKey {
				if src, ok := a.Value.Any().(*slog.Source); ok {
					src.File = filepath.Base(src.File)
				}
			}
			return a
		}
	}

	var handler slog.Handler
	if cfg.Format == "json" {
		// JSON 格式适合生产环境日志聚合（ELK/Loki）
		handler = slog.NewJSONHandler(writer, opts)
	} else {
		if cfg.Color && cfg.Output != "file" {
			// 开发环境彩色输出，按日志等级着色
			handler = NewColoredTextHandler(writer, opts)
		} else {
			handler = slog.NewTextHandler(writer, opts)
		}
	}

	return &Logger{Logger: slog.New(handler)}
}

// ANSI 颜色码
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m" // ERROR
	colorYellow = "\033[33m" // WARN
	colorBlue   = "\033[34m" // INFO
	colorPurple = "\033[35m" // DEBUG
)

// ColoredTextHandler 在 TextHandler 输出前后包裹 ANSI 颜色码。
// 不同日志等级用不同颜色：DEBUG=紫 INFO=蓝 WARN=黄 ERROR=红。
type ColoredTextHandler struct {
	*slog.TextHandler
	writer io.Writer
}

func NewColoredTextHandler(w io.Writer, opts *slog.HandlerOptions) *ColoredTextHandler {
	return &ColoredTextHandler{
		TextHandler: slog.NewTextHandler(w, opts),
		writer:      w,
	}
}

func (h *ColoredTextHandler) Handle(ctx context.Context, r slog.Record) error {
	var color string
	switch r.Level {
	case slog.LevelDebug:
		color = colorPurple
	case slog.LevelInfo:
		color = colorBlue
	case slog.LevelWarn:
		color = colorYellow
	case slog.LevelError:
		color = colorRed
	default:
		color = colorReset
	}
	h.writer.Write([]byte(color))
	err := h.TextHandler.Handle(ctx, r)
	h.writer.Write([]byte(colorReset))
	return err
}

// Fatal 输出 Error 级别日志后退出程序（os.Exit(1)）。
func (l *Logger) Fatal(msg string, args ...any) {
	l.Error(msg, args...)
	os.Exit(1)
}
