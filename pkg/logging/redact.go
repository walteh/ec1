package logging

import (
	"context"
	"log/slog"
	"slices"
	"strings"
	"sync"
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

func RegisterRedactedLogValue(ctx context.Context, key string, value string) {
	l := slog.Default().WithGroup("redactor")
	l.DebugContext(ctx, "registering redacted log value", "key", key, "value", value)

	redactedLogValuesMutex.Lock()
	defer redactedLogValuesMutex.Unlock()
	redactedLogValues = append(redactedLogValues, RedactedKey{Key: key, Value: value})
	go func() {
		<-ctx.Done()
		redactedLogValuesMutex.Lock()
		defer redactedLogValuesMutex.Unlock()
		redactedLogValues = slices.DeleteFunc(redactedLogValues, func(v RedactedKey) bool {
			return v.Key == key
		})
	}()
}
