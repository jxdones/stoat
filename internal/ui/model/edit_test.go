package model

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestHandleEditKey(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(*Model)
		key           string
		wantMode      mode
		wantEditValue string
	}{
		{
			name:          "enter_with_table_focused_enters_edit_mode",
			key:           "enter",
			wantMode:      modeInsert,
			wantEditValue: "Alice",
		},
		{
			name: "enter_blocked_when_viewing_query_result",
			setup: func(m *Model) {
				m.viewingQueryResult = true
			},
			key:      "enter",
			wantMode: modeNormal,
		},
		{
			name: "enter_blocked_when_no_table_selected",
			setup: func(m *Model) {
				m.sidebar.SetDatabases([]string{})
			},
			key:      "enter",
			wantMode: modeNormal,
		},
		{
			name: "enter_blocked_when_tab_is_not_records",
			setup: func(m *Model) {
				m.tabs.SetActive(1) // Columns
			},
			key:      "enter",
			wantMode: modeNormal,
		},
		{
			name:     "non_enter_key_does_not_enter_edit_mode",
			key:      "j",
			wantMode: modeNormal,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := modelWithTableFocusAndData()
			if tt.setup != nil {
				tt.setup(&m)
			}
			got, _, _ := m.handleEditKey(keyMsg(tt.key))
			next := got.(Model)
			if next.mode != tt.wantMode {
				t.Errorf("mode = %v, want %v", next.mode, tt.wantMode)
			}
			if tt.wantEditValue != "" && next.editbox.Value() != tt.wantEditValue {
				t.Errorf("editbox value = %q, want %q", next.editbox.Value(), tt.wantEditValue)
			}
		})
	}
}

