package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	// "github.com/golang-cz/devslog"
	slogmulti "github.com/samber/slog-multi"
	"gitlab.com/tozd/go/errors"
	"gopkg.in/natefinch/lumberjack.v2"

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
			a = formatErrorStacks2(groups, a)
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

	slogOpts := &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			a = formatErrorStacks2(groups, a)
			return Redact(groups, a)
		},
	}

	lumberjackLogger := &lumberjack.Logger{
		Filename:   "/tmp/ec1.log", // Path to your log file
		MaxSize:    10,             // Max size in megabytes before rotation
		MaxBackups: 5,              // Max number of old log files to retain
		MaxAge:     1,              // Max number of days to retain old log files
		Compress:   true,           // Compress old log files
	}

	fileHandler := slog.NewJSONHandler(lumberjackLogger, slogOpts)

	clogHandler := termlog.NewTermLogger(w, slogOpts,
		termlog.WithLoggerName(processName),
		termlog.WithProfile(termenv.ANSI256),
		termlog.WithRenderOption(termenv.WithTTY(true)),
		termlog.WithLoggerName(processName),
	)

	multiHandler := slogmulti.Fanout(fileHandler, clogHandler)

	ctxHandler := slogctx.NewHandler(multiHandler, &slogctx.HandlerOptions{})

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
	pkg := packageName(&frame)
	uri := fmt.Sprintf("%s:%d", frame.File, frame.Line)
	return CallerURI{
		Package:  pkg,
		File:     filepath.Base(filepath.Dir(uri)),
		Line:     frame.Line,
		Function: frame.Function,
	}
}

func packageName(frame *runtime.Frame) string {
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
func formatErrorStacks(groups []string, a slog.Attr) slog.Attr {
	if a.Key == "error" {
		if err, ok := a.Value.Any().(error); ok {
			if terr, ok := err.(errors.E); ok {
				trc := terr.StackTrace()
				frames := runtime.CallersFrames(trc[0:1])
				firstFramed, _ := frames.Next()
				pkg := packageName(&firstFramed)
				uri := fmt.Sprintf("%s:%d", firstFramed.File, firstFramed.Line)
				a.Value = slog.GroupValue(
					slog.Any("error", err),
					slog.String("func", strings.TrimPrefix(firstFramed.Function, pkg+".")),
					slog.String("package", pkg),
					// the quotes are to make sure the file name can be clicked by vscode/cursor
					slog.String("file", "'"+filepath.Base(filepath.Dir(uri))+"/"+filepath.Base(uri)+"'"),
				)
				if runtime.GOOS == "linux" {
					fmt.Println("error processed", err)
				}
			}
		}
	}
	return a
}

var pkgCache sync.Map // funcName → packageName

func packageName2(funcName string) string {
	if v, ok := pkgCache.Load(funcName); ok {
		return v.(string)
	}
	// one-time compute:
	slash := strings.LastIndex(funcName, "/")
	pkg := ""
	if slash >= 0 {
		rem := funcName[slash+1:]
		if dot := strings.IndexByte(rem, '.'); dot >= 0 {
			pkg = funcName[:slash] + "/" + rem[:dot]
		}
	}
	pkgCache.Store(funcName, pkg)
	return pkg
}

func formatErrorStacks2(groups []string, a slog.Attr) slog.Attr {
	if a.Key != "error" {
		return a
	}
	errVal, ok := a.Value.Any().(error)
	if !ok {
		return a
	}

	var frame uintptr
	switch v := errVal.(type) {
	case interface{ Frame() uintptr }:
		frame = v.Frame()
	case interface{ StackTrace() []uintptr }:
		frame = v.StackTrace()[0]
	default:
		frame, _, _, _ = runtime.Caller(2)
	}

	fn := runtime.FuncForPC(frame) // step back to the actual call
	fullName := fn.Name()

	file, line := fn.FileLine(frame)

	pkg := packageName2(fullName)

	// funcName = fullName minus "pkg."
	funcName := fullName
	if pkg != "" && strings.HasPrefix(fullName, pkg+".") {
		funcName = fullName[len(pkg)+1:]
	}

	// Build file string: "dirname/basename:line"
	dir := filepath.Base(filepath.Dir(file))
	base := filepath.Base(file)
	// simple concat instead of Sprintf
	fileStr := "'" + dir + "/" + base + ":" + strconv.Itoa(line) + "'"

	a.Value = slog.GroupValue(
		slog.Any("payload", errVal),
		slog.String("func", funcName),
		slog.String("package", pkg),
		slog.String("file", fileStr),
	)
	return a
}
