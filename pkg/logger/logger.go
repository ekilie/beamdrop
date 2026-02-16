// Package logger provides dual-output structured logging.
//
// Terminal output is human-readable with colors.
// File output is structured JSON written to <sharedDir>/.beamdrop/beamdrop.log.
//
// Call Init() early in main to configure the default slog logger.
// All other packages should use slog.Info/Debug/Warn/Error directly.
// This package also provides a Fatal helper (slog has no Fatal).
package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
)

// ANSI color printers for each log level.
var (
	debugColor = color.New(color.FgHiCyan)
	infoColor  = color.New(color.FgHiGreen)
	warnColor  = color.New(color.FgHiYellow)
	errorColor = color.New(color.FgHiRed, color.Bold)
	timeColor  = color.New(color.FgWhite, color.Faint)
	srcColor   = color.New(color.FgHiBlack)
	keyColor   = color.New(color.FgHiBlue)
)

// logFile holds the open log file handle so we can close it later.
var logFile *os.File

// Init configures the default slog logger with dual output.
//
//	level:    "debug" | "info" | "warn" | "error" (default "info")
//	sharedDir: directory where .beamdrop/beamdrop.log will be written (empty = no file logging)
func Init(level, sharedDir string) {
	lvl := parseLevel(level)

	// --- Terminal handler (colored, human-readable) ---
	termHandler := &colorHandler{
		level: lvl,
		w:     os.Stdout,
	}

	// --- File handler (structured JSON) ---
	var fileHandler slog.Handler
	if sharedDir != "" {
		logDir := filepath.Join(sharedDir, ".beamdrop")
		if err := os.MkdirAll(logDir, 0755); err == nil {
			logPath := filepath.Join(logDir, "beamdrop.log")
			f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err == nil {
				logFile = f
				fileHandler = slog.NewJSONHandler(f, &slog.HandlerOptions{
					Level:     lvl,
					AddSource: true,
				})
			}
		}
	}

	var handler slog.Handler
	if fileHandler != nil {
		handler = &multiHandler{handlers: []slog.Handler{termHandler, fileHandler}}
	} else {
		handler = termHandler
	}

	slog.SetDefault(slog.New(handler))
}

// Close closes the log file. Call this on shutdown if desired.
func Close() {
	if logFile != nil {
		logFile.Close()
	}
}

// Fatal logs at error level and exits the process.
// slog does not provide a Fatal level, so this is a convenience wrapper.
func Fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}

// ---------------------------------------------------------------------------
// colorHandler – a slog.Handler that writes colored, human-readable output.
// ---------------------------------------------------------------------------

type colorHandler struct {
	level  slog.Level
	w      io.Writer
	mu     sync.Mutex
	attrs  []slog.Attr
	groups []string
}

func (h *colorHandler) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.level
}

func (h *colorHandler) Handle(_ context.Context, r slog.Record) error {
	var buf strings.Builder

	// Timestamp
	timeColor.Fprint(&buf, r.Time.Format("15:04:05.000"))
	buf.WriteByte(' ')

	// Level with color
	levelStr := r.Level.String()
	switch {
	case r.Level >= slog.LevelError:
		errorColor.Fprintf(&buf, "%-5s", levelStr)
	case r.Level >= slog.LevelWarn:
		warnColor.Fprintf(&buf, "%-5s", levelStr)
	case r.Level >= slog.LevelInfo:
		infoColor.Fprintf(&buf, "%-5s", levelStr)
	default:
		debugColor.Fprintf(&buf, "%-5s", levelStr)
	}
	buf.WriteByte(' ')

	// Source (file:line) – short path
	if r.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		if f.File != "" {
			file := shortFile(f.File)
			srcColor.Fprintf(&buf, "%s:%d", file, f.Line)
			buf.WriteByte(' ')
		}
	}

	// Message
	fmt.Fprint(&buf, r.Message)

	// Pre-set attrs from WithAttrs
	for _, a := range h.attrs {
		h.writeAttr(&buf, a)
	}

	// Record attrs
	r.Attrs(func(a slog.Attr) bool {
		h.writeAttr(&buf, a)
		return true
	})

	buf.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := io.WriteString(h.w, buf.String())
	return err
}

func (h *colorHandler) writeAttr(buf *strings.Builder, a slog.Attr) {
	buf.WriteByte(' ')
	keyColor.Fprint(buf, a.Key)
	buf.WriteByte('=')
	fmt.Fprint(buf, a.Value)
}

func (h *colorHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &colorHandler{
		level:  h.level,
		w:      h.w,
		attrs:  append(append([]slog.Attr{}, h.attrs...), attrs...),
		groups: h.groups,
	}
}

func (h *colorHandler) WithGroup(name string) slog.Handler {
	return &colorHandler{
		level:  h.level,
		w:      h.w,
		attrs:  h.attrs,
		groups: append(append([]string{}, h.groups...), name),
	}
}

// shortFile trims the path to show only the last two directory components.
func shortFile(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	if len(parts) > 2 {
		return strings.Join(parts[len(parts)-2:], "/")
	}
	return path
}

// ---------------------------------------------------------------------------
// multiHandler – fans out each log record to multiple slog.Handlers.
// ---------------------------------------------------------------------------

type multiHandler struct {
	handlers []slog.Handler
}

func (m *multiHandler) Enabled(ctx context.Context, l slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, l) {
			return true
		}
	}
	return false
}

func (m *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, h := range m.handlers {
		if h.Enabled(ctx, r.Level) {
			if err := h.Handle(ctx, r); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithAttrs(attrs)
	}
	return &multiHandler{handlers: handlers}
}

func (m *multiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithGroup(name)
	}
	return &multiHandler{handlers: handlers}
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
