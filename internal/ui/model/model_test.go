package model

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// keyMsg returns a tea.KeyPressMsg for testing. s must match msg.String() in model.Update (e.g. "q", "tab", "shift+tab", "ctrl+c").
func keyMsg(s string) tea.KeyPressMsg {
	switch s {
	case "q":
		return tea.KeyPressMsg(tea.Key{Code: 'q', Text: "q"})
	case "tab":
		// KeyTab.String() returns "\t", not "tab"; model expects "tab". Use for focusNext tests only.
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyTab, Text: "\t"})
	case "shift+tab":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyTab, Mod: tea.ModShift})
	case "ctrl+c":
		return tea.KeyPressMsg(tea.Key{Code: 'c', Mod: tea.ModCtrl})
	case "enter":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})
	default:
		return tea.KeyPressMsg(tea.Key{Code: rune(s[0]), Text: s})
	}
}

func TestView_Smoke(t *testing.T) {
	m := New()
	m.view.width = 80
	m.view.height = 24
	view := m.View()
	if view.Content == "" {
		t.Error("View() returned empty string")
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name       string
		checkFocus bool
		checkSize  bool
	}{
		{
			name:       "initial_focus_is_sidebar",
			checkFocus: true,
			checkSize:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			if tt.checkFocus && m.view.focus != FocusSidebar {
				t.Errorf("New() focus = %v, want FocusSidebar", m.view.focus)
			}
			if tt.checkSize && (m.view.width != 80 || m.view.height != 24) {
				t.Errorf("New() default size = %dx%d, want 80x24", m.view.width, m.view.height)
			}
		})
	}
}

func TestInit(t *testing.T) {
	tests := []struct {
		name    string
		wantNil bool
	}{
		{
			name:    "returns_nil_cmd",
			wantNil: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			cmd := m.Init()
			if tt.wantNil && cmd != nil {
				t.Error("Init() should return nil Cmd")
			}
		})
	}
}

