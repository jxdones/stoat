package model

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/jxdones/stoat/internal/database"
)

// modelWithTableFocusAndData returns a model with FocusTable, a sidebar pointing
// at mydb.users, and a one-column/one-row table so ActiveCell() returns a value.
func modelWithTableFocusAndData() Model {
	m := New()
	m.view.focus = FocusTable
	m.sidebar.SetDatabases([]string{"mydb"})
	m.sidebar.SetTables("mydb", []string{"users"})
	m.sidebar.OpenSelectedDatabase()
	m.table.SetColumns(dbColumnsToTable([]database.Column{
		{Key: "name", Title: "name", Type: "text", MinWidth: 4},
	}))
	m.table.SetRows(dbRowsToTable([]database.Row{
		{"name": "Alice"},
	}))
	return m
}

// ellipsis is the same character used in queryPreviewForHeader for truncation.
const ellipsis = "…"

func TestQueryPreviewForHeader(t *testing.T) {
	const maxLen = 52

	tests := []struct {
		name  string
		query string
		want  string
	}{
		{
			name:  "empty_returns_empty",
			query: "",
			want:  "",
		},
		{
			name:  "whitespace_only_returns_empty",
			query: "   \n\t  ",
			want:  "",
		},
		{
			name:  "short_query_unchanged",
			query: "SELECT 1",
			want:  "SELECT 1",
		},
		{
			name:  "single_word_unchanged",
			query: "SELECT",
			want:  "SELECT",
		},
		{
			name:  "newlines_collapsed_to_single_space",
			query: "SELECT *\nFROM users\nWHERE id = 1",
			want:  "SELECT * FROM users WHERE id = 1",
		},
		{
			name:  "multiple_spaces_collapsed",
			query: "SELECT   *   FROM   users",
			want:  "SELECT * FROM users",
		},
		{
			name:  "leading_and_trailing_space_trimmed",
			query: "  SELECT * FROM users  ",
			want:  "SELECT * FROM users",
		},
		{
			name:  "exactly_52_chars_not_truncated",
			query: strings.Repeat("x", maxLen),
			want:  strings.Repeat("x", maxLen),
		},
		{
			name:  "53_chars_truncated_with_ellipsis",
			query: strings.Repeat("a", 53),
			want:  strings.Repeat("a", maxLen-1) + ellipsis,
		},
		{
			name:  "long_query_truncated",
			query: "SELECT id, name, email FROM users WHERE active = 1 ORDER BY created_at DESC LIMIT 100",
			want:  "SELECT id, name, email FROM users WHERE active = 1 " + ellipsis,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := queryPreviewForHeader(tt.query)
			if got != tt.want {
				t.Errorf("queryPreviewForHeader() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHelpExpanded(t *testing.T) {
	tests := []struct {
		name              string
		msg               tea.KeyPressMsg
		initiallyExpanded bool
		wantExpanded      bool
	}{
		{
			name:              "toggle_help_expanded_on",
			initiallyExpanded: false,
			msg:               keyMsg("?"),
			wantExpanded:      true,
		},
		{
			name:              "toggle_help_expanded_off",
			initiallyExpanded: true,
			msg:               keyMsg("esc"),
			wantExpanded:      false,
		},
		{
			name:              "toggle_help_expanded_on_when_already_expanded",
			initiallyExpanded: true,
			msg:               keyMsg("?"),
			wantExpanded:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			m.helpExpanded = tt.initiallyExpanded
			result, _ := m.handleKeyPress(tt.msg)
			got := result.(Model)
			if got.helpExpanded != tt.wantExpanded {
				t.Errorf("helpExpanded = %v, want %v", m.helpExpanded, tt.wantExpanded)
			}
		})
	}
}

func TestHandleUpdateFromCell(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(*Model)
		key           string
		wantEditMode  bool
		wantEditValue string
	}{
		{
			name:          "enter_with_table_focused_enters_edit_mode",
			key:           "enter",
			wantEditMode:  true,
			wantEditValue: "Alice",
		},
		{
			name: "enter_blocked_when_viewing_query_result",
			setup: func(m *Model) {
				m.viewingQueryResult = true
			},
			key:          "enter",
			wantEditMode: false,
		},
		{
			name: "enter_blocked_when_no_table_selected",
			setup: func(m *Model) {
				m.sidebar.SetDatabases([]string{})
			},
			key:          "enter",
			wantEditMode: false,
		},
		{
			name: "enter_blocked_when_tab_is_not_records",
			setup: func(m *Model) {
				m.tabs.SetActive(1) // Columns
			},
			key:          "enter",
			wantEditMode: false,
		},
		{
			name:         "non_enter_key_does_not_enter_edit_mode",
			key:          "j",
			wantEditMode: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := modelWithTableFocusAndData()
			if tt.setup != nil {
				tt.setup(&m)
			}
			got, _, _ := m.handleUpdateFromCell(keyMsg(tt.key))
			next := got.(Model)
			if next.inlineEditMode != tt.wantEditMode {
				t.Errorf("inlineEditMode = %v, want %v", next.inlineEditMode, tt.wantEditMode)
			}
			if tt.wantEditValue != "" && next.editbox.Value() != tt.wantEditValue {
				t.Errorf("editbox value = %q, want %q", next.editbox.Value(), tt.wantEditValue)
			}
		})
	}
}

func TestHandleInlineEditConfirm(t *testing.T) {
	tests := []struct {
		name              string
		setup             func(*Model)
		wantHandled       bool
		wantEditMode      bool
		wantCmd           bool
		wantPendingReload bool
	}{
		{
			name: "confirm_with_changed_value_fires_query",
			setup: func(m *Model) {
				m.inlineEditMode = true
				m.editbox.SetValue("Bob")
			},
			wantHandled:       true,
			wantEditMode:      false,
			wantCmd:           true,
			wantPendingReload: true,
		},
		{
			name: "confirm_with_unchanged_value_skips_query",
			setup: func(m *Model) {
				m.inlineEditMode = true
				m.editbox.SetValue("Alice")
			},
			wantHandled:       true,
			wantEditMode:      false,
			wantCmd:           false,
			wantPendingReload: false,
		},
		{
			name:         "not_in_edit_mode_returns_unhandled",
			setup:        func(m *Model) { m.inlineEditMode = false },
			wantHandled:  false,
			wantEditMode: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := modelWithTableFocusAndData()
			m.SetDataSource(mockDataSource{})
			if tt.setup != nil {
				tt.setup(&m)
			}
			got, cmd, handled := m.handleInlineEditConfirm(keyMsg("enter"))
			next := got.(Model)
			if handled != tt.wantHandled {
				t.Errorf("handled = %v, want %v", handled, tt.wantHandled)
			}
			if next.inlineEditMode != tt.wantEditMode {
				t.Errorf("inlineEditMode = %v, want %v", next.inlineEditMode, tt.wantEditMode)
			}
			if tt.wantCmd && cmd == nil {
				t.Error("expected non-nil cmd, got nil")
			}
			if !tt.wantCmd && cmd != nil {
				t.Errorf("expected nil cmd, got %v", cmd)
			}
			if next.pendingTableReload != tt.wantPendingReload {
				t.Errorf("pendingTableReload = %v, want %v", next.pendingTableReload, tt.wantPendingReload)
			}
		})
	}
}

