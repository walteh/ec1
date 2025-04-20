package clog

import (
	"context"
	"log/slog"

	slogctx "github.com/veqryn/slog-context"
)

func Ctx(ctx context.Context) *slog.Logger {
	return slogctx.FromCtx(ctx)
}

func WithCtx(ctx context.Context, logger *slog.Logger) context.Context {
	return slogctx.NewCtx(ctx, logger)
}

func AddAttrs(ctx context.Context, attrs ...any) context.Context {
	return slogctx.With(ctx, attrs...)
}

func WithAttrs(ctx context.Context, attrs ...any) context.Context {
	return slogctx.With(ctx, attrs...)
}

func NewLoggerFromHandler(ctx context.Context, handler slog.Handler) (*slog.Logger, context.Context) {
	customHandler := slogctx.NewHandler(handler, nil)
	logger := slog.New(customHandler)
	ctx = slogctx.NewCtx(ctx, logger)
	return logger, ctx
}