func TestIsFocused(t *testing.T) {
	tests := []struct {
		name  string
		focus FocusedPanel
		panel FocusedPanel
		want  bool
	}{
		{
			name:  "sidebar_focused_when_focus_sidebar",
			focus: FocusSidebar,
			panel: FocusSidebar,
			want:  true,
		},
		{
			name:  "sidebar_not_focused_when_focus_filterbox",
			focus: FocusFilterbox,
			panel: FocusSidebar,
			want:  false,
		},
		{
			name:  "table_focused_when_focus_table",
			focus: FocusTable,
			panel: FocusTable,
			want:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			m.view.focus = tt.focus
			got := m.isFocused(tt.panel)
			if got != tt.want {
				t.Errorf("isFocused(%v) with focus %v = %v, want %v", tt.panel, tt.focus, got, tt.want)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	tests := []struct {
		name        string
		msg         tea.Msg
		setup       func(*Model)
		wantWidth   int
		wantHeight  int
		wantQuitCmd bool
		wantFocus   *FocusedPanel
	}{
		{
			name:       "WindowSizeMsg_updates_dimensions",
			msg:        tea.WindowSizeMsg{Width: 100, Height: 30},
			wantWidth:  100,
			wantHeight: 30,
		},
		{
			name:        "key_q_returns_quit_cmd",
			msg:         keyMsg("q"),
			wantQuitCmd: true,
		},
		{
			name:        "key_ctrl_c_returns_quit_cmd",
			msg:         keyMsg("ctrl+c"),
			wantQuitCmd: true,
		},
		{
			name: "shift_tab_from_table_cycles_focus_backward",
			msg:  keyMsg("shift+tab"),
			setup: func(m *Model) {
				m.view.focus = FocusTable
			},
			wantFocus: ptrFocus(FocusFilterbox),
		},
		{
			name:       "WindowSizeMsg_returns_nil_cmd",
			msg:        tea.WindowSizeMsg{Width: 80, Height: 24},
			wantWidth:  80,
			wantHeight: 24,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			if tt.setup != nil {
				tt.setup(&m)
			}
			next, cmd := m.Update(tt.msg)
			got := next.(Model)
			if tt.wantQuitCmd {
				if cmd == nil {
					t.Error("Update() should return non-nil Quit Cmd")
				}
				return
			}
			if cmd != nil && !tt.wantQuitCmd {
				t.Errorf("Update() should return nil Cmd, got %v", cmd)
			}
			if tt.wantWidth != 0 && got.view.width != tt.wantWidth {
				t.Errorf("view.width = %d, want %d", got.view.width, tt.wantWidth)
			}
			if tt.wantHeight != 0 && got.view.height != tt.wantHeight {
				t.Errorf("view.height = %d, want %d", got.view.height, tt.wantHeight)
			}
			if tt.wantFocus != nil && got.view.focus != *tt.wantFocus {
				t.Errorf("view.focus = %v, want %v", got.view.focus, *tt.wantFocus)
			}
		})
	}
}

func ptrFocus(p FocusedPanel) *FocusedPanel {
	return &p
}

func TestUpdateFocused_sidebarEnter(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*Model)
		wantFocus  FocusedPanel
		wantOpenDB bool
	}{
		{
			name: "enter_on_db_list_opens_database",
			setup: func(m *Model) {
				m.sidebar.SetDatabases([]string{"mydb"})
				m.view.focus = FocusSidebar
			},
			wantFocus:  FocusSidebar,
			wantOpenDB: true,
		},
		{
			name: "enter_on_tables_section_moves_focus_to_table",
			setup: func(m *Model) {
				m.sidebar.SetDatabases([]string{"mydb"})
				m.sidebar.SetTables("mydb", []string{"users"})
				m.sidebar.OpenSelectedDatabase()
				m.view.focus = FocusSidebar
			},
			wantFocus: FocusTable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			if tt.setup != nil {
				tt.setup(&m)
			}
			msg := keyMsg("enter")
			next, cmd := m.Update(msg)
			if cmd != nil {
				t.Errorf("Update(Enter) should return nil Cmd, got %v", cmd)
			}
			got := next.(Model)
			if got.view.focus != tt.wantFocus {
				t.Errorf("after Enter: focus = %v, want %v", got.view.focus, tt.wantFocus)
			}
			if tt.wantOpenDB && got.sidebar.ActiveDB() == "" {
				t.Error("expected database to be opened")
			}
		})
	}
}

func TestFocusNext(t *testing.T) {
	tests := []struct {
		name     string
		initial  FocusedPanel
		wantNext FocusedPanel
	}{
		{
			name:     "sidebar_to_filterbox",
			initial:  FocusSidebar,
			wantNext: FocusFilterbox,
		},
		{
			name:     "filterbox_to_table",
			initial:  FocusFilterbox,
			wantNext: FocusTable,
		},
		{
			name:     "table_to_querybox",
			initial:  FocusTable,
			wantNext: FocusQuerybox,
		},
		{
			name:     "querybox_to_sidebar",
			initial:  FocusQuerybox,
			wantNext: FocusSidebar,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			m.view.focus = tt.initial
			m.focusNext()
			if m.view.focus != tt.wantNext {
				t.Errorf("focusNext() from %v: focus = %v, want %v", tt.initial, m.view.focus, tt.wantNext)
			}
		})
	}
}

func TestFocusPrevious(t *testing.T) {
	tests := []struct {
		name     string
		initial  FocusedPanel
		wantPrev FocusedPanel
	}{
		{
			name:     "sidebar_to_querybox",
			initial:  FocusSidebar,
			wantPrev: FocusQuerybox,
		},

		{
			name:     "querybox_to_table",
			initial:  FocusQuerybox,
			wantPrev: FocusTable,
		},
		{
			name:     "table_to_filterbox",
			initial:  FocusTable,
			wantPrev: FocusFilterbox,
		},
		{
			name:     "filterbox_to_sidebar",
			initial:  FocusFilterbox,
			wantPrev: FocusSidebar,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			m.view.focus = tt.initial
			m.focusPrevious()
			if m.view.focus != tt.wantPrev {
				t.Errorf("focusPrevious() from %v: focus = %v, want %v", tt.initial, m.view.focus, tt.wantPrev)
			}
		})
	}
}
