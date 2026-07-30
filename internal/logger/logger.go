// Package logger provides leveled, structured logging built on the standard
// library's log/slog, keeping Repo Mapper dependency-free for something as
// foundational as logging.
package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// Level mirrors the PRD's four levels (Debug, Info, Warning, Error).
type Level string

const (
	LevelDebug   Level = "debug"
	LevelInfo    Level = "info"
	LevelWarning Level = "warning"
	LevelError   Level = "error"
)

func (l Level) toSlog() slog.Level {
	switch strings.ToLower(string(l)) {
	case string(LevelDebug):
		return slog.LevelDebug
	case string(LevelWarning):
		return slog.LevelWarn
	case string(LevelError):
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Logger wraps *slog.Logger to give Repo Mapper a small, stable surface.
type Logger struct {
	*slog.Logger
}

// New builds a structured text logger writing to w at the given level.
func New(w io.Writer, level Level) *Logger {
	h := slog.NewTextHandler(w, &slog.HandlerOptions{Level: level.toSlog()})
	return &Logger{Logger: slog.New(h)}
}

// Default returns a logger writing to stderr at Info level.
func Default() *Logger {
	return New(os.Stderr, LevelInfo)
}

// Nop returns a logger that discards all output. Useful in tests.
func Nop() *Logger {
	return New(io.Discard, LevelError)
}

// With returns a child logger with additional structured fields.
func (l *Logger) With(args ...any) *Logger {
	return &Logger{Logger: l.Logger.With(args...)}
}
