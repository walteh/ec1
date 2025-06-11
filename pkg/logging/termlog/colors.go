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
		Light: lipgloss.CompleteColor{TrueColor: "#BBBBF9", ANSI256: "105", ANSI: "12"},
		Dark:  lipgloss.CompleteColor{TrueColor: "#BBBBF9", ANSI256: "105", ANSI: "12"},
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
		Light: lipgloss.CompleteColor{TrueColor: "#E100FF", ANSI256: "165", ANSI: "5"},  // Electric magenta
		Dark:  lipgloss.CompleteColor{TrueColor: "#FF69B4", ANSI256: "205", ANSI: "13"}, // Hot pink
	}
	TreeBranchColor = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#00D4AA", ANSI256: "43", ANSI: "6"},  // Electric teal
		Dark:  lipgloss.CompleteColor{TrueColor: "#40E0D0", ANSI256: "80", ANSI: "14"}, // Turquoise
	}
	TreeKeyColor = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#FF6600", ANSI256: "202", ANSI: "3"},  // Electric orange
		Dark:  lipgloss.CompleteColor{TrueColor: "#FF8C00", ANSI256: "208", ANSI: "11"}, // Dark orange
	}
	TreeStringColor = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#00FF41", ANSI256: "46", ANSI: "2"},  // Matrix green
		Dark:  lipgloss.CompleteColor{TrueColor: "#39FF14", ANSI256: "82", ANSI: "10"}, // Neon green
	}
	TreeNumberColor = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#0080FF", ANSI256: "33", ANSI: "4"},  // Electric blue
		Dark:  lipgloss.CompleteColor{TrueColor: "#00BFFF", ANSI256: "39", ANSI: "12"}, // Deep sky blue
	}
	TreeBoolColor = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#FF1493", ANSI256: "198", ANSI: "1"}, // Deep pink
		Dark:  lipgloss.CompleteColor{TrueColor: "#FF69B4", ANSI256: "205", ANSI: "9"}, // Hot pink
	}
	TreeNullColor = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#6C6C6C", ANSI256: "242", ANSI: "8"}, // Darker gray
		Dark:  lipgloss.CompleteColor{TrueColor: "#A8A8A8", ANSI256: "248", ANSI: "7"}, // Light gray
	}
	TreeIndexColor = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#FF0080", ANSI256: "198", ANSI: "5"},  // Electric pink
		Dark:  lipgloss.CompleteColor{TrueColor: "#FF007F", ANSI256: "198", ANSI: "13"}, // Rose
	}
	TreeStructColor = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#8A2BE2", ANSI256: "92", ANSI: "5"},  // Blue violet
		Dark:  lipgloss.CompleteColor{TrueColor: "#9370DB", ANSI256: "98", ANSI: "13"}, // Medium slate blue
	}
	TreeBorderColor = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#4169E1", ANSI256: "62", ANSI: "8"}, // Royal blue
		Dark:  lipgloss.CompleteColor{TrueColor: "#6495ED", ANSI256: "68", ANSI: "7"}, // Cornflower blue
	}

	// Error trace styling colors - Rust-inspired error display
	ErrorMainColor = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#FF0040", ANSI256: "196", ANSI: "1"}, // Electric red
		Dark:  lipgloss.CompleteColor{TrueColor: "#FF4444", ANSI256: "203", ANSI: "9"}, // Bright red
	}
	ErrorTraceArrowColor = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#FF4500", ANSI256: "196", ANSI: "3"},  // Orange red
		Dark:  lipgloss.CompleteColor{TrueColor: "#FF6347", ANSI256: "203", ANSI: "11"}, // Tomato
	}
	ErrorFunctionColor = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#9932CC", ANSI256: "92", ANSI: "5"},   // Dark orchid
		Dark:  lipgloss.CompleteColor{TrueColor: "#DA70D6", ANSI256: "170", ANSI: "13"}, // Orchid
	}
	ErrorPackageColor = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#0066FF", ANSI256: "27", ANSI: "4"},  // Electric blue
		Dark:  lipgloss.CompleteColor{TrueColor: "#1E90FF", ANSI256: "33", ANSI: "12"}, // Dodger blue
	}
	ErrorFileColor = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#00AA00", ANSI256: "34", ANSI: "2"},  // Electric green
		Dark:  lipgloss.CompleteColor{TrueColor: "#32CD32", ANSI256: "76", ANSI: "10"}, // Lime green
	}
	ErrorLineColor = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#FFD700", ANSI256: "220", ANSI: "3"},  // Gold
		Dark:  lipgloss.CompleteColor{TrueColor: "#FFFF00", ANSI256: "226", ANSI: "11"}, // Electric yellow
	}
	ErrorContextColor = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#708090", ANSI256: "244", ANSI: "8"}, // Slate gray
		Dark:  lipgloss.CompleteColor{TrueColor: "#C0C0C0", ANSI256: "250", ANSI: "7"}, // Silver
	}
	ErrorBorderColor = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#DC143C", ANSI256: "161", ANSI: "1"}, // Crimson
		Dark:  lipgloss.CompleteColor{TrueColor: "#FF1493", ANSI256: "198", ANSI: "9"}, // Deep pink
	}
)
