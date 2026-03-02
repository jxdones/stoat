package statusbar

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/jxdones/stoat/internal/ui/theme"
)

const (
	minContentWidth       = 8
	horizontalPaddingCols = 2
)

// Kind is the status severity level (info, success, warning, error).
type Kind int

const (
	Info Kind = iota
	Success
	Warning
	Error
)

// Model holds the status message and level for the status bar.
type Model struct {
	text string
	kind Kind
}

// New returns a new status bar model with default " Ready" info message.
func New() Model {
	return Model{
		text: " Ready",
		kind: Info,
	}
}

// SetStatus sets the status message and level.
func (m *Model) SetStatus(text string, kind Kind) {
	m.text = text
	m.kind = kind
}

// View renders the status bar at the given width using the current theme.
func (m Model) View(width int) tea.View {
	style := lipgloss.NewStyle().Foreground(theme.Current.TextMuted)
	switch m.kind {
	case Success:
		style = style.Foreground(theme.Current.TextAccent)
	case Warning:
		style = style.Foreground(theme.Current.TextWarning)
	case Error:
		style = style.Foreground(theme.Current.TextError).Bold(true)
	}
	contentWidth := max(minContentWidth, width-horizontalPaddingCols)
	content := style.Render(ansi.Truncate(m.text, contentWidth, "…"))
	rendered := lipgloss.NewStyle().
		Width(width).
		Render(content)
	return tea.NewView(rendered)
}
