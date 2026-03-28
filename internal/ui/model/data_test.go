package model

import (
	"errors"
	"strings"
	"testing"
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
			next, cmd := m.onDatabasesLoaded(DatabasesLoadedMsg{Databases: tt.databases})
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
			next, _ := m.onTablesLoaded(TablesLoadedMsg{Database: "mydb", Tables: tt.tables, Err: tt.err})
			got := next.(Model)
			if !strings.Contains(statusText(got), tt.wantStatusText) {
				t.Errorf("status %q does not contain %q", statusText(got), tt.wantStatusText)
			}
		})
	}
}
