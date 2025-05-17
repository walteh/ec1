package logging

import (
	"context"
	"io"
	"log/slog"
	"os"

	"github.com/lmittmann/tint"

	slogctx "github.com/veqryn/slog-context"
)

func SetupSlogSimple(ctx context.Context) context.Context {
	return SetupSlogSimpleToWriter(ctx, os.Stdout, true)
}

func SetupSlogSimpleNoColor(ctx context.Context) context.Context {
	return SetupSlogSimpleToWriter(ctx, os.Stdout, false)
}

type SlogProcessor interface {
	Process(ctx context.Context, a slog.Handler) slog.Handler
}

type SlogProcessorFunc func(ctx context.Context, a slog.Handler) slog.Handler

func (f SlogProcessorFunc) Process(ctx context.Context, a slog.Handler) slog.Handler {
	return f(ctx, a)
}

type TintProcessor struct {
	color bool
}

func SetupSlogSimpleToWriter(ctx context.Context, w io.Writer, color bool, processor ...SlogProcessor) context.Context {

	tintHandler := tint.NewHandler(w, &tint.Options{
		Level:      slog.LevelDebug,
		TimeFormat: "2006-01-02 15:04 05.0000",
		AddSource:  true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			return Redact(groups, a)
		},
		NoColor: !color,
	})

	ctxHandler := slogctx.NewHandler(tintHandler, nil)

	mylogger := slog.New(ctxHandler)
	slog.SetDefault(mylogger)

	return slogctx.NewCtx(ctx, mylogger)
}
