package termlog

import "github.com/charmbracelet/lipgloss"

// Adaptive colors used by the termlog styles. Using AdaptiveColor ensures
// sensible theming in both light and dark terminals while still honouring the
// original ANSI-256 palette codes.

// Caller related colours.
var (
	CallerLineColor = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#ffafff", ANSI256: "219", ANSI: "13"},
		Dark:  lipgloss.CompleteColor{TrueColor: "#ffafff", ANSI256: "219", ANSI: "13"},
	}
	CallerFuncColor = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#d7ff87", ANSI256: "192", ANSI: "10"},
		Dark:  lipgloss.CompleteColor{TrueColor: "#d7ff87", ANSI256: "192", ANSI: "10"},
	}
	CallerPkgColor = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#8787ff", ANSI256: "105", ANSI: "12"},
		Dark:  lipgloss.CompleteColor{TrueColor: "#8787ff", ANSI256: "105", ANSI: "12"},
	}
	CallerProjectColor = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#00ffff", ANSI256: "51", ANSI: "14"},
		Dark:  lipgloss.CompleteColor{TrueColor: "#00ffff", ANSI256: "51", ANSI: "14"},
	}

	// Level colours.
	LevelDebugColor = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#5f5fff", ANSI256: "63", ANSI: "4"},
		Dark:  lipgloss.CompleteColor{TrueColor: "#5f5fff", ANSI256: "63", ANSI: "4"},
	}
	LevelInfoColor = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#181818", ANSI256: "0", ANSI: "0"},
		Dark:  lipgloss.CompleteColor{TrueColor: "#F6F6F6", ANSI256: "15", ANSI: "15"},
	}
	LevelWarnColor = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#FFB947", ANSI256: "208", ANSI: "3"},
		Dark:  lipgloss.CompleteColor{TrueColor: "#FFB947", ANSI256: "208", ANSI: "3"},
	}
	LevelErrorColor = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#ff5f87", ANSI256: "204", ANSI: "1"},
		Dark:  lipgloss.CompleteColor{TrueColor: "#ff5f87", ANSI256: "204", ANSI: "1"},
	}

	// gray with a little bit of green
	MessageColor = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#4E613D", ANSI256: "0", ANSI: "0"},
		Dark:  lipgloss.CompleteColor{TrueColor: "#C2D9B0", ANSI256: "15", ANSI: "15"},
	}
)
