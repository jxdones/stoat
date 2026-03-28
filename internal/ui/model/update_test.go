package model

import (
	"strings"
	"testing"

	"github.com/jxdones/stoat/internal/database"
)

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
		key               string
		initiallyExpanded bool
		wantExpanded      bool
	}{
		{
			name:              "toggle_help_expanded_on",
			initiallyExpanded: false,
			key:               "?",
			wantExpanded:      true,
		},
		{
			name:              "toggle_help_expanded_off",
			initiallyExpanded: true,
			key:               "esc",
			wantExpanded:      false,
		},
		{
			name:              "toggle_help_expanded_on_when_already_expanded",
			initiallyExpanded: true,
			key:               "?",
			wantExpanded:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			m.helpExpanded = tt.initiallyExpanded
			result, _ := m.handleKeyPress(keyMsg(tt.key))
			got := result.(Model)
			if got.helpExpanded != tt.wantExpanded {
				t.Errorf("helpExpanded = %v, want %v", m.helpExpanded, tt.wantExpanded)
			}
		})
	}
}

func TestHandleCellDetailKey(t *testing.T) {
	tests := []struct {
		name      string
		setup     func() Model
		wantModal activeModal
	}{
		{
			name:      "v_on_table_with_data_opens_modal",
			setup:     modelWithTableFocusAndData,
			wantModal: modalCellDetail,
		},
		{
			name:      "v_on_jsonb_column_opens_modal",
			setup:     modelWithJSONBColumn,
			wantModal: modalCellDetail,
		},
		{
			name: "v_without_table_focus_does_nothing",
			setup: func() Model {
				m := modelWithTableFocusAndData()
				m.view.focus = FocusSidebar
				return m
			},
			wantModal: modalNone,
		},
		{
			name: "v_on_empty_table_does_nothing",
			setup: func() Model {
				m := New()
				m.view.focus = FocusTable
				return m
			},
			wantModal: modalNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.setup()
			result, _ := m.handleKeyPress(keyMsg("v"))
			got := result.(Model)
			if got.activeModal != tt.wantModal {
				t.Errorf("activeModal = %v, want %v", got.activeModal, tt.wantModal)
			}
		})
	}
}

func TestHandleCellDetailModalKey(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		wantModal activeModal
	}{
		{
			name:      "esc_closes_modal",
			key:       "esc",
			wantModal: modalNone,
		},
		{
			name:      "other_key_keeps_modal_open",
			key:       "j",
			wantModal: modalCellDetail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := modelWithTableFocusAndData()
			opened, _ := m.handleKeyPress(keyMsg("v"))
			m = opened.(Model)

			result, _ := m.handleKeyPress(keyMsg(tt.key))
			got := result.(Model)
			if got.activeModal != tt.wantModal {
				t.Errorf("activeModal = %v, want %v", got.activeModal, tt.wantModal)
			}
		})
	}
}

func modelWithJSONBColumn() Model {
	m := New()
	m.view.focus = FocusTable
	m.sidebar.SetDatabases([]string{"mydb"})
	m.sidebar.SetTables("mydb", []string{"books"})
	m.sidebar.OpenSelectedDatabase()
	m.table.SetColumns(dbColumnsToTable([]database.Column{
		{Key: "metadata", Title: "metadata", Type: "jsonb", MinWidth: 8},
	}))
	m.table.SetRows(dbRowsToTable([]database.Row{
		{"metadata": `{"publisher":"O'Reilly","tags":["go","db"]}`},
	}))
	return m
}
