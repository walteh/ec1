package testutils

import (
	"context"
	"io"
	"log/slog"

	"github.com/sirupsen/logrus"
)

var _ logrus.Hook = &SlogBridgeHook{}

func init() {
	logrus.SetReportCaller(true)
	logrus.AddHook(&SlogBridgeHook{})
	logrus.SetOutput(io.Discard)
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableColors: true,
	})
}

type SlogBridgeHook struct {
}

func (h *SlogBridgeHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *SlogBridgeHook) Fire(entry *logrus.Entry) error {
	// Map logrus levels to slog levels
	var level slog.Level
	switch entry.Level {
	case logrus.PanicLevel, logrus.FatalLevel, logrus.ErrorLevel:
		level = slog.LevelError
	case logrus.WarnLevel:
		level = slog.LevelWarn
	case logrus.InfoLevel:
		level = slog.LevelInfo
	case logrus.DebugLevel, logrus.TraceLevel:
		level = slog.LevelDebug
	default:
		level = slog.LevelInfo
	}

	// Prepare slog attributes
	attrs := make([]slog.Attr, 0, len(entry.Data))
	for k, v := range entry.Data {
		attrs = append(attrs, slog.Any(k, v))
	}

	record := slog.NewRecord(entry.Time, level, entry.Message, entry.Caller.PC)
	record.AddAttrs(attrs...)

	// Send to slog

	ctx := entry.Context
	if ctx == nil {
		ctx = context.Background()
	}

	return slog.Default().Handler().Handle(ctx, record)
}
