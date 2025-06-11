package termlog

import (
	"bytes"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
	"testing"

	gerrs "errors"

	"gitlab.com/tozd/go/errors"
)

func logCapturingOutput(t *testing.T, err error, opts slog.HandlerOptions) string {
	buf := bytes.NewBufferString("")
	handler := NewTermLogger(buf, &opts, WithStyles(EmptyStyles()), WithHyperlinkFunc(func(link, renderedText string) string {
		return renderedText
	}))

	logger := slog.New(handler)

	logger.Error("test", "error", err)

	fmt.Println(buf.String())

	return buf.String()
}

func logErrorLocation(t *testing.T, err error, opts slog.HandlerOptions) string {
	if e, ok := err.(interface{ Frame() uintptr }); ok {
		frames := runtime.CallersFrames([]uintptr{e.Frame()})
		frame, _ := frames.Next()
		return fmt.Sprintf("%s:%d", frame.File, frame.Line)
	}
	return ""
}

func TestErrorDisplay(t *testing.T) {
	opts := slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	}

	err1 := gerrs.New("test error 1")

	nestedFunc2 := func() error {
		return err1
	}
	err2 := errors.Errorf("test error 2 %w", nestedFunc2())
	err3 := errors.Errorf("test error 3 %w", err2)
	err4 := errors.Errorf("test error 4 %w", err3)
	err5 := errors.Errorf("test error 5 %w", err4)
	err6 := errors.Errorf("test error 6 %w", err5)
	nestedFunc := func() error {
		return errors.Errorf("nested function error %w", err6)
	}
	err7 := nestedFunc()

	logger := logCapturingOutput(t, err7, opts)

	errd := []error{err1, err2, err3, err4, err5, err6, err7}

	for _, err := range errd {
		fmt.Println(logErrorLocation(t, err, opts))
	}

	// Test the simple format: individual error traces + simple stack trace
	// Error traces should show individual error creation locations
	expectedErrorTraces := []string{
		"test error 2", "test error 3", "test error 4",
		"test error 5", "test error 6", "nested function error",
	}

	// Count how many error traces we found
	errorTraceCount := 0
	for _, expected := range expectedErrorTraces {
		if strings.Contains(logger, expected) {
			errorTraceCount++
		}
	}

	// Validate we have most of the expected error traces
	if errorTraceCount < 3 {
		t.Errorf("expected at least 3 error traces but found %d", errorTraceCount)
	}

	// Validate we have a stack trace (simple, no contexts)
	stackTraceCount := strings.Count(logger, "    ")
	if stackTraceCount == 0 {
		t.Errorf("expected some stack trace frames but found none")
	}

	t.Logf("✅ Simple format working: %d error traces, %d stack frames", errorTraceCount, stackTraceCount)

	// Optional: test for hyperlinks if needed
	// hyperlinkCount := strings.Count(logger, "\x1b]8;;")
	// t.Logf("Found %d hyperlinks", hyperlinkCount)
}
