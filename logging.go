package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// parseLogLevel maps a PUSHOVER_LOG_LEVEL value to a slog.Level.
func parseLogLevel(value string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "debug":
		return slog.LevelDebug, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	case "", "info":
		return slog.LevelInfo, nil
	default:
		return slog.LevelInfo, fmt.Errorf("PUSHOVER_LOG_LEVEL: invalid value %q", value)
	}
}

// setupServerLogging configures slog with a JSON handler writing to stderr.
// Stderr is required because in STDIO transport stdout carries JSON-RPC frames.
func setupServerLogging(transport string) error {
	level, err := parseLogLevel(os.Getenv("PUSHOVER_LOG_LEVEL"))
	if err != nil {
		return err
	}
	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	logger := slog.New(handler).With("service", "pushover-mcp", "transport", transport)
	slog.SetDefault(logger)
	return nil
}
