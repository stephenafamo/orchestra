package orchestra

import (
	"context"
	"log/slog"
)

// Logger is accepted by some Players ([Conductor], [ServerPlayer])
type Logger interface {
	Info(msg string, attrs ...slog.Attr)
	Error(msg string, attrs ...slog.Attr)
	WithGroup(name string) Logger
}

var _ slogInterface = &slog.Logger{}

type slogInterface interface {
	LogAttrs(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr)
	WithGroup(name string) *slog.Logger
}

func LoggerFromSlog(infoLevel, errorLevel slog.Level, l slogInterface) Logger {
	return slogLogger{infoLevel, errorLevel, l}
}

// DefaultLogger is used when a conductor's logger is nil
var DefaultLogger Logger = LoggerFromSlog(slog.LevelInfo, slog.LevelError, slog.Default())

type slogLogger struct {
	lvlInfo  slog.Level
	lvlError slog.Level
	logger   slogInterface
}

func (d slogLogger) Info(msg string, attrs ...slog.Attr) {
	d.logger.LogAttrs(context.Background(), d.lvlInfo, msg, attrs...)
}

func (d slogLogger) Error(msg string, attrs ...slog.Attr) {
	d.logger.LogAttrs(context.Background(), d.lvlError, msg, attrs...)
}

func (d slogLogger) WithGroup(name string) Logger {
	return slogLogger{d.lvlInfo, d.lvlError, d.logger.WithGroup(name)}
}
