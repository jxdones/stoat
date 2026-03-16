package connectionpicker

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/jxdones/stoat/internal/config"
	"github.com/jxdones/stoat/internal/ui/keys"
	"github.com/jxdones/stoat/internal/ui/theme"
)

type Event int

const (
	EventNone Event = iota
	EventSelected
	EventClosed
)

type Model struct {
	connections []config.Connection
	selected    int
}

func New() Model {
	return Model{
		connections: []config.Connection{},
		selected:    0,
	}
}

func (m Model) Selected() config.Connection {
	if len(m.connections) == 0 {
		return config.Connection{}
	}
	return m.connections[m.selected]
}

func (m *Model) SetConnections(connections []config.Connection) {
	m.connections = connections
}

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

func (m Model) View() string {
	lines := make([]string, len(m.connections))
	for i, conn := range m.connections {
		if i == m.selected {
			lines[i] = lipgloss.NewStyle().Foreground(theme.Current.TextAccent).Render("> " + conn.Name)
		} else {
			lines[i] = lipgloss.NewStyle().Foreground(theme.Current.TextPrimary).Render("  " + conn.Name)
		}
	}
	return strings.Join(lines, "\n")
}
