package connectionpicker

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/jxdones/stoat/internal/config"
	"github.com/jxdones/stoat/internal/ui/keys"
	"github.com/jxdones/stoat/internal/ui/modal"
	"github.com/jxdones/stoat/internal/ui/theme"
)

const modalWidth = 50

type Event int

const (
	EventNone Event = iota
	EventSelected
	EventClosed
)

// Model is the connection picker component.
type Model struct {
	connections []config.Connection
	selected    int
}

// New creates a new connection picker model.
func New() Model {
	return Model{
		connections: []config.Connection{},
		selected:    0,
	}
}

// Selected returns the selected connection.
func (m Model) Selected() config.Connection {
	if len(m.connections) == 0 {
		return config.Connection{}
	}
	return m.connections[m.selected]
}

// SetConnections sets the connections.
func (m *Model) SetConnections(connections []config.Connection) {
	m.connections = connections
}

// Connection returns the connection with the given name.
func (m Model) ConnectionByName(name string) (config.Connection, bool) {
	for _, conn := range m.connections {
		if conn.Name == name {
			return conn, true
		}
	}
	return config.Connection{}, false
}

// Update handles the key press message and returns the updated model and event.
func (m Model) Update(msg tea.KeyPressMsg) (Model, Event) {
	switch {
	case key.Matches(msg, keys.Default.MoveUp):
		if m.selected > 0 {
			m.selected--
		}
	case key.Matches(msg, keys.Default.MoveDown):
		if m.selected < len(m.connections)-1 {
			m.selected++
		}
	case key.Matches(msg, keys.Default.Enter):
		return m, EventSelected
	case key.Matches(msg, keys.Default.Escape):
		return m, EventClosed
	}
	return m, EventNone
}

// View returns the fully rendered modal for the connection picker.
func (m Model) View() tea.View {
	lines := make([]string, len(m.connections))
	for i, conn := range m.connections {
		if i == m.selected {
			lines[i] = lipgloss.NewStyle().Foreground(theme.Current.TextAccent).Bold(true).Render("> " + conn.Name)
		} else {
			lines[i] = lipgloss.NewStyle().Foreground(theme.Current.TextPrimary).Render("  " + conn.Name)
		}
	}
	content := strings.Join(lines, "\n")
	return tea.NewView(modal.Render("Connections", content, "j/k navigate · enter select · esc close", modalWidth))
}
