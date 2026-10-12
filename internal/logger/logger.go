// Package logger configures the process-wide structured (JSON) logger.
package logger

import (
	"io"
	"log/slog"
	"os"
)

// New returns a JSON slog logger writing to stdout at the given level
// (debug, info, warn, error). Unknown levels fall back to info.
func New(level string) *slog.Logger {
	return NewWithWriter(os.Stdout, level)
}

// NewWithWriter is New with an explicit destination, used by tests.
func NewWithWriter(w io.Writer, level string) *slog.Logger {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: lvl}))
}
