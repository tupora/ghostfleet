package logging

import (
	"fmt"
	"io"
	"log/slog"
)

func New(level string, output io.Writer) (*slog.Logger, error) {
	var slogLevel slog.Level
	switch level {
	case "debug":
		slogLevel = slog.LevelDebug
	case "info":
		slogLevel = slog.LevelInfo
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		return nil, fmt.Errorf("unknown log level %q", level)
	}
	return slog.New(slog.NewJSONHandler(output, &slog.HandlerOptions{Level: slogLevel})), nil
}
