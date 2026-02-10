// Package logging provides structured logging setup for house-finder.
package logging

import (
	"log/slog"
	"os"
)

// Setup initializes the default slog logger.
// Dev mode uses human-readable text; prod uses JSON.
func Setup(devMode bool) {
	var handler slog.Handler
	if devMode {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	} else {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	}
	slog.SetDefault(slog.New(handler))
}
