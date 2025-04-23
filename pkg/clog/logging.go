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

var _ slog.Handler = (*orchestratorHandlerWorker)(nil)

type OrchestratorHandler struct {
	DefaultHandler slog.Handler
	Handlers       map[string]slog.Handler
}

func NewOrchestratorHandler(defaultHandler slog.Handler) *OrchestratorHandler {
	return &OrchestratorHandler{
		DefaultHandler: defaultHandler,
		Handlers:       make(map[string]slog.Handler),
	}
}

func (me *OrchestratorHandler) RegisterHandler(group string, handler slog.Handler) {
	me.Handlers[group] = handler
}

func (me *OrchestratorHandler) HandlerFor(group string) slog.Handler {
	if handler, ok := me.Handlers[group]; ok {
		return handler
	}
	if me.DefaultHandler != nil {
		return me.DefaultHandler
	}
	panic("no default handler registered")
}

type orchestratorHandlerWorker struct {
	parent  *OrchestratorHandler
	MyAttrs []slog.Attr
	MyGroup string
}

// Enabled implements slog.Handler.
func (h *orchestratorHandlerWorker) Enabled(ctx context.Context, level slog.Level) bool {
	return h.parent.HandlerFor(h.MyGroup).Enabled(ctx, level)
}

// WithAttrs implements slog.Handler.
func (h *orchestratorHandlerWorker) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &orchestratorHandlerWorker{
		parent:  h.parent,
		MyAttrs: append(h.MyAttrs, attrs...),
		MyGroup: h.MyGroup,
	}
}

// WithGroup implements slog.Handler.
func (h *orchestratorHandlerWorker) WithGroup(name string) slog.Handler {
	return &orchestratorHandlerWorker{
		parent:  h.parent,
		MyAttrs: append(h.MyAttrs, []slog.Attr{}...), // to copy the slice
		MyGroup: name,
	}
}

func (h *orchestratorHandlerWorker) Handle(ctx context.Context, record slog.Record) error {
	return h.parent.HandlerFor(h.MyGroup).Handle(ctx, record)
}
