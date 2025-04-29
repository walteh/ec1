package testutils

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/lmittmann/tint"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	slogctx "github.com/veqryn/slog-context"
	"github.com/walteh/ec1/pkg/machines/host"
)

type RedactedKey struct {
	Key   string
	Value string
}

// use array to preserve order
var redactedLogValues = make([]RedactedKey, 0)
var redactedLogValuesMutex = &sync.Mutex{}

func Redact(groups []string, a slog.Attr) slog.Attr {
	if slices.Contains(groups, "test-redactor") {
		return a
	}
	redactedLogValuesMutex.Lock()
	reversed := slices.Clone(redactedLogValues)
	redactedLogValuesMutex.Unlock()
	slices.Reverse(reversed)
	for _, value := range reversed {
		if strings.Contains(a.Value.String(), value.Key) {
			a = slog.Attr{Key: a.Key, Value: slog.StringValue(strings.ReplaceAll(a.Value.String(), value.Key, value.Value))}
		}
	}
	return a
}

func RegisterRedactedLogValue(t *testing.T, key string, value string) {
	l := slog.Default().WithGroup("test-redactor")
	l.DebugContext(t.Context(), "registering redacted log value", "key", key, "value", value)

	redactedLogValuesMutex.Lock()
	defer redactedLogValuesMutex.Unlock()
	redactedLogValues = append(redactedLogValues, RedactedKey{Key: key, Value: value})
	t.Cleanup(func() {
		redactedLogValuesMutex.Lock()
		defer redactedLogValuesMutex.Unlock()
		redactedLogValues = slices.DeleteFunc(redactedLogValues, func(v RedactedKey) bool {
			return v.Key == key
		})
	})
}

func SetupSlog(t *testing.T, ctx context.Context) context.Context {

	cached, err := host.CacheDirPrefix()
	require.NoError(t, err)

	tintHandler := tint.NewHandler(os.Stdout, &tint.Options{
		Level:      slog.LevelDebug,
		TimeFormat: "2006-01-02 15:04 05.0000",
		AddSource:  true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			return Redact(groups, a)
		},
	})

	ctxHandler := slogctx.NewHandler(tintHandler, nil)

	mylogger := slog.New(ctxHandler)
	slog.SetDefault(mylogger)

	// point logrus at our slog
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	RegisterRedactedLogValue(t, os.TempDir()+"/", "[os-tmp-dir]")
	RegisterRedactedLogValue(t, cached, "[vm-cache-dir]")
	RegisterRedactedLogValue(t, filepath.Dir(t.TempDir()), "[test-tmp-dir]") // higher priority than os-tmp-dir

	return slogctx.NewCtx(ctx, mylogger)
}

// LOGRUS

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

type SlogRawJSONValue struct {
	rawJson json.RawMessage
}

var _ slog.LogValuer = &SlogRawJSONValue{}

func (s SlogRawJSONValue) LogValue() slog.Value {
	if s.rawJson == nil {
		return slog.AnyValue(nil)
	}
	var v any
	err := json.Unmarshal(s.rawJson, &v)
	if err != nil {
		return slog.StringValue(string(s.rawJson))
	}
	return slog.AnyValue(v)
}
