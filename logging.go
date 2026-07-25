package main

import (
	"log/slog"
	"os"
	"strings"
)

// parseLogLevel maps a PUSHOVER_LOG_LEVEL value to a slog.Level.
func parseLogLevel(value string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// setupServerLogging configures slog with a JSON handler writing to stderr.
// Stderr is required because in STDIO transport stdout carries JSON-RPC frames.
func setupServerLogging(transport string) {
	level := parseLogLevel(os.Getenv("PUSHOVER_LOG_LEVEL"))
	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	logger := slog.New(handler).With("service", "pushover-mcp", "transport", transport)
	slog.SetDefault(logger)
}