func TestConfirmInlineEdit(t *testing.T) {
	tests := []struct {
		name              string
		setup             func(*Model)
		wantHandled       bool
		wantMode          mode
		wantCmd           bool
		wantPendingReload bool
	}{
		{
			name: "confirm_with_changed_value_fires_query",
			setup: func(m *Model) {
				m.mode = modeInsert
				m.editbox.SetValue("Bob")
			},
			wantHandled:       true,
			wantMode:          modeNormal,
			wantCmd:           true,
			wantPendingReload: true,
		},
		{
			name: "confirm_with_unchanged_value_skips_query",
			setup: func(m *Model) {
				m.mode = modeInsert
				m.editbox.SetValue("Alice")
			},
			wantHandled:       true,
			wantMode:          modeNormal,
			wantCmd:           false,
			wantPendingReload: false,
		},
		{
			name:        "not_in_edit_mode_returns_unhandled",
			setup:       func(m *Model) { m.mode = modeNormal },
			wantHandled: false,
			wantMode:    modeNormal,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := modelWithTableFocusAndData()
			m.SetDataSource(mockDataSource{})
			if tt.setup != nil {
				tt.setup(&m)
			}
			got, cmd, handled := m.confirmInlineEdit(keyMsg("enter"))
			next := got.(Model)
			if handled != tt.wantHandled {
				t.Errorf("handled = %v, want %v", handled, tt.wantHandled)
			}
			if next.mode != tt.wantMode {
				t.Errorf("mode = %v, want %v", next.mode, tt.wantMode)
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
		name        string
		initialMode mode
		wantMode    mode
		wantFocus   FocusedPanel
	}{
		{
			name:        "esc_in_edit_mode_cancels_and_keeps_table_focus",
			initialMode: modeInsert,
			wantMode:    modeNormal,
			wantFocus:   FocusTable,
		},
		{
			name:        "esc_in_normal_mode_clears_focus",
			initialMode: modeNormal,
			wantMode:    modeNormal,
			wantFocus:   FocusNone,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := modelWithTableFocusAndData()
			m.mode = tt.initialMode
			got, _ := m.handleKeyPress(keyMsg("esc"))
			next := got.(Model)
			if next.mode != tt.wantMode {
				t.Errorf("mode = %v, want %v", next.mode, tt.wantMode)
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
			m.mode = modeInsert
			m.editbox.Focus()
			before := m.editbox.Value()
			got, _ := m.handleKeyPress(keyMsg(tt.key))
			next := got.(Model)
			if next.mode != modeInsert {
				t.Error("mode was unexpectedly cleared")
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
			m.mode = modeInsert
			got, cmd := m.handleKeyPress(keyMsg(tt.key))
			next := got.(Model)
			if next.mode != modeInsert {
				t.Error("mode was unexpectedly cleared")
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
		name             string
		initialMode      mode
		wantGlobalsInBar bool
		wantPaneKeys     []string
	}{
		{
			name:             "edit_mode_shows_only_editbox_bindings",
			initialMode:      modeInsert,
			wantGlobalsInBar: false,
			wantPaneKeys:     []string{"enter", "esc"},
		},
		{
			name:             "normal_mode_includes_global_bindings",
			initialMode:      modeNormal,
			wantGlobalsInBar: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := modelWithTableFocusAndData()
			m.mode = tt.initialMode
			bindings := m.statusBindings()
			hasGlobal := false
			for _, b := range bindings {
				if b.Help().Key == "tab" {
					hasGlobal = true
					break
				}
			}
			if tt.wantGlobalsInBar && !hasGlobal {
				t.Error("expected global bindings in status bar")
			}
			if !tt.wantGlobalsInBar && hasGlobal {
				t.Error("global bindings should be suppressed in edit mode")
			}
			for _, wantKey := range tt.wantPaneKeys {
				found := false
				for _, b := range bindings {
					if strings.Contains(b.Help().Key, wantKey) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("status bindings missing key %q", wantKey)
				}
			}
		})
	}
}

func TestOpenEditor(t *testing.T) {
	tests := []struct {
		name           string
		hasConnection  bool
		wantStatusText string
		wantCmd        bool
	}{
		{
			name:           "no_connection_shows_warning",
			hasConnection:  false,
			wantStatusText: "No active connection",
			wantCmd:        true, // TTL timer cmd
		},
		{
			name:          "with_connection_fires_editor_cmd",
			hasConnection: true,
			wantCmd:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			if tt.hasConnection {
				m.source = mockDataSource{}
			}
			next, cmd := m.openEditor()
			got := next.(Model)
			if tt.wantStatusText != "" && !strings.Contains(statusText(got), tt.wantStatusText) {
				t.Errorf("status %q does not contain %q", statusText(got), tt.wantStatusText)
			}
			if tt.wantCmd && cmd == nil {
				t.Error("expected non-nil cmd, got nil")
			}
		})
	}
}

func TestHandleDeleteKey(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*Model)
		key         string
		prevKey     string
		wantHandled bool
		wantCmd     bool
		wantMode    mode
		wantLastKey string
	}{
		{
			name: "requires_table_focus",
			setup: func(m *Model) {
				m.view.focus = FocusSidebar
			},
			key:         "d",
			prevKey:     "d",
			wantHandled: false,
			wantCmd:     false,
			wantMode:    modeNormal,
		},
		{
			name:        "requires_d_key",
			key:         "x",
			prevKey:     "d",
			wantHandled: false,
			wantCmd:     false,
			wantMode:    modeNormal,
		},
		{
			name: "requires_records_tab",
			setup: func(m *Model) {
				m.tabs.SetActive(1) // Columns
			},
			key:         "d",
			prevKey:     "d",
			wantHandled: false,
			wantCmd:     false,
			wantMode:    modeNormal,
		},
		{
			name: "viewing_query_result_blocks_delete",
			setup: func(m *Model) {
				m.viewingQueryResult = true
			},
			key:         "d",
			prevKey:     "d",
			wantHandled: true,
			wantCmd:     true,
			wantMode:    modeNormal,
		},
		{
			name: "read_only_blocks_delete",
			setup: func(m *Model) {
				m.readOnly = true
			},
			key:         "d",
			prevKey:     "d",
			wantHandled: true,
			wantCmd:     true,
			wantMode:    modeNormal,
		},
		{
			name: "requires_active_row",
			setup: func(m *Model) {
				m.table.SetRows(nil)
			},
			key:         "d",
			prevKey:     "d",
			wantHandled: false,
			wantCmd:     false,
			wantMode:    modeNormal,
		},
		{
			name:        "first_d_stores_last_key",
			key:         "d",
			prevKey:     "",
			wantHandled: true,
			wantCmd:     false,
			wantMode:    modeNormal,
			wantLastKey: "d",
		},
		{
			name:        "dd_starts_delete_confirmation",
			key:         "d",
			prevKey:     "d",
			wantHandled: true,
			wantCmd:     false,
			wantMode:    modeDelete,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := modelWithTableFocusAndData()
			if tt.setup != nil {
				tt.setup(&m)
			}

			got, cmd, handled := m.handleDeleteKey(keyMsg(tt.key), tt.prevKey)
			next := got.(Model)
			if handled != tt.wantHandled {
				t.Errorf("handled = %v, want %v", handled, tt.wantHandled)
			}
			if tt.wantCmd && cmd == nil {
				t.Error("expected non-nil cmd, got nil")
			}
			if !tt.wantCmd && cmd != nil {
				t.Errorf("expected nil cmd, got %v", cmd)
			}
			if next.mode != tt.wantMode {
				t.Errorf("mode = %v, want %v", next.mode, tt.wantMode)
			}
			if next.lastKey != tt.wantLastKey {
				t.Errorf("lastKey = %q, want %q", next.lastKey, tt.wantLastKey)
			}
		})
	}
}

func TestConfirmDelete(t *testing.T) {
	tests := []struct {
		name              string
		setup             func(*Model)
		key               string
		wantHandled       bool
		wantCmd           bool
		wantMode          mode
		wantPendingReload bool
		wantQueryContains []string
	}{
		{
			name:              "requires_pending_confirmation",
			key:               "y",
			wantHandled:       false,
			wantCmd:           false,
			wantMode:          modeNormal,
			wantPendingReload: false,
		},
		{
			name: "requires_y_key",
			setup: func(m *Model) {
				m.mode = modeDelete
			},
			key:               "n",
			wantHandled:       false,
			wantCmd:           false,
			wantMode:          modeDelete,
			wantPendingReload: false,
		},
		{
			name: "missing_row_is_handled_without_query",
			setup: func(m *Model) {
				m.mode = modeDelete
				m.table.SetRows(nil)
			},
			key:               "y",
			wantHandled:       true,
			wantCmd:           false,
			wantMode:          modeNormal,
			wantPendingReload: false,
		},
		{
			name: "missing_database_or_table_is_handled_without_query",
			setup: func(m *Model) {
				m.mode = modeDelete
				m.sidebar.SetDatabases(nil)
			},
			key:               "y",
			wantHandled:       true,
			wantCmd:           false,
			wantMode:          modeNormal,
			wantPendingReload: false,
		},
		{
			name: "enqueues_delete_query_run",
			setup: func(m *Model) {
				m.mode = modeDelete
			},
			key:               "y",
			wantHandled:       true,
			wantCmd:           true,
			wantMode:          modeNormal,
			wantPendingReload: true,
			wantQueryContains: []string{"DELETE FROM \"users\"", "\"name\" = 'Alice'"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := modelWithTableFocusAndData()
			if tt.setup != nil {
				tt.setup(&m)
			}

			got, cmd, handled := m.confirmDelete(keyMsg(tt.key))
			next := got.(Model)
			if handled != tt.wantHandled {
				t.Errorf("handled = %v, want %v", handled, tt.wantHandled)
			}
			if tt.wantCmd && cmd == nil {
				t.Fatal("expected non-nil cmd, got nil")
			}
			if !tt.wantCmd && cmd != nil {
				t.Errorf("expected nil cmd, got %v", cmd)
			}
			if next.mode != tt.wantMode {
				t.Errorf("mode = %v, want %v", next.mode, tt.wantMode)
			}
			if next.pendingTableReload != tt.wantPendingReload {
				t.Errorf("pendingTableReload = %v, want %v", next.pendingTableReload, tt.wantPendingReload)
			}

			if len(tt.wantQueryContains) > 0 {
				msg, ok := findMsg[QueryRunRequestedMsg](cmd)
				if !ok {
					t.Fatal("expected QueryRunRequestedMsg in command")
				}
				for _, sub := range tt.wantQueryContains {
					if !strings.Contains(msg.Query, sub) {
						t.Errorf("query %q does not contain %q", msg.Query, sub)
					}
				}
			}
		})
	}
}

func TestCancelDelete(t *testing.T) {
	tests := []struct {
		name        string
		initialMode mode
		wantMode    mode
		key         string
		wantHandled bool
	}{
		{
			name:        "requires_pending_confirmation",
			initialMode: modeNormal,
			wantMode:    modeNormal,
			key:         "n",
			wantHandled: false,
		},
		{
			name:        "ignores_other_keys",
			initialMode: modeDelete,
			wantMode:    modeDelete,
			key:         "y",
			wantHandled: false,
		},
		{
			name:        "n_cancels_pending_delete",
			initialMode: modeDelete,
			wantMode:    modeNormal,
			key:         "n",
			wantHandled: true,
		},
		{
			name:        "esc_cancels_pending_delete",
			initialMode: modeDelete,
			wantMode:    modeNormal,
			key:         "esc",
			wantHandled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := modelWithTableFocusAndData()
			m.mode = tt.initialMode

			got, cmd, handled := m.cancelDelete(keyMsg(tt.key))
			next := got.(Model)
			if handled != tt.wantHandled {
				t.Errorf("handled = %v, want %v", handled, tt.wantHandled)
			}
			if cmd != nil {
				t.Errorf("expected nil cmd, got %v", cmd)
			}
			if next.mode != tt.wantMode {
				t.Errorf("mode = %v, want %v", next.mode, tt.wantMode)
			}
		})
	}
}

