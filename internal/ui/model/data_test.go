package model

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/jxdones/stoat/internal/database"
)

func TestHandleDatabasesLoaded(t *testing.T) {
	tests := []struct {
		name           string
		databases      []string
		wantStatusText string
		wantCmd        bool
	}{
		{
			name:           "empty_list_sets_ready",
			databases:      []string{},
			wantStatusText: "Ready",
			wantCmd:        false,
		},
		{
			name:           "non_empty_list_populates_sidebar",
			databases:      []string{"mydb"},
			wantStatusText: "Ready",
			wantCmd:        false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			m.source = mockDataSource{}
			next, cmd := m.onDatabasesLoaded(DatabasesLoadedMsg{
				Databases:     tt.databases,
				ConnectionSeq: m.connectionSeq,
			})
			got := next.(Model)
			if !strings.Contains(statusText(got), tt.wantStatusText) {
				t.Errorf("status %q does not contain %q", statusText(got), tt.wantStatusText)
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

func TestHandleTablesLoaded(t *testing.T) {
	tests := []struct {
		name           string
		tables         []string
		err            error
		wantStatusText string
	}{
		{
			name:           "success_sets_ready",
			tables:         []string{"users", "posts"},
			wantStatusText: "Ready",
		},
		{
			name:           "empty_tables_still_sets_ready",
			tables:         []string{},
			wantStatusText: "Ready",
		},
		{
			name:           "error_shows_in_status",
			err:            errors.New("permission denied"),
			wantStatusText: "permission denied",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			next, _ := m.onTablesLoaded(TablesLoadedMsg{
				Database:      "mydb",
				Tables:        tt.tables,
				Err:           tt.err,
				ConnectionSeq: m.connectionSeq,
			})
			got := next.(Model)
			if !strings.Contains(statusText(got), tt.wantStatusText) {
				t.Errorf("status %q does not contain %q", statusText(got), tt.wantStatusText)
			}
		})
	}
}

func TestStaleConnectionSeq_ignoredByDataHandlers(t *testing.T) {
	target := database.DatabaseTarget{Database: "mydb", Table: "users"}
	tests := []struct {
		name   string
		setup  func() Model
		act    func(m Model) (tea.Model, tea.Cmd)
		assert func(t *testing.T, got Model, cmd tea.Cmd)
	}{
		{
			name: "databases_loaded",
			setup: func() Model {
				m := New()
				m.source = mockDataSource{}
				m.connectionSeq = 2
				m.sidebar.SetDatabases([]string{"alpha", "beta"})
				m.sidebar.OpenSelectedDatabase()
				return m
			},
			act: func(m Model) (tea.Model, tea.Cmd) {
				return m.onDatabasesLoaded(DatabasesLoadedMsg{
					Databases:     []string{"only"},
					ConnectionSeq: 1,
				})
			},
			assert: func(t *testing.T, got Model, cmd tea.Cmd) {
				if cmd != nil {
					t.Errorf("cmd = %v, want nil", cmd)
				}
				if got.sidebar.SelectedDB() != "alpha" {
					t.Errorf("SelectedDB() = %q, want unchanged alpha", got.sidebar.SelectedDB())
				}
			},
		},
		{
			name: "tables_loaded",
			setup: func() Model {
				m := New()
				m.connectionSeq = 2
				m.sidebar.SetDatabases([]string{"mydb"})
				m.sidebar.SetTables("mydb", []string{"users", "posts"})
				m.sidebar.OpenSelectedDatabase()
				return m
			},
			act: func(m Model) (tea.Model, tea.Cmd) {
				return m.onTablesLoaded(TablesLoadedMsg{
					Database:      "mydb",
					Tables:        []string{"other"},
					ConnectionSeq: 1,
				})
			},
			assert: func(t *testing.T, got Model, cmd tea.Cmd) {
				if cmd != nil {
					t.Errorf("cmd = %v, want nil", cmd)
				}
				if got.sidebar.SelectedTable() != "users" {
					t.Errorf("SelectedTable() = %q, want unchanged users", got.sidebar.SelectedTable())
				}
			},
		},
		{
			name: "rows_loaded",
			setup: func() Model {
				m := modelWithTableFocusAndData()
				m.connectionSeq = 2
				return m
			},
			act: func(m Model) (tea.Model, tea.Cmd) {
				return m.onRowsLoaded(RowsLoadedMsg{
					ConnectionSeq: 1,
					Result: database.PageResult{
						Result: database.QueryResult{
							Columns: []database.Column{{Key: "z", Title: "z", Type: "text"}},
							Rows:    []database.Row{{"z": "9"}},
						},
					},
				})
			},
			assert: func(t *testing.T, got Model, cmd tea.Cmd) {
				if cmd != nil {
					t.Errorf("cmd = %v, want nil", cmd)
				}
				if got.table.ColumnCount() != 1 || got.table.RowCount() != 1 {
					t.Errorf("table columns/rows = %d/%d, want 1/1 unchanged",
						got.table.ColumnCount(), got.table.RowCount())
				}
			},
		},
		{
			name: "table_constraints_loaded",
			setup: func() Model {
				m := New()
				m.connectionSeq = 2
				m.tablePKColumns = []string{"id"}
				return m
			},
			act: func(m Model) (tea.Model, tea.Cmd) {
				return m.onTableConstraintsLoaded(TableConstraintsLoadedMsg{
					Target:        target,
					Constraints:   []database.Constraint{{Type: "PRIMARY KEY", Columns: []string{"other_id"}}},
					ConnectionSeq: 1,
				})
			},
			assert: func(t *testing.T, got Model, cmd tea.Cmd) {
				if cmd != nil {
					t.Errorf("cmd = %v, want nil", cmd)
				}
				if len(got.tablePKColumns) != 1 || got.tablePKColumns[0] != "id" {
					t.Errorf("tablePKColumns = %#v, want [id]", got.tablePKColumns)
				}
			},
		},
		{
			name: "indexes_loaded",
			setup: func() Model {
				m := New()
				m.connectionSeq = 2
				m.sidebar.SetDatabases([]string{"mydb"})
				m.sidebar.SetTables("mydb", []string{"users"})
				m.sidebar.OpenSelectedDatabase()
				m.tableSchema.indexes = []database.Index{{Name: "keep_idx"}}
				return m
			},
			act: func(m Model) (tea.Model, tea.Cmd) {
				return m.onIndexesLoaded(IndexesLoadedMsg{
					Target:        target,
					Indexes:       []database.Index{{Name: "new_idx"}},
					ConnectionSeq: 1,
				})
			},
			assert: func(t *testing.T, got Model, cmd tea.Cmd) {
				if cmd != nil {
					t.Errorf("cmd = %v, want nil", cmd)
				}
				if len(got.tableSchema.indexes) != 1 || got.tableSchema.indexes[0].Name != "keep_idx" {
					t.Errorf("indexes = %#v, want keep_idx unchanged", got.tableSchema.indexes)
				}
			},
		},
		{
			name: "foreign_keys_loaded",
			setup: func() Model {
				m := New()
				m.connectionSeq = 2
				m.sidebar.SetDatabases([]string{"mydb"})
				m.sidebar.SetTables("mydb", []string{"users"})
				m.sidebar.OpenSelectedDatabase()
				m.tableSchema.foreignKeys = []database.ForeignKey{{Name: "fk_keep"}}
				return m
			},
			act: func(m Model) (tea.Model, tea.Cmd) {
				return m.onForeignKeysLoaded(ForeignKeysLoadedMsg{
					Target:        target,
					ForeignKeys:   []database.ForeignKey{{Name: "fk_new"}},
					ConnectionSeq: 1,
				})
			},
			assert: func(t *testing.T, got Model, cmd tea.Cmd) {
				if cmd != nil {
					t.Errorf("cmd = %v, want nil", cmd)
				}
				if len(got.tableSchema.foreignKeys) != 1 || got.tableSchema.foreignKeys[0].Name != "fk_keep" {
					t.Errorf("foreignKeys = %#v, want fk_keep unchanged", got.tableSchema.foreignKeys)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.setup()
			next, cmd := tt.act(m)
			tt.assert(t, next.(Model), cmd)
		})
	}
}
