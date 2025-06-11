package termlog

import (
	"fmt"
	"runtime"
	"strings"

	"gitlab.com/tozd/go/errors"
)

// ErrorToTrace converts an errors.E to a beautifully styled Rust-inspired error trace.
func ErrorToTrace(err error, styles *Styles, render renderFunc) string {
	if err == nil {
		return ""
	}

	// Check if it's our enhanced error type
	terr, ok := err.(errors.E)
	if !ok {
		// Fallback for regular errors
		return renderSimpleError(err, styles, render)
	}

	var b strings.Builder

	// Main error message with prominent styling
	b.WriteString(render(styles.Error.Main, "Error: "+terr.Error()) + "\n")

	// Get stack trace frames
	frames := runtime.CallersFrames(terr.StackTrace())
	var traceFrames []runtime.Frame

	// Collect frames (limit to avoid overwhelming output)
	maxFrames := 8
	for i := 0; i < maxFrames; i++ {
		frame, more := frames.Next()
		if !more {
			break
		}
		traceFrames = append(traceFrames, frame)
	}

	// Build the trace display
	if len(traceFrames) > 0 {
		b.WriteString("\n")
		for i, frame := range traceFrames {
			isLast := i == len(traceFrames)-1
			buildErrorFrame(&b, frame, isLast, styles, render)
		}
	}

	// Wrap in container style
	return render(styles.Error.Container, b.String())
}

// renderSimpleError handles regular Go errors without stack traces
func renderSimpleError(err error, styles *Styles, render renderFunc) string {
	return render(styles.Error.Container,
		render(styles.Error.Main, "Error: "+err.Error()))
}

// buildErrorFrame builds a single stack frame in the trace
func buildErrorFrame(b *strings.Builder, frame runtime.Frame, isLast bool, styles *Styles, render renderFunc) {
	// Choose connector style
	connector := "├─"
	indent := "│ "
	if isLast {
		connector = "└─"
		indent = "  "
	}

	// Extract package name from function
	pkg := packageFromFunction(frame.Function)
	funcName := functionFromFunction(frame.Function)

	// Build the frame line
	arrow := render(styles.Error.Arrow, connector)
	at := render(styles.Error.Context, " at ")
	function := render(styles.Error.Function, funcName)
	inPkg := render(styles.Error.Context, " in ")
	packageName := render(styles.Error.Package, pkg)

	b.WriteString(arrow + at + function + inPkg + packageName + "\n")

	// File location with hyperlink
	fileIndent := render(styles.Error.Arrow, indent)
	locationArrow := render(styles.Error.Arrow, "└─ ")

	// Create clickable file path
	lineNum := fmt.Sprintf("%d", frame.Line)

	// Create hyperlink (VS Code/Cursor compatible)
	hyperlink := createFileHyperlink(frame.File, frame.Line)
	filePart := render(styles.Error.File, hyperlink)
	linePart := render(styles.Error.Line, ":"+lineNum)

	b.WriteString(fileIndent + locationArrow + filePart + linePart + "\n")
}

// createFileHyperlink creates a clickable hyperlink for the file location
func createFileHyperlink(filePath string, line int) string {
	// Format: 'file:///absolute/path/to/file.go:123'
	// The quotes help VS Code/Cursor recognize it as clickable
	return fmt.Sprintf("'%s:%d'", filePath, line)
}

// packageFromFunction extracts package name from a full function name
func packageFromFunction(fullFunc string) string {
	// Handle cases like "github.com/walteh/ec1/pkg/vmm.(*Manager).Start"
	parts := strings.Split(fullFunc, ".")
	if len(parts) < 2 {
		return "unknown"
	}

	// Find the last part that looks like a package (before method/function)
	for i := len(parts) - 2; i >= 0; i-- {
		part := parts[i]
		// Skip method receivers like "(*Manager)"
		if !strings.Contains(part, "(") && !strings.Contains(part, ")") {
			// Take package path parts
			pkgParts := parts[:i+1]
			pkg := strings.Join(pkgParts, ".")

			// Simplify long package paths to just the last few parts
			return simplifyPackageName(pkg)
		}
	}

	return "unknown"
}

// functionFromFunction extracts function/method name from a full function name
func functionFromFunction(fullFunc string) string {
	// Handle cases like "github.com/walteh/ec1/pkg/vmm.(*Manager).Start"
	parts := strings.Split(fullFunc, ".")
	if len(parts) == 0 {
		return "unknown"
	}

	// Get the last part (function/method name)
	funcName := parts[len(parts)-1]

	// Handle method receivers - look for the actual method name
	if len(parts) >= 2 {
		prevPart := parts[len(parts)-2]
		if strings.Contains(prevPart, "(") && strings.Contains(prevPart, ")") {
			// This is a method call, use the last part
			return funcName
		}
	}

	return funcName
}

// simplifyPackageName shortens long package paths for readability
func simplifyPackageName(pkg string) string {
	parts := strings.Split(pkg, "/")
	if len(parts) <= 2 {
		return pkg
	}

	// Show last 2-3 parts of the package path
	if len(parts) >= 3 {
		return strings.Join(parts[len(parts)-2:], "/")
	}

	return pkg
}