func TestDelegatePaste_InlineEditMode(t *testing.T) {
	tests := []struct {
		name         string
		wantMode     mode
		initialValue string
		pasteContent string
		wantValue    string
	}{
		{
			name:         "paste_appended_when_in_inline_edit_mode",
			wantMode:     modeInsert,
			initialValue: "hello",
			pasteContent: " world",
			wantValue:    "hello world",
		},
		{
			name:         "paste_ignored_when_not_in_inline_edit_mode",
			wantMode:     modeNormal,
			initialValue: "hello",
			pasteContent: " world",
			wantValue:    "hello",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			m.mode = tt.wantMode
			m.editbox.SetValue(tt.initialValue)
			if tt.wantMode == modeInsert {
				m.editbox.Focus()
			}

			next, _ := m.delegatePaste(tea.PasteMsg{Content: tt.pasteContent})
			got := next.(Model)
			if got.editbox.Value() != tt.wantValue {
				t.Errorf("editbox value = %q, want %q", got.editbox.Value(), tt.wantValue)
			}
		})
	}
}

func TestOnCellEditorDone(t *testing.T) {
	tests := []struct {
		name              string
		setup             func(*Model)
		wantCmd           bool
		wantPendingReload bool
		msg               EditorCellMsg
	}{
		{
			name:              "runs_update_query_when_value_is_changed",
			wantCmd:           true,
			wantPendingReload: true,
			msg:               EditorCellMsg{Value: "Bob", Err: nil},
		},
		{
			name:              "skips_query_when_value_is_unchanged",
			wantCmd:           false,
			wantPendingReload: false,
			msg:               EditorCellMsg{Value: "Alice", Err: nil},
		},
		{
			name: "shows_error_status_on_editor_error",
			setup: func(m *Model) {
				m.mode = modeNormal
			},
			wantCmd:           true, // TTL status cmd
			wantPendingReload: false,
			msg:               EditorCellMsg{Err: errors.New("editor failed")},
		},
		{
			name: "blocked_when_viewing_query_result",
			setup: func(m *Model) {
				m.viewingQueryResult = true
			},
			wantCmd:           true, // TTL status cmd
			wantPendingReload: false,
			msg:               EditorCellMsg{Value: "Bob"},
		},
		{
			name: "blocked_when_read_only",
			setup: func(m *Model) {
				m.readOnly = true
			},
			wantCmd:           true, // TTL status cmd
			wantPendingReload: false,
			msg:               EditorCellMsg{Value: "Bob"},
		},
		{
			name: "no_op_when_tab_is_not_records",
			setup: func(m *Model) {
				m.tabs.SetActive(1) // Columns
			},
			wantCmd:           false,
			wantPendingReload: false,
			msg:               EditorCellMsg{Value: "Bob"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := modelWithTableFocusAndData()
			m.SetDataSource(mockDataSource{})
			if tt.setup != nil {
				tt.setup(&m)
			}
			next, cmd := m.onCellEditorDone(tt.msg)
			got := next.(Model)
			if tt.wantCmd && cmd == nil {
				t.Error("expected non-nil cmd, got nil")
			}
			if !tt.wantCmd && cmd != nil {
				t.Errorf("expected nil cmd, got %v", cmd)
			}
			if got.pendingTableReload != tt.wantPendingReload {
				t.Errorf("pendingTableReload = %v, want %v", got.pendingTableReload, tt.wantPendingReload)
			}
		})
	}
}

