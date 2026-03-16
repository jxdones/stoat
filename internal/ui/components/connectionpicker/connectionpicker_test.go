package connectionpicker

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/jxdones/stoat/internal/config"
)

var testConnections = []config.Connection{
	{Name: "local-sqlite", Type: "sqlite", DSN: "/tmp/test.db"},
	{Name: "prod-postgres", Type: "postgres", DSN: "postgres://localhost/prod"},
	{Name: "staging-postgres", Type: "postgres", DSN: "postgres://localhost/staging"},
}

func keyMsg(key string) tea.KeyPressMsg {
	switch key {
	case "j", "down":
		return tea.KeyPressMsg(tea.Key{Code: 'j', Text: "j"})
	case "k", "up":
		return tea.KeyPressMsg(tea.Key{Code: 'k', Text: "k"})
	case "enter":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})
	case "esc":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape})
	default:
		return tea.KeyPressMsg(tea.Key{})
	}
}

func modelWith(conns []config.Connection) Model {
	m := New()
	m.SetConnections(conns)
	return m
}

func TestUpdate_Navigation(t *testing.T) {
	tests := []struct {
		name         string
		keys         []string
		wantSelected int
	}{
		{
			name:         "move_down_once",
			keys:         []string{"j"},
			wantSelected: 1,
		},
		{
			name:         "move_down_twice",
			keys:         []string{"j", "j"},
			wantSelected: 2,
		},
		{
			name:         "clamps_at_bottom",
			keys:         []string{"j", "j", "j"},
			wantSelected: 2,
		},
		{
			name:         "move_down_then_up",
			keys:         []string{"j", "k"},
			wantSelected: 0,
		},
		{
			name:         "clamps_at_top",
			keys:         []string{"k"},
			wantSelected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := modelWith(testConnections)
			for _, k := range tt.keys {
				m, _ = m.Update(keyMsg(k))
			}
			if got := m.selected; got != tt.wantSelected {
				t.Errorf("selected = %d, want %d", got, tt.wantSelected)
			}
		})
	}
}

func TestUpdate_Events(t *testing.T) {
	tests := []struct {
		name      string
		keys      []string
		wantEvent Event
	}{
		{
			name:      "enter_returns_event_selected",
			keys:      []string{"enter"},
			wantEvent: EventSelected,
		},
		{
			name:      "esc_returns_event_closed",
			keys:      []string{"esc"},
			wantEvent: EventClosed,
		},
		{
			name:      "navigation_returns_event_none",
			keys:      []string{"j"},
			wantEvent: EventNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := modelWith(testConnections)
			var event Event
			for _, k := range tt.keys {
				m, event = m.Update(keyMsg(k))
			}
			if event != tt.wantEvent {
				t.Errorf("event = %v, want %v", event, tt.wantEvent)
			}
		})
	}
}

func TestSelected(t *testing.T) {
	tests := []struct {
		name     string
		keys     []string
		wantName string
	}{
		{
			name:     "first_item_by_default",
			keys:     nil,
			wantName: "local-sqlite",
		},
		{
			name:     "second_item_after_moving_down",
			keys:     []string{"j"},
			wantName: "prod-postgres",
		},
		{
			name:     "third_item_after_moving_down_twice",
			keys:     []string{"j", "j"},
			wantName: "staging-postgres",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := modelWith(testConnections)
			for _, k := range tt.keys {
				m, _ = m.Update(keyMsg(k))
			}
			if got := m.Selected().Name; got != tt.wantName {
				t.Errorf("Selected().Name = %q, want %q", got, tt.wantName)
			}
		})
	}
}

func TestSelected_Empty(t *testing.T) {
	m := New()
	if got := m.Selected(); got != (config.Connection{}) {
		t.Errorf("Selected() on empty model = %v, want zero value", got)
	}
}
