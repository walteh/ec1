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
	Value          lipgloss.Style
	ValueAppendage lipgloss.Style

	// Separator is the style for separators.
	Separator lipgloss.Style

	// Levels are the styles for each level.
	Levels map[slog.Level]lipgloss.Style

	// Keys overrides styles for specific keys.
	Keys map[string]lipgloss.Style

	// Values overrides value styles for specific keys.
	Values map[string]lipgloss.Style

	// Tree styles for tree visualizations
	Tree TreeStyles
}

type CallerStyle struct {
	File    lipgloss.Style
	Line    lipgloss.Style
	Func    lipgloss.Style
	Pkg     lipgloss.Style
	Sep     lipgloss.Style
	Project lipgloss.Style
}

// TreeStyles defines beautiful styling for tree visualizations
type TreeStyles struct {
	// Root is the style for the root node
	Root lipgloss.Style

	// Branch is the style for tree structure (├──, └──, etc.)
	Branch lipgloss.Style

	// Key is the style for object keys/field names
	Key lipgloss.Style

	// Values for different data types
	String lipgloss.Style
	Number lipgloss.Style
	Bool   lipgloss.Style
	Null   lipgloss.Style

	// Index is the style for array indices [0], [1], etc.
	Index lipgloss.Style

	// Struct is the style for struct type indicators
	Struct lipgloss.Style

	// Container is the overall style wrapper for the tree
	Container lipgloss.Style
}

// DefaultStyles returns the default styles.
func DefaultStyles() *Styles {
	return &Styles{
		Timestamp: lipgloss.NewStyle().Width(len(timeFormat)).Faint(true).Align(lipgloss.Center),
		Caller: CallerStyle{
			File:    lipgloss.NewStyle().Bold(true),
			Line:    lipgloss.NewStyle().Foreground(CallerLineColor).Faint(true),
			Func:    lipgloss.NewStyle().Foreground(CallerFuncColor),
			Pkg:     lipgloss.NewStyle().Foreground(CallerPkgColor),
			Sep:     lipgloss.NewStyle().Faint(true),
			Project: lipgloss.NewStyle().Foreground(CallerProjectColor).Bold(true),
		},
		Prefix: lipgloss.NewStyle().Bold(true).Faint(true).Width(10).Align(lipgloss.Left),
		// gray with a little bit of green
		// Prefix:         lipgloss.NewStyle().Bold(true).Faint(true).Width(10).Align(lipgloss.Left).Foreground(lipgloss.Color("#888888")),
		Message:        lipgloss.NewStyle().Foreground(MessageColor),
		Key:            lipgloss.NewStyle().Faint(true),
		Value:          lipgloss.NewStyle(),
		ValueAppendage: lipgloss.NewStyle().Faint(true).Foreground(CallerLineColor),
		Separator:      lipgloss.NewStyle().Faint(true),
		Levels: map[slog.Level]lipgloss.Style{
			slog.LevelDebug: lipgloss.NewStyle().
				SetString(strings.ToUpper(slog.LevelDebug.String())).
				Bold(true).
				MaxWidth(4).
				Align(lipgloss.Left).
				Foreground(LevelDebugColor),
			slog.LevelInfo: lipgloss.NewStyle().
				SetString(strings.ToUpper(slog.LevelInfo.String())).
				Bold(true).
				MaxWidth(4).
				Align(lipgloss.Left).
				Foreground(LevelInfoColor),
			slog.LevelWarn: lipgloss.NewStyle().
				SetString(strings.ToUpper(slog.LevelWarn.String())).
				Bold(true).
				MaxWidth(4).
				Align(lipgloss.Left).
				Foreground(LevelWarnColor),
			slog.LevelError: lipgloss.NewStyle().
				SetString(strings.ToUpper("ERR!")).
				Bold(true).
				MaxWidth(4).
				Background(LevelErrorColor).
				// Foreground(lipgloss.Color("black")).
				Align(lipgloss.Left),
		},
		Keys: map[string]lipgloss.Style{
			"error": lipgloss.NewStyle().
				Bold(true).
				Background(LevelErrorColor).
				Foreground(lipgloss.Color("black")),
		},
		Values: map[string]lipgloss.Style{
			"error": lipgloss.NewStyle().
				// Bold(true).
				Foreground(LevelErrorColor),
		},
		Tree: TreeStyles{
			Root:   lipgloss.NewStyle().Foreground(TreeRootColor).Bold(true),
			Branch: lipgloss.NewStyle().Foreground(TreeBranchColor).Faint(true),
			Key:    lipgloss.NewStyle().Foreground(TreeKeyColor).Bold(true),
			String: lipgloss.NewStyle().Foreground(TreeStringColor),
			Number: lipgloss.NewStyle().Foreground(TreeNumberColor),
			Bool:   lipgloss.NewStyle().Foreground(TreeBoolColor).Bold(true),
			Null:   lipgloss.NewStyle().Foreground(TreeNullColor).Italic(true),
			Index:  lipgloss.NewStyle().Foreground(TreeIndexColor).Bold(true),
			Struct: lipgloss.NewStyle().Foreground(TreeStructColor).Bold(true),
			Container: lipgloss.NewStyle().
				Padding(1, 2).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(TreeBorderColor),
		},
	}
}