func TestEscCancelsEditMode(t *testing.T) {
	tests := []struct {
		name           string
		inlineEditMode bool
		wantEditMode   bool
		wantFocus      FocusedPanel
	}{
		{
			name:           "esc_in_edit_mode_cancels_and_keeps_table_focus",
			inlineEditMode: true,
			wantEditMode:   false,
			wantFocus:      FocusTable,
		},
		{
			name:           "esc_in_normal_mode_clears_focus",
			inlineEditMode: false,
			wantEditMode:   false,
			wantFocus:      FocusNone,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := modelWithTableFocusAndData()
			m.inlineEditMode = tt.inlineEditMode
			got, _ := m.handleKeyPress(keyMsg("esc"))
			next := got.(Model)
			if next.inlineEditMode != tt.wantEditMode {
				t.Errorf("inlineEditMode = %v, want %v", next.inlineEditMode, tt.wantEditMode)
			}
			if next.view.focus != tt.wantFocus {
				t.Errorf("focus = %v, want %v", next.view.focus, tt.wantFocus)
			}
		})
	}
}

func TestEditModeTypingNotIntercepted(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{
			name: "y_not_intercepted_by_copy_handler",
			key:  "y",
		},
		{
			name: "j_not_intercepted_by_table_navigation",
			key:  "j",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := modelWithTableFocusAndData()
			m.inlineEditMode = true
			m.editbox.Focus()
			before := m.editbox.Value()
			got, _ := m.handleKeyPress(keyMsg(tt.key))
			next := got.(Model)
			if !next.inlineEditMode {
				t.Error("inlineEditMode was unexpectedly cleared")
			}
			if next.editbox.Value() == before+"copy" {
				t.Errorf("key %q was intercepted by copy handler", tt.key)
			}
		})
	}
}

func TestEditModeBlocksNavigationKeys(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{
			name: "tab_does_not_change_focus",
			key:  "tab",
		},
		{
			name: "shift_tab_does_not_change_focus",
			key:  "shift+tab",
		},
		{
			name: "slash_does_not_switch_to_filterbox",
			key:  "/",
		},
		{
			name: "question_mark_does_not_toggle_help",
			key:  "?",
		},
		{
			name: "ctrl_r_does_not_reload",
			key:  "ctrl+r",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := modelWithTableFocusAndData()
			m.inlineEditMode = true
			got, cmd := m.handleKeyPress(keyMsg(tt.key))
			next := got.(Model)
			if !next.inlineEditMode {
				t.Error("inlineEditMode was cleared unexpectedly")
			}
			if next.view.focus != FocusTable {
				t.Errorf("focus changed to %v, want FocusTable", next.view.focus)
			}
			if cmd != nil {
				t.Errorf("expected nil cmd, got %v", cmd)
			}
		})
	}
}

func TestHelpBindingsInEditMode(t *testing.T) {
	tests := []struct {
		name           string
		inlineEditMode bool
		wantGlobalsNil bool
		wantPaneKeys   []string
	}{
		{
			name:           "edit_mode_returns_only_editbox_bindings",
			inlineEditMode: true,
			wantGlobalsNil: true,
			wantPaneKeys:   []string{"enter", "esc"},
		},
		{
			name:           "normal_mode_returns_table_bindings_and_globals",
			inlineEditMode: false,
			wantGlobalsNil: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := modelWithTableFocusAndData()
			m.inlineEditMode = tt.inlineEditMode
			pane, global := m.helpBindings()
			if tt.wantGlobalsNil && len(global) != 0 {
				t.Errorf("global bindings = %d, want 0", len(global))
			}
			if !tt.wantGlobalsNil && len(global) == 0 {
				t.Error("expected non-empty global bindings")
			}
			for _, wantKey := range tt.wantPaneKeys {
				found := false
				for _, b := range pane {
					if strings.Contains(b.Help().Key, wantKey) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("pane bindings missing key %q", wantKey)
				}
			}
		})
	}
}
