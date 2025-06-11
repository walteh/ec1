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

	// "github.com/golang-cz/devslog"

	"gitlab.com/tozd/go/errors"

	"github.com/muesli/termenv"
	slogctx "github.com/veqryn/slog-context"

	"github.com/walteh/ec1/pkg/logging/termlog"
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

var logWriter io.Writer

func SetupSlogSimpleToWriterWithProcessNameJSON(ctx context.Context, w io.Writer, color bool, processName string, processor ...SlogProcessor) context.Context {
	devHandler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			a = formatErrorStacks(groups, a)
			return Redact(groups, a)
		},
	})

	ctxHandler := slogctx.NewHandler(devHandler, &slogctx.HandlerOptions{})

	mylogger := slog.New(ctxHandler)
	slog.SetDefault(mylogger)
	logWriter = w

	return slogctx.NewCtx(ctx, mylogger)
}

func SetupSlogSimpleToWriterWithProcessName(ctx context.Context, w io.Writer, color bool, processName string, processor ...SlogProcessor) context.Context {

	// devHandler := tint.NewHandler(w, &tint.Options{
	// 	Level:      slog.LevelDebug,
	// 	TimeFormat: "2006-01-02 15:04 05.0000",
	// 	AddSource:  true,
	// 	ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
	// 		a = formatErrorStacks(groups, a)
	// 		return Redact(groups, a)
	// 	},
	// 	NoColor:     !color,
	// 	ProcessName: processName,
	// })

	// Override the default error level style.
	// styles := clog.DefaultStyles()
	// styles.Levels[clog.ErrorLevel] = lipgloss.NewStyle().
	// 	SetString("ERROR!!").
	// 	Padding(0, 1, 0, 1).
	// 	Background(lipgloss.Color("204")).
	// 	Foreground(lipgloss.Color("0"))
	// // Add a custom style for key `err`
	// styles.Keys["err"] = lipgloss.NewStyle().Foreground(lipgloss.Color("204"))
	// styles.Values["err"] = lipgloss.NewStyle().Bold(true)

	// clogHandler := clog.NewWithOptions(w, clog.Options{
	// 	CallerFormatter: func(file string, line int, _ string) string {
	// 		return NewEnhancedSource(uintptr(line)).ColorizedString(termenv.ANSI)
	// 	},

	// 	Level:        clog.DebugLevel,
	// 	TimeFormat:   "2006-01-02 15:04 05.0000",
	// 	Prefix:       processName,
	// 	ReportCaller: true,
	// })

	// log

	slogOpts := &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			a = formatErrorStacks(groups, a)
			return Redact(groups, a)
		},
	}
	clogHandler := termlog.NewTermLogger(w, slogOpts,
		termlog.WithLoggerName(processName),
		termlog.WithProfile(termenv.ANSI256),
		termlog.WithRenderOption(termenv.WithTTY(true)),
		termlog.WithLoggerName(processName),
	)

	// clog.StandardLog(clog.StandardLogOptions{
	// 	ForceLevel: clog.DebugLevel,
	// })

	// opts := slog.HandlerOptions{
	// 	Level:     slog.LevelDebug,
	// 	AddSource: true,
	// 	ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
	// 		a = formatErrorStacks(groups, a)
	// 		return Redact(groups, a)
	// 	},
	// }

	// devHandler := devslog.NewHandler(w, &devslog.Options{
	// 	HandlerOptions:      &opts,
	// 	NewLineAfterLog:     true,
	// 	SameSourceInfoColor: color,
	// })

	ctxHandler := slogctx.NewHandler(clogHandler, &slogctx.HandlerOptions{})

	mylogger := slog.New(ctxHandler)
	slog.SetDefault(mylogger)
	logWriter = w

	return slogctx.NewCtx(ctx, mylogger)
}

func GetDefaultLogWriter() io.Writer {
	if logWriter == nil {
		return os.Stdout
	}
	return logWriter
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
	Package  string
	File     string
	Line     int
	Function string
}

func GetCurrentCallerURI() CallerURI {
	ptr, _, _, _ := runtime.Caller(1)
	return getCallerURI(ptr)
}

func GetCurrentCallerURIOffset(offset int) CallerURI {
	ptr, _, _, _ := runtime.Caller(1 + offset)
	return getCallerURI(ptr)
}

func getCallerURI(ptr uintptr) CallerURI {
	frames := runtime.CallersFrames([]uintptr{ptr})
	frame, _ := frames.Next()
	pkg := packageName(frame)
	uri := fmt.Sprintf("%s:%d", frame.File, frame.Line)
	return CallerURI{
		Package:  pkg,
		File:     filepath.Base(filepath.Dir(uri)),
		Line:     frame.Line,
		Function: frame.Function,
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
					slog.Any("error", err),
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
