package logger

import (
	"log/slog"
	"os"
	"strings"
)

var (
	// Logger is the global slog logger instance
	Logger *slog.Logger
)

// Init initializes the global logger with the configured level from LOG_LEVEL environment variable
// Default level is INFO for production, DEBUG for development
func Init() {
	// Get log level from environment variable
	logLevelStr := os.Getenv("LOG_LEVEL")
	if logLevelStr == "" {
		// Default to info level
		logLevelStr = "info"
	}

	// Parse log level
	var level slog.Level
	switch strings.ToLower(logLevelStr) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Create handler options with the configured level
	opts := &slog.HandlerOptions{
		Level: level,
	}

	// Create JSON handler for structured logging
	handler := slog.NewJSONHandler(os.Stdout, opts)

	// Set the global logger
	Logger = slog.New(handler)
	slog.SetDefault(Logger)

	Logger.Info("Logger initialized", "level", logLevelStr)
}

// Debug logs a debug message
func Debug(msg string, args ...any) {
	Logger.Debug(msg, args...)
}

// Info logs an info message
func Info(msg string, args ...any) {
	Logger.Info(msg, args...)
}

// Warn logs a warning message
func Warn(msg string, args ...any) {
	Logger.Warn(msg, args...)
}

// Error logs an error message
func Error(msg string, args ...any) {
	Logger.Error(msg, args...)
}
