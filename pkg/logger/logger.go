// Package logger provides structured logging via log/slog.
//
// Call Init() early in main to configure the default slog logger.
// All other packages should use slog.Info/Debug/Warn/Error directly.
// This package also provides a Fatal helper (slog has no Fatal).
package logger

import (
	"log/slog"
	"os"
	"strings"
)

// Init configures the default slog logger.
//
//	format: "json" | "text" (default "text")
//	level:  "debug" | "info" | "warn" | "error" (default "info")
func Init(format, level string) {
	lvl := parseLevel(level)

	opts := &slog.HandlerOptions{
		Level:     lvl,
		AddSource: true,
	}

	var handler slog.Handler
	switch strings.ToLower(format) {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	default:
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	slog.SetDefault(slog.New(handler))
}

// Fatal logs at error level and exits the process.
// slog does not provide a Fatal level, so this is a convenience wrapper.
func Fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
