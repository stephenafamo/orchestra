package orchestra

import (
	"context"
	"log/slog"
)

// Logger is accepted by some Players ([Conductor], [ServerPlayer])
type Logger interface {
	Log(msg string, attrs ...slog.Attr)
}

type slogInterface interface {
	LogAttrs(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr)
}

func LoggerFromSlog(level slog.Level, l slogInterface) Logger {
	return slogLogger{level, l}
}

// DefaultLogger is used when a conductor's logger is nil
var DefaultLogger Logger = LoggerFromSlog(slog.LevelInfo, slog.Default())

type slogLogger struct {
	lvl    slog.Level
	logger slogInterface
}

func (d slogLogger) Log(msg string, attrs ...slog.Attr) {
	d.logger.LogAttrs(context.Background(), d.lvl, msg, attrs...)
}

type subConductorLogger struct {
	name string
	l    Logger
}

func (s subConductorLogger) Log(msg string, attrs ...slog.Attr) {
	l := s.l
	if s.l == nil {
		l = DefaultLogger
	}

	l.Log(msg, append([]slog.Attr{slog.String("conductor", s.name)}, attrs...)...)
}
