package logging

import (
	"io"
	"log/slog"

	"github.com/gi8lino/tiledash/internal/flag"
)

// LogFormat defines the supported log formats.
type LogFormat string

const (
	LogFormatText LogFormat = "text"
	LogFormatJSON LogFormat = "json"
)

// SetupLogger configures a structured logger with the parsed CLI flags.
func SetupLogger(cfg flag.Config, output io.Writer) *slog.Logger {
	handlerOpts := &slog.HandlerOptions{}

	if cfg.Debug {
		handlerOpts.Level = slog.LevelDebug
	}

	var handler slog.Handler
	switch LogFormat(cfg.LogFormat) {
	case LogFormatJSON:
		handler = slog.NewJSONHandler(output, handlerOpts)
	case LogFormatText:
		handler = slog.NewTextHandler(output, handlerOpts)
	default:
		// Default to JSON if an invalid format is provided.
		handler = slog.NewJSONHandler(output, handlerOpts)
	}

	return slog.New(handler)
}
