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
		Level:     slog.LevelDebug,
		AddSource: false, // Remove file/line info for cleaner output
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
