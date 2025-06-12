package termlog

import (
	"fmt"
	"log/slog"
	"runtime"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/list"
	"gitlab.com/tozd/go/errors"
)

// stackTracer interface matches the one from pkg/errors for extracting stack traces
type stackTracer interface {
	StackTrace() []uintptr
}

// frameProvider interface for getting individual error creation frames
type frameProvider interface {
	Frame() uintptr
}

// ErrorToTrace renders an error with a simple two-part structure:
// 1. Error traces list (individual errors with their locations)
// 2. Simple stack trace (just call stack, no context mapping)
func ErrorToTrace(err error, record slog.Record, styles *Styles, render renderFunc, hyperlink HyperlinkFunc) string {
	if err == nil {
		return ""
	}

	// For non-errors.E types, fall back to simple display
	if _, ok := err.(errors.E); !ok {
		return renderSimpleError(err, styles, render, record, hyperlink)
	}

	// Build the display
	var sections []string

	// Section 1: Error traces (individual errors)
	errorTraces := buildErrorTraces(err, styles, render, hyperlink)
	if len(errorTraces) > 0 {
		sections = append(sections, strings.Join(errorTraces, "\n"))
	}

	// Section 2: Simple stack trace (no context mapping)
	stackTrace := buildSimpleStackTrace(err, styles, render, hyperlink, errorTraces)
	if len(stackTrace) > 0 {
		sections = append(sections, stackTrace)
	}

	if len(sections) == 0 {
		return renderSimpleError(err, styles, render, record, hyperlink)
	}

	// Get the root error message for header (avoid repetition)
	rootError := getRootError(err)
	header := render(styles.Error.Main, rootError.Error())
	content := strings.Join(sections, "\n\n")

	return render(styles.Error.Container, header+"\n\n"+content)
}

// buildErrorTraces creates a list of individual errors with their creation locations
func buildErrorTraces(err error, styles *Styles, render renderFunc, hyperlink HyperlinkFunc) []string {
	var traces []string

	current := err
	for {
		next := errors.Unwrap(current)
		if next == nil {
			// Core error - skip it since we show it in the header
			break
		}

		// Wrapper error
		if fp, ok := current.(frameProvider); ok {
			if pc := fp.Frame(); pc != 0 {
				enhancedSource := NewEnhancedSource(pc)
				location := enhancedSource.Render(styles, render, hyperlink)

				// Extract wrapper message
				currentMsg := current.Error()
				nextMsg := next.Error()
				wrapper := ""
				if idx := strings.Index(currentMsg, nextMsg); idx > 0 {
					wrapper = strings.TrimSuffix(strings.TrimSpace(currentMsg[:idx]), ":")
				}

				if wrapper != "" {
					message := render(styles.Error.Main, wrapper)
					traces = append(traces, fmt.Sprintf("    %s: %s", location, message))
				}
			}
		}

		current = next
	}

	slices.Reverse(traces)

	return traces
}

// buildSimpleStackTrace creates a simple call stack without context mapping
func buildSimpleStackTrace(err error, styles *Styles, render renderFunc, hyperlink HyperlinkFunc, errorTraces []string) string {
	// Get stack trace from the outermost error
	if st, ok := err.(stackTracer); ok {
		frames := filterRelevantFrames(convertStackTrace(st.StackTrace()))
		if len(frames) == 0 {
			return ""
		}

		var rows []string
		// HERE:
		for _, frame := range frames {
			enhancedSource := NewEnhancedSource(frame.PC)
			locationDisplay := enhancedSource.Render(styles, render, hyperlink)
			// for _, trace := range errorTraces {
			// 	if strings.Contains(trace, locationDisplay) {
			// 		continue HERE
			// 	}
			// }
			rows = append(rows, locationDisplay)
		}

		// Dedupe consecutive identical rows
		rows = dedupeConsecutive(rows)

		// Create the list
		l := list.New(convertToListItems(rows)...).
			Enumerator(errorEnumerator).
			EnumeratorStyleFunc(errorEnumeratorStyle)

		return l.String()
	}

	return ""
}

// convertPCToFrame converts a program counter to a runtime.Frame
func convertPCToFrame(pc uintptr) runtime.Frame {
	frames := runtime.CallersFrames([]uintptr{pc})
	frame, _ := frames.Next()
	return frame
}

// dedupeConsecutive removes consecutive identical strings
func dedupeConsecutive(rows []string) []string {
	if len(rows) <= 1 {
		return rows
	}

	var result []string
	result = append(result, rows[0])

	for i := 1; i < len(rows); i++ {
		if rows[i] != rows[i-1] {
			result = append(result, rows[i])
		}
	}

	return result
}

// convertToListItems converts strings to list items
func convertToListItems(rows []string) []any {
	items := make([]any, len(rows))
	for i, row := range rows {
		items[i] = row
	}
	return items
}

// filterRelevantFrames filters out internal/uninteresting frames
func filterRelevantFrames(frames []runtime.Frame) []runtime.Frame {
	var relevant []runtime.Frame
	for _, frame := range frames {
		if isRelevantFrame(frame) {
			relevant = append(relevant, frame)
		}
	}
	return relevant
}

// errorEnumerator returns rounded tree-style connectors like Rust errors
func errorEnumerator(items list.Items, i int) string {
	if i == items.Length()-1 {
		return "╰── "
	}
	return "├── "
}

// errorEnumeratorStyle returns the style for enumerators
func errorEnumeratorStyle(items list.Items, i int) lipgloss.Style {
	return lipgloss.NewStyle()
}

// getPC extracts the program counter from a frame
func getPC(frame runtime.Frame) uintptr {
	return frame.PC
}

// isRelevantFrame determines if a frame should be included in the stack trace
func isRelevantFrame(frame runtime.Frame) bool {
	// Skip runtime internals
	if strings.HasPrefix(frame.Function, "runtime.") {
		return false
	}

	// Skip reflect internals
	if strings.HasPrefix(frame.Function, "reflect.") {
		return false
	}

	// Skip testing internals
	if strings.Contains(frame.Function, "testing.") {
		return false
	}

	// Skip empty frames
	if frame.Function == "" && frame.File == "" {
		return false
	}

	return true
}

// convertStackTrace converts a slice of PCs to runtime.Frames
func convertStackTrace(st []uintptr) []runtime.Frame {
	var frames []runtime.Frame

	callersFrames := runtime.CallersFrames(st)
	for {
		frame, more := callersFrames.Next()
		frames = append(frames, frame)
		if !more {
			break
		}
	}

	return frames
}

// getRootError finds the innermost (root) error in the chain
func getRootError(err error) error {
	current := err
	for {
		next := errors.Unwrap(current)
		if next == nil {
			return current
		}
		current = next
	}
}

// renderSimpleError renders a simple error with optional stack trace from log record
func renderSimpleError(err error, styles *Styles, render renderFunc, record slog.Record, hyperlink HyperlinkFunc) string {
	// Start with the error message
	header := render(styles.Error.Main, err.Error())

	// If we have source information from the log record, add it as a simple stack trace
	if record.PC != 0 {
		enhancedSource := NewEnhancedSource(record.PC)
		location := enhancedSource.Render(styles, render, hyperlink)

		// Create a simple single-item list with the location
		l := list.New(location).
			Enumerator(errorEnumerator).
			EnumeratorStyleFunc(errorEnumeratorStyle)

		content := l.String()
		return render(styles.Error.Container, header+"\n\n"+content)
	}

	// Fallback to just the error message if no source info
	return render(styles.Error.Main, err.Error())
}
