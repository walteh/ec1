package termlog

import (
	"log/slog"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Styles defines the styles for the text logger.
type Styles struct {
	// Timestamp is the style for timestamps.
	Timestamp lipgloss.Style

	// Caller is the style for source caller.
	Caller CallerStyle

	// Prefix is the style for prefix.
	Prefix lipgloss.Style

	// Message is the style for messages.
	Message lipgloss.Style

	// Key is the style for keys.
	Key lipgloss.Style

	// Value is the style for values.
	Value lipgloss.Style

	// Separator is the style for separators.
	Separator lipgloss.Style

	// Levels are the styles for each level.
	Levels map[slog.Level]lipgloss.Style

	// Keys overrides styles for specific keys.
	Keys map[string]lipgloss.Style

	// Values overrides value styles for specific keys.
	Values map[string]lipgloss.Style
}

type CallerStyle struct {
	File    lipgloss.Style
	Line    lipgloss.Style
	Func    lipgloss.Style
	Pkg     lipgloss.Style
	Sep     lipgloss.Style
	Project lipgloss.Style
}

// DefaultStyles returns the default styles.
func DefaultStyles() *Styles {
	return &Styles{
		Timestamp: lipgloss.NewStyle().Width(len(timeFormat)).Faint(true).Align(lipgloss.Center),
		Caller: CallerStyle{
			File:    lipgloss.NewStyle().Bold(true),
			Line:    lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(219)).Faint(true),
			Func:    lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(192)),
			Pkg:     lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(105)),
			Sep:     lipgloss.NewStyle().Faint(true),
			Project: lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(51)).Bold(true),
		},
		Prefix:    lipgloss.NewStyle().Bold(true).Faint(true).Width(10),
		Message:   lipgloss.NewStyle(),
		Key:       lipgloss.NewStyle().Faint(true),
		Value:     lipgloss.NewStyle(),
		Separator: lipgloss.NewStyle().Faint(true),
		Levels: map[slog.Level]lipgloss.Style{
			slog.LevelDebug: lipgloss.NewStyle().
				SetString(strings.ToUpper(slog.LevelDebug.String())).
				Bold(true).
				MaxWidth(4).
				Foreground(lipgloss.Color("63")),
			slog.LevelInfo: lipgloss.NewStyle().
				SetString(strings.ToUpper(slog.LevelInfo.String())).
				Bold(true).
				MaxWidth(4).
				Foreground(lipgloss.Color("86")),
			slog.LevelWarn: lipgloss.NewStyle().
				SetString(strings.ToUpper(slog.LevelWarn.String())).
				Bold(true).
				MaxWidth(4).
				// orange
				Foreground(lipgloss.Color("208")),
			slog.LevelError: lipgloss.NewStyle().
				SetString(strings.ToUpper("ERR!")).
				Bold(true).
				MaxWidth(4).
				Foreground(lipgloss.Color("204")).
				Padding(0, 1, 0, 1),
			// Background(lipgloss.Color("204")).
			// Foreground(lipgloss.Color("0")),
		},
		Keys:   map[string]lipgloss.Style{},
		Values: map[string]lipgloss.Style{},
	}
}
