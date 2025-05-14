package testutils

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"runtime"
	"time"
)

type LogfBridge struct {
	level slog.Level
	attrs []slog.Attr
	ctx   context.Context
}

func NewLogfBridge(ctx context.Context, level slog.Level, attrs ...slog.Attr) *LogfBridge {
	return &LogfBridge{level: level, attrs: attrs, ctx: ctx}
}

func (l *LogfBridge) Logf(f string, v ...interface{}) {
	callerpc, _, _, ok := runtime.Caller(1)
	if !ok {
		callerpc = 0
	}

	message := fmt.Sprintf(f, v...)

	record := slog.NewRecord(time.Now(), l.level, message, callerpc)
	record.AddAttrs(l.attrs...)

	// Send to slog

	ctx := l.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	err := slog.Default().Handler().Handle(ctx, record)
	if err != nil {
		log.Printf("error logging: %v", err)
	}
}
