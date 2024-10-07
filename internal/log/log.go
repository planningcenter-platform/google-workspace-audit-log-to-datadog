package log

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

func init() {
	logger := slog.New(newLoggerHandler(os.Stdout))

	slog.SetDefault(logger)
}

func newLoggerHandler(w io.Writer) slog.Handler {
	return slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: getLoggerLevelFromEnv(),
	})
}

func getLoggerLevelFromEnv() slog.Level {
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "trace":
		return slog.LevelDebug - 1
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

func Default() *slog.Logger {
	return slog.Default()
}

func Log(ctx context.Context, level slog.Level, msg string, args ...any) {
	Default().Log(ctx, level, msg, args...)
}

func Trace(ctx context.Context, message string, args ...any) {
	Log(ctx, slog.LevelDebug-1, message, args...)
}

func Debug(ctx context.Context, message string, args ...any) {
	Log(ctx, slog.LevelDebug, message, args...)
}

func Info(ctx context.Context, message string, args ...any) {
	Log(ctx, slog.LevelInfo, message, args...)
}

func Warn(ctx context.Context, message string, args ...any) {
	Log(ctx, slog.LevelWarn, message, args...)
}

func Error(ctx context.Context, message string, args ...any) {
	var normalized = make([]any, len(args))
	for i, arg := range args {
		switch arg := arg.(type) {
		case error:
			normalized[i] = slog.Any("error", arg)
		default:
			normalized[i] = arg
		}
	}
	Log(ctx, slog.LevelError, message, normalized...)
}
