package termlog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/walteh/ec1/pkg/logging/valuelog"
)

var _ slog.Handler = (*TermLogger)(nil)

type TermLoggerOption = func(*TermLogger)

func WithStyles(styles *Styles) TermLoggerOption {
	return func(l *TermLogger) {
		l.styles = styles
	}
}

func WithLoggerName(name string) TermLoggerOption {
	return func(l *TermLogger) {
		l.name = name
	}
}

func WithProfile(profile termenv.Profile) TermLoggerOption {
	return func(l *TermLogger) {
		l.renderOpts = append(l.renderOpts, termenv.WithProfile(profile))
	}
}

func WithRenderOption(opt termenv.OutputOption) TermLoggerOption {
	return func(l *TermLogger) {
		l.renderOpts = append(l.renderOpts, opt)
	}
}

type TermLogger struct {
	slogOptions *slog.HandlerOptions
	styles      *Styles
	writer      io.Writer
	renderOpts  []termenv.OutputOption
	renderer    *lipgloss.Renderer
	name        string
}

func NewTermLogger(writer io.Writer, sopts *slog.HandlerOptions, opts ...TermLoggerOption) *TermLogger {
	l := &TermLogger{
		writer:      writer,
		slogOptions: sopts,
		styles:      DefaultStyles(),
		renderOpts:  []termenv.OutputOption{},
		name:        "",
	}
	for _, opt := range opts {
		opt(l)
	}

	l.renderer = lipgloss.NewRenderer(l.writer, l.renderOpts...)

	return l
}

// Enabled implements slog.Handler.
func (l *TermLogger) Enabled(ctx context.Context, level slog.Level) bool {
	return l.slogOptions.Level.Level() <= level.Level()
}

func (l *TermLogger) render(s lipgloss.Style, strs ...string) string {
	return s.Renderer(l.renderer).Render(strs...)
}

func (l *TermLogger) renderFunc(s lipgloss.Style, strs string) string {
	return s.Renderer(l.renderer).Render(strs)
}

const (
	timeFormat    = "15:04:05.0000 MST"
	maxNameLength = 10
)

func (l *TermLogger) Handle(ctx context.Context, r slog.Record) error {
	// Build a pretty, human-friendly log line.

	var b strings.Builder
	var appendageBuilder strings.Builder

	// 0. Name.
	if l.name != "" {
		b.WriteString(l.render(l.styles.Prefix, strings.ToUpper(l.name)))
		b.WriteByte(' ')
	}

	// 1 Level.
	levelStyle, ok := l.styles.Levels[r.Level]
	if !ok {
		levelStyle = l.styles.Levels[slog.LevelInfo]
	}
	b.WriteString(l.render(levelStyle, r.Level.String()))
	b.WriteByte(' ')

	// 2. Timestamp.
	ts := r.Time
	if ts.IsZero() {
		ts = time.Now()
	}

	b.WriteString(l.render(l.styles.Timestamp, ts.Format(timeFormat)))
	b.WriteByte(' ')

	// 3. Source (if requested).
	if l.slogOptions != nil && l.slogOptions.AddSource {
		b.WriteString(NewEnhancedSource(r.PC).Render(l.renderer, l.styles))
		b.WriteByte(' ')
	}

	// 4. Message.
	msg := l.styles.Message.Render(r.Message)
	b.WriteString(msg)

	// 5. Attributes (key=value ...).
	if r.NumAttrs() > 0 {
		b.WriteByte(' ')
		r.Attrs(func(a slog.Attr) bool {
			// Key styling (supports per-key overrides).
			keyStyle, ok := l.styles.Keys[a.Key]
			if !ok {
				keyStyle = l.styles.Key
			}

			key := l.render(keyStyle, a.Key)

			var valColored string

			if pv, ok := a.Value.Any().(valuelog.PrettyRawJSONValue); ok {
				appendageBuilder.WriteString("\n" + a.Key + ":\n")
				appendageBuilder.WriteString(JSONToTree(a.Key, pv.RawJSON(), l.styles, l.renderFunc))
				appendageBuilder.WriteByte('\n')
				valColored = l.render(l.styles.ValueAppendage, "[json rendered below]")
			} else if pv, ok := a.Value.Any().(valuelog.PrettyAnyValue); ok {
				appendageBuilder.WriteString("\n" + a.Key + ":\n")
				appendageBuilder.WriteString(StructToTree(pv.Any(), l.styles, l.renderFunc))
				appendageBuilder.WriteByte('\n')
				valColored = l.render(l.styles.ValueAppendage, "[struct rendered below]")
			} else {
				// Value styling (supports per-key overrides).
				valStyle, ok := l.styles.Values[a.Key]
				if !ok {
					valStyle = l.styles.Value
				}

				// Resolve slog.Value to an interface{} and stringify.
				val := fmt.Sprint(a.Value)
				valColored = l.render(valStyle, val)

			}

			b.WriteString(key)
			b.WriteByte('=')
			b.WriteString(valColored)
			b.WriteByte(' ')

			return true
		})
		// Remove trailing space that was added inside the loop.
		if b.Len() > 0 && b.String()[b.Len()-1] == ' ' {
			str := b.String()
			b.Reset()
			b.WriteString(strings.TrimRight(str, " "))
		}
	}

	// 6. Final newline.
	b.WriteByte('\n')

	// Determine output writer (defaults to stdout).
	w := l.writer
	if w == nil {
		w = os.Stdout
	}

	_, err := fmt.Fprint(w, b.String())
	if appendageBuilder.Len() > 0 {
		_, err = fmt.Fprint(w, appendageBuilder.String())
	}
	return err
}

func (l *TermLogger) WithAttrs(attrs []slog.Attr) slog.Handler {
	return l
}

func (l *TermLogger) WithGroup(name string) slog.Handler {
	return l
}
