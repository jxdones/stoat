package model

import (
	"strings"
	"testing"

	"github.com/jxdones/stoat/internal/database"
	"github.com/jxdones/stoat/internal/ui/components/table"
)

func TestParseColumnFilterExpression(t *testing.T) {
	tests := []struct {
		name       string
		expr       string
		wantColumn string
		wantValue  string
		wantQuoted bool
		wantOk     bool
	}{
		{
			name:   "no_equals_sign_returns_not_ok",
			expr:   "john",
			wantOk: false,
		},
		{
			name:   "empty_column_returns_not_ok",
			expr:   "= value",
			wantOk: false,
		},
		{
			name:       "unquoted_value",
			expr:       "id = 3",
			wantColumn: "id",
			wantValue:  "3",
			wantQuoted: false,
			wantOk:     true,
		},
		{
			name:       "single_quoted_value",
			expr:       "name = 'John Doe'",
			wantColumn: "name",
			wantValue:  "John Doe",
			wantQuoted: true,
			wantOk:     true,
		},
		{
			name:       "double_quoted_value",
			expr:       `name = "John Doe"`,
			wantColumn: "name",
			wantValue:  "John Doe",
			wantQuoted: true,
			wantOk:     true,
		},
		{
			name:       "equals_sign_inside_quoted_value",
			expr:       `name = "John = Doe"`,
			wantColumn: "name",
			wantValue:  "John = Doe",
			wantQuoted: true,
			wantOk:     true,
		},
		{
			name:       "extra_whitespace_around_equals",
			expr:       "id  =  3",
			wantColumn: "id",
			wantValue:  "3",
			wantQuoted: false,
			wantOk:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			col, val, quoted, ok := parseColumnFilterExpression(tt.expr)
			if ok != tt.wantOk {
				t.Errorf("ok = %v, want %v", ok, tt.wantOk)
			}
			if col != tt.wantColumn {
				t.Errorf("column = %q, want %q", col, tt.wantColumn)
			}
			if val != tt.wantValue {
				t.Errorf("value = %q, want %q", val, tt.wantValue)
			}
			if quoted != tt.wantQuoted {
				t.Errorf("quoted = %v, want %v", quoted, tt.wantQuoted)
			}
		})
	}
}

func TestFilterRowsByExpression(t *testing.T) {
	columns := []table.Column{
		{Key: "id", Title: "id", Type: "integer"},
		{Key: "name", Title: "name", Type: "text"},
	}
	rows := []table.Row{
		{"id": "1", "name": "Alice"},
		{"id": "2", "name": "Bob"},
		{"id": "10", "name": "charlie"},
	}

	tests := []struct {
		name     string
		expr     string
		wantKeys []string // expected "id" values of matched rows
	}{
		{
			name:     "empty_expression_returns_all_rows",
			expr:     "",
			wantKeys: []string{"1", "2", "10"},
		},
		{
			name:     "plain_substring_match",
			expr:     "alice",
			wantKeys: []string{"1"},
		},
		{
			name:     "plain_substring_case_insensitive",
			expr:     "ALICE",
			wantKeys: []string{"1"},
		},
		{
			name:     "column_exact_match_hit",
			expr:     "id = 1",
			wantKeys: []string{"1"},
		},
		{
			name:     "column_exact_match_case_insensitive",
			expr:     "name = ALICE",
			wantKeys: []string{"1"},
		},
		{
			name:     "column_exact_match_miss",
			expr:     "id = 99",
			wantKeys: []string{},
		},
		{
			name:     "column_exact_does_not_match_partial",
			expr:     "id = 1",
			wantKeys: []string{"1"}, // must not include id=10
		},
		{
			name:     "column_substring_match",
			expr:     `name = "Ali"`,
			wantKeys: []string{"1"},
		},
		{
			name:     "column_substring_match_is_case_sensitive",
			expr:     `name = "ali"`,
			wantKeys: []string{},
		},
		{
			name:     "unknown_column_falls_back_to_plain_search",
			expr:     "charlie",
			wantKeys: []string{"10"},
		},
		{
			name:     "unknown_column_with_equals_returns_empty",
			expr:     "foo = charlie",
			wantKeys: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterRowsByExpression(rows, columns, tt.expr)
			if len(got) != len(tt.wantKeys) {
				t.Fatalf("got %d rows, want %d", len(got), len(tt.wantKeys))
			}
			for i, row := range got {
				if row["id"] != tt.wantKeys[i] {
					t.Errorf("row[%d] id = %q, want %q", i, row["id"], tt.wantKeys[i])
				}
			}
		})
	}
}

func TestHandleFilterKey(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(*Model)
		filterValue   string
		wantHandled   bool
		wantRowCount  int
		wantStatusMsg string
	}{
		{
			name:        "wrong_focus_returns_unhandled",
			setup:       func(m *Model) { m.view.focus = FocusTable },
			filterValue: "Alice",
			wantHandled: false,
		},
		{
			name:          "no_connection_returns_warning",
			setup:         func(m *Model) { m.source = nil },
			filterValue:   "Alice",
			wantHandled:   true,
			wantStatusMsg: "No active connection",
		},
		{
			name: "no_table_selected_returns_warning",
			setup: func(m *Model) {
				m.sidebar.SetDatabases([]string{})
			},
			filterValue:   "Alice",
			wantHandled:   true,
			wantStatusMsg: "No table selected",
		},
		{
			name:          "invalid_column_filter_missing_value_returns_warning",
			filterValue:   "name =",
			wantHandled:   true,
			wantStatusMsg: "Invalid filter",
		},
		{
			name:         "plain_filter_matches_rows",
			filterValue:  "alice",
			wantHandled:  true,
			wantRowCount: 1,
		},
		{
			name:         "column_exact_filter_matches_row",
			filterValue:  "id = 2",
			wantHandled:  true,
			wantRowCount: 1,
		},
		{
			name:        "refilter_works_against_unfiltered_rows_not_previous_result",
			filterValue: "id = 2",
			setup: func(m *Model) {
				// simulate a previous filter having reduced the table to one row
				m.table.SetRows(dbRowsToTable([]database.Row{{"id": "1", "name": "Alice"}}))
			},
			wantHandled:  true,
			wantRowCount: 1, // must find id=2 from unfilteredRows, not from the reduced table
		},
		{
			name:        "clearing_filter_on_query_result_restores_full_result",
			filterValue: "",
			setup: func(m *Model) {
				m.viewingQueryResult = true
				// simulate a previous filter having reduced the table to one row
				m.table.SetRows(dbRowsToTable([]database.Row{{"id": "1", "name": "Alice"}}))
			},
			wantHandled:  true,
			wantRowCount: 3, // must restore all unfilteredRows, not reload from DB
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := modelWithFilterboxFocusAndData()
			if tt.setup != nil {
				tt.setup(&m)
			}
			m.filterbox.SetValue(tt.filterValue)
			got, _, handled := m.handleFilterKey(keyMsg("enter"))
			next := got.(Model)
			if handled != tt.wantHandled {
				t.Errorf("handled = %v, want %v", handled, tt.wantHandled)
			}
			if tt.wantStatusMsg != "" && !strings.Contains(statusText(next), tt.wantStatusMsg) {
				t.Errorf("status %q does not contain %q", statusText(next), tt.wantStatusMsg)
			}
			if tt.wantRowCount > 0 && len(next.table.Rows()) != tt.wantRowCount {
				t.Errorf("row count = %d, want %d", len(next.table.Rows()), tt.wantRowCount)
			}
		})
	}
}