func TestHandleCellEditorKey(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*Model)
		key         string
		wantHandled bool
		wantCmd     bool
	}{
		{
			name:        "e_with_table_focused_fires_editor_cmd",
			key:         "e",
			wantHandled: true,
			wantCmd:     true,
		},
		{
			name: "e_blocked_when_not_table_focused",
			setup: func(m *Model) {
				m.view.focus = FocusQuerybox
			},
			key:         "e",
			wantHandled: false,
			wantCmd:     false,
		},
		{
			name: "e_blocked_when_viewing_query_result",
			setup: func(m *Model) {
				m.viewingQueryResult = true
			},
			key:         "e",
			wantHandled: true,
			wantCmd:     true, // TTL status cmd
		},
		{
			name: "e_blocked_when_tab_is_not_records",
			setup: func(m *Model) {
				m.tabs.SetActive(1) // Columns
			},
			key:         "e",
			wantHandled: false,
			wantCmd:     false,
		},
		{
			name:        "non_e_key_returns_unhandled",
			key:         "x",
			wantHandled: false,
			wantCmd:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := modelWithTableFocusAndData()
			if tt.setup != nil {
				tt.setup(&m)
			}
			_, cmd, handled := m.handleCellEditorKey(keyMsg(tt.key))
			if handled != tt.wantHandled {
				t.Errorf("handled = %v, want %v", handled, tt.wantHandled)
			}
			if tt.wantCmd && cmd == nil {
				t.Error("expected non-nil cmd, got nil")
			}
			if !tt.wantCmd && cmd != nil {
				t.Errorf("expected nil cmd, got %v", cmd)
			}
		})
	}
}
