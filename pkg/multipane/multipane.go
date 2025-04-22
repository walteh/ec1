package multipane

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styles for the UI components
var (
	// Base styles
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}

	// Title style for panes
	titleStyle = lipgloss.NewStyle().
			Foreground(highlight).
			Bold(true).
			Padding(0, 1)

	// Border style for panes
	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(subtle).
			Padding(0, 0, 0, 1)

	// Error style
	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true)

	// Status bar style
	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#FF5F87")).
			Padding(0, 1)

	// Help style
	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			Italic(true)
)

// pane wraps a viewport for displaying reader output
type pane struct {
	title  string
	vp     viewport.Model
	active bool
}

// Model holds all active panes
type Model struct {
	panes         []pane
	activePane    int
	width         int
	height        int
	ready         bool
	showTimestamp bool
}

// internal message to add a pane
type addPaneMsg struct {
	name   string
	reader io.Reader
}

// timestamp message
type tickMsg time.Time

// global program instance; Initialized in Run
var program *tea.Program

// NewModel creates an initial empty model
func NewModel() Model {
	return Model{
		panes:         []pane{},
		activePane:    0,
		showTimestamp: true,
	}
}

// AddPane sends a request to add a new pane from the provided reader
// Safe to call at any time before or during Run
func AddPane(name string, r io.Reader) {
	if program == nil {
		log.Println("multipane: program not started; call Run() first")
		return
	}
	program.Send(addPaneMsg{name: name, reader: r})
}

// Run starts the bubbletea program. Call this once.
func Run(ctx context.Context, in io.Reader, out io.Writer) error {
	m := NewModel()
	// create a Bubble Tea program with a 100ms tick for cleanup
	program = tea.NewProgram(m,
		tea.WithOutput(out),
		tea.WithInput(in),
		tea.WithContext(ctx),
		tea.WithFPS(10),
		tea.WithAltScreen(),
	)
	_, err := program.Run()
	if err != nil {
		return err
	}
	return nil
}

// Init is part of tea.Model; starts a ticker for updating time
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
	)
}

// tickCmd produces a tick every second for updating timestamps
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// formatContent adds timestamps if enabled
func (m Model) formatContent(content string) string {
	if !m.showTimestamp {
		return content
	}

	var formattedLines []string
	lines := strings.Split(content, "\n")
	timestamp := time.Now().Format("15:04:05")

	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			if strings.Contains(line, "Error") || strings.Contains(line, "error") ||
				strings.Contains(line, "failed") || strings.Contains(line, "Failed") {
				formattedLines = append(formattedLines, errorStyle.Render(fmt.Sprintf("[%s] %s", timestamp, line)))
			} else {
				formattedLines = append(formattedLines, fmt.Sprintf("[%s] %s", timestamp, line))
			}
		} else {
			formattedLines = append(formattedLines, line)
		}
	}

	return strings.Join(formattedLines, "\n")
}

// Update handles incoming messages: adding panes and global events
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case addPaneMsg:
		// read the entire reader into buffer
		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, msg.reader); err != nil {
			log.Printf("multipane: failed to read pane: %v", err)
		}

		// create viewport sized later in View
		v := viewport.Model{}
		content := m.formatContent(buf.String())
		v.SetContent(content)

		newPane := pane{
			title:  msg.name,
			vp:     v,
			active: len(m.panes) == 0, // First pane is active by default
		}

		m.panes = append(m.panes, newPane)

		// Recalculate sizes for all panes
		if m.ready {
			m = m.resizePanes()
		}

		return m, nil

	case tickMsg:
		// Tick updates for timestamps, etc.
		return m, tickCmd()

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ready = true
		m = m.resizePanes()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			// Scroll up in active pane
			if len(m.panes) > 0 {
				var cmd tea.Cmd
				m.panes[m.activePane].vp, cmd = m.panes[m.activePane].vp.Update(msg)
				return m, cmd
			}
		case "down", "j":
			// Scroll down in active pane
			if len(m.panes) > 0 {
				var cmd tea.Cmd
				m.panes[m.activePane].vp, cmd = m.panes[m.activePane].vp.Update(msg)
				return m, cmd
			}
		case "tab", "right", "l":
			// Move to next pane
			if len(m.panes) > 0 {
				m.panes[m.activePane].active = false
				m.activePane = (m.activePane + 1) % len(m.panes)
				m.panes[m.activePane].active = true
			}
		case "shift+tab", "left", "h":
			// Move to previous pane
			if len(m.panes) > 0 {
				m.panes[m.activePane].active = false
				m.activePane = (m.activePane - 1 + len(m.panes)) % len(m.panes)
				m.panes[m.activePane].active = true
			}
		case "t":
			// Toggle timestamps
			m.showTimestamp = !m.showTimestamp
		}
	}
	return m, nil
}

// resizePanes calculates and sets the dimensions for each pane
func (m Model) resizePanes() Model {
	if len(m.panes) == 0 || !m.ready {
		return m
	}

	// Reserve space for the help text at the bottom
	availableHeight := m.height - 2

	paneHeight := availableHeight / len(m.panes)

	// Make sure each pane gets at least a minimum height
	if paneHeight < 5 {
		paneHeight = 5
	}

	for i := range m.panes {
		m.panes[i].vp.Width = m.width - 4     // Account for borders
		m.panes[i].vp.Height = paneHeight - 3 // Account for title and borders
	}

	return m
}

// View renders all panes stacked vertically
func (m Model) View() string {
	if !m.ready || len(m.panes) == 0 {
		return "Initializing..."
	}

	var out strings.Builder

	for i, p := range m.panes {
		// Create a border style based on whether the pane is active
		style := borderStyle
		if p.active {
			style = style.BorderForeground(highlight)
		}

		// Render the title with the appropriate style
		title := titleStyle.Render(p.title)
		if p.active {
			title = titleStyle.Copy().
				Foreground(special).
				Underline(true).
				Render(p.title + " (active)")
		}

		// Combine the title and content
		content := fmt.Sprintf("%s\n%s", title, p.vp.View())

		// Apply the border style to the entire pane
		renderedPane := style.Render(content)

		out.WriteString(renderedPane)

		// Add a separator between panes, except for the last one
		if i < len(m.panes)-1 {
			out.WriteString("\n")
		}
	}

	// Add the help text at the bottom
	helpText := helpStyle.Render("Press [↑/↓] to scroll • [Tab] to switch panes • [t] toggle timestamps • [q] to quit")
	statusBar := statusStyle.Width(m.width).Render("EC1 Control Panel")

	out.WriteString("\n" + statusBar + "\n" + helpText)

	return out.String()
}
