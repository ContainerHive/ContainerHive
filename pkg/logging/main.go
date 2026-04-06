package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/lmittmann/tint"
)

// Setup initializes and configures the global logger with colorized logging
func Setup(w io.Writer, level string) {
	logLevel := parseLevel(level)
	slog.SetDefault(slog.New(
		tint.NewHandler(w, &tint.Options{
			Level:      logLevel,
			TimeFormat: time.DateTime,
			NoColor:    os.Getenv("NO_COLOR") != "",
			AddSource:  logLevel == slog.LevelDebug,
		}),
	))
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
