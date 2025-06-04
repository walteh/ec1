package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/lmittmann/tint"
	"gitlab.com/tozd/go/errors"

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
	return SetupSlogSimpleToWriterWithProcessName(ctx, w, color, "", processor...)
}

func SetupSlogSimpleToWriterWithProcessName(ctx context.Context, w io.Writer, color bool, processName string, processor ...SlogProcessor) context.Context {

	tintHandler := tint.NewHandler(w, &tint.Options{
		Level:      slog.LevelDebug,
		TimeFormat: "2006-01-02 15:04 05.0000",
		AddSource:  true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			a = formatErrorStacks(groups, a)
			return Redact(groups, a)
		},
		NoColor:     !color,
		ProcessName: processName,
	})

	ctxHandler := slogctx.NewHandler(tintHandler, nil)

	mylogger := slog.New(ctxHandler)
	slog.SetDefault(mylogger)

	return slogctx.NewCtx(ctx, mylogger)
}

func packageName(frame runtime.Frame) string {
	lastSlash := strings.LastIndex(frame.Function, "/")
	if lastSlash == -1 {
		return ""
	}
	almost := frame.Function[:lastSlash]
	remaining := frame.Function[lastSlash+1:]
	firstDot := strings.Index(remaining, ".")
	if firstDot == -1 {
		return ""
	}
	return almost + "/" + remaining[:firstDot]
}

func link(url string, text string) string {
	return fmt.Sprintf("\\e]8;;%s\\e\\%s\\e]8;;\\e\\", url, text)
}

type CallerURI struct {
	Package string
	File    string
	Line    int
}

func GetCurrentCallerURI() CallerURI {
	ptr, _, _, _ := runtime.Caller(1)
	return getCallerURI(ptr)
}

func getCallerURI(ptr uintptr) CallerURI {
	frames := runtime.CallersFrames([]uintptr{ptr})
	frame, _ := frames.Next()
	pkg := packageName(frame)
	uri := fmt.Sprintf("%s:%d", frame.File, frame.Line)
	return CallerURI{
		Package: pkg,
		File:    filepath.Base(filepath.Dir(uri)),
		Line:    frame.Line,
	}
}
func formatErrorStacks(groups []string, a slog.Attr) slog.Attr {
	if a.Key == "error" {
		if err, ok := a.Value.Any().(error); ok {
			if terr, ok := err.(errors.E); ok {
				frames := runtime.CallersFrames(terr.StackTrace())
				firstFramed, _ := frames.Next()
				pkg := packageName(firstFramed)
				uri := fmt.Sprintf("%s:%d", firstFramed.File, firstFramed.Line)
				a.Value = slog.GroupValue(
					slog.String("message", fmt.Sprintf("%v", err)),
					slog.String("func", strings.TrimPrefix(firstFramed.Function, pkg+".")),
					slog.String("package", pkg),
					// the quotes are to make sure the file name can be clicked by vscode/cursor
					slog.String("file", "'"+filepath.Base(filepath.Dir(uri))+"/"+filepath.Base(uri)+"'"),
				)
			}
		}
	}
	return a
}
