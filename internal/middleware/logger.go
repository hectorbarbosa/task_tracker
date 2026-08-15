package middleware

import (
	"log/slog"
	"os"
)

// NewLogger creates a new structured logger using slog.
// It returns a JSON handler for production and text handler for development.
func NewLogger(env string) *slog.Logger {
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true, // Adds file name and line number (like zap)
	}

	if env == "development" {
		// Text handler for better readability in development
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		// JSON handler for production (structured logging)
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logger
}
