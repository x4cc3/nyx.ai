package observability

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"

	"nyx/internal/ids"
)

type LoggerConfig struct {
	Service string
	Format  string
	Level   string
	Writer  io.Writer
}

type traceContextKey struct{}

func NewLogger(cfg LoggerConfig) *slog.Logger {
	writer := cfg.Writer
	if writer == nil {
		writer = os.Stdout
	}
	level := parseLevel(cfg.Level)
	opts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	if strings.EqualFold(strings.TrimSpace(cfg.Format), "text") {
		handler = slog.NewTextHandler(writer, opts)
	} else {
		handler = slog.NewJSONHandler(writer, opts)
	}
	return slog.New(handler).With("service", cfg.Service)
}

func WithTrace(ctx context.Context, traceID string) context.Context {
	if strings.TrimSpace(traceID) == "" {
		traceID = ids.New("trace")
	}
	return context.WithValue(ctx, traceContextKey{}, traceID)
}

func TraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if value, ok := ctx.Value(traceContextKey{}).(string); ok {
		return value
	}
	return ""
}

func EnsureTrace(ctx context.Context) (context.Context, string) {
	traceID := TraceID(ctx)
	if traceID != "" {
		return ctx, traceID
	}
	traceID = ids.New("trace")
	return WithTrace(ctx, traceID), traceID
}

func parseLevel(raw string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
