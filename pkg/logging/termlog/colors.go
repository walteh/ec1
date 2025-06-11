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

	// Tree styling colors - beautiful rainbow-ish palette for tree visualization
	TreeRootColor = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#8B5FBF", ANSI256: "97", ANSI: "5"},   // Rich purple
		Dark:  lipgloss.CompleteColor{TrueColor: "#D7AFFF", ANSI256: "183", ANSI: "13"}, // Light purple
	}
	TreeBranchColor = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#5F8787", ANSI256: "66", ANSI: "6"},   // Teal
		Dark:  lipgloss.CompleteColor{TrueColor: "#87D7D7", ANSI256: "116", ANSI: "14"}, // Light cyan
	}
	TreeKeyColor = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#D75F00", ANSI256: "166", ANSI: "3"},  // Orange
		Dark:  lipgloss.CompleteColor{TrueColor: "#FF875F", ANSI256: "209", ANSI: "11"}, // Light orange
	}
	TreeStringColor = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#5FAF5F", ANSI256: "71", ANSI: "2"},   // Green
		Dark:  lipgloss.CompleteColor{TrueColor: "#87FF87", ANSI256: "120", ANSI: "10"}, // Light green
	}
	TreeNumberColor = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#5F87FF", ANSI256: "69", ANSI: "4"},   // Blue
		Dark:  lipgloss.CompleteColor{TrueColor: "#87AFFF", ANSI256: "111", ANSI: "12"}, // Light blue
	}
	TreeBoolColor = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#FF5FD7", ANSI256: "206", ANSI: "1"}, // Magenta
		Dark:  lipgloss.CompleteColor{TrueColor: "#FFAFD7", ANSI256: "218", ANSI: "9"}, // Light magenta
	}
	TreeNullColor = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#808080", ANSI256: "244", ANSI: "8"}, // Gray
		Dark:  lipgloss.CompleteColor{TrueColor: "#BCBCBC", ANSI256: "250", ANSI: "7"}, // Light gray
	}
	TreeIndexColor = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#AF5F87", ANSI256: "132", ANSI: "5"},  // Rose
		Dark:  lipgloss.CompleteColor{TrueColor: "#D787AF", ANSI256: "175", ANSI: "13"}, // Light rose
	}
	TreeStructColor = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#875FAF", ANSI256: "97", ANSI: "5"},   // Purple
		Dark:  lipgloss.CompleteColor{TrueColor: "#AF87D7", ANSI256: "140", ANSI: "13"}, // Light purple
	}
	TreeBorderColor = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#585858", ANSI256: "240", ANSI: "8"}, // Dark gray
		Dark:  lipgloss.CompleteColor{TrueColor: "#9E9E9E", ANSI256: "247", ANSI: "7"}, // Medium gray
	}
)
