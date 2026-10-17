package config

import (
	"log/slog"
	"os"
	"strings"
)

// InitLogger configures the default slog logger from LOG_LEVEL (debug, info,
// warn, error; default info) and LOG_FORMAT (json or text; default json).
// It also redirects the standard library "log" package through slog, so any
// remaining log.Printf calls are emitted as structured records at info level.
func InitLogger() {
	var level slog.Level
	if err := level.UnmarshalText([]byte(getEnvWithDefault("LOG_LEVEL", "info"))); err != nil {
		level = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if strings.EqualFold(getEnvWithDefault("LOG_FORMAT", "json"), "text") {
		handler = slog.NewTextHandler(os.Stderr, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	}
	slog.SetDefault(slog.New(handler))
}
