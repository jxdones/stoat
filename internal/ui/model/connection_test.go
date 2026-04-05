package model

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/jxdones/stoat/internal/database"
)

func TestHandleConnectionFailed(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantMsg string
	}{
		{
			name:    "shows_error_in_status_bar",
			err:     errors.New("connection refused"),
			wantMsg: "Connection failed",
		},
		{
			name:    "includes_error_detail",
			err:     errors.New("timeout"),
			wantMsg: "timeout",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			next, cmd := m.onConnectionFailed(ConnectionFailedMsg{connectionSeq: m.connectionSeq, err: tt.err})
			got := next.(Model)
			if cmd != nil {
				t.Errorf("onConnectionFailed() cmd = %v, want nil", cmd)
			}
			if !strings.Contains(statusText(got), tt.wantMsg) {
				t.Errorf("status %q does not contain %q", statusText(got), tt.wantMsg)
			}
		})
	}
}

func TestOnConnected(t *testing.T) {
	tests := []struct {
		name                string
		connectionSeq       int
		preloadTable        bool
		connectionName      string
		wantSourceSet       bool
		wantStatusSubstring string
		wantCmd             bool
		wantTableCleared    bool
	}{
		{
			name:                "sets_source_and_triggers_parallel_load",
			connectionSeq:       0,
			wantSourceSet:       true,
			wantStatusSubstring: "Loading tables",
			wantCmd:             true,
		},
		{
			name:                "clears_prior_table_data",
			connectionSeq:       1,
			preloadTable:        true,
			connectionName:      "local",
			wantSourceSet:       true,
			wantStatusSubstring: "Loading tables",
			wantCmd:             true,
			wantTableCleared:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			m.connectionSeq = tt.connectionSeq
			if tt.preloadTable {
				m.table.SetColumns(dbColumnsToTable([]database.Column{
					{Key: "name", Title: "name", Type: "text", MinWidth: 4},
				}))
				rows := dbRowsToTable([]database.Row{{"name": "Alice"}})
				m.unfilteredRows = rows
				m.table.SetRows(rows)
			}

			next, cmd := m.onConnected(ConnectedMsg{
				source:        mockDataSource{},
				connectionSeq: tt.connectionSeq,
				name:          tt.connectionName,
				readOnly:      false,
			})
			got := next.(Model)
			if tt.wantSourceSet && !got.HasConnection() {
				t.Error("onConnected() source not set on model")
			}
			if !strings.Contains(statusText(got), tt.wantStatusSubstring) {
				t.Errorf("status %q does not contain %q", statusText(got), tt.wantStatusSubstring)
			}
			if tt.wantCmd && cmd == nil {
				t.Error("onConnected() cmd = nil, want follow-up load command")
			}
			if tt.wantTableCleared {
				if got.table.ColumnCount() != 0 || got.table.RowCount() != 0 {
					t.Errorf("onConnected() table should be cleared, got %d columns and %d rows",
						got.table.ColumnCount(), got.table.RowCount())
				}
				if got.unfilteredRows != nil {
					t.Errorf("onConnected() unfilteredRows = %#v, want nil", got.unfilteredRows)
				}
			}
		})
	}
}

func TestStaleConnectionSeq_ignoredByConnectionHandlers(t *testing.T) {
	tests := []struct {
		name   string
		act    func(m Model) (tea.Model, tea.Cmd)
		assert func(t *testing.T, got Model, cmd tea.Cmd)
	}{
		{
			name: "connection_failed",
			act: func(m Model) (tea.Model, tea.Cmd) {
				return m.onConnectionFailed(ConnectionFailedMsg{
					connectionSeq: 1,
					err:           errors.New("stale_failure_marker"),
				})
			},
			assert: func(t *testing.T, got Model, cmd tea.Cmd) {
				if cmd != nil {
					t.Errorf("cmd = %v, want nil", cmd)
				}
				if strings.Contains(statusText(got), "stale_failure_marker") {
					t.Errorf("stale ConnectionFailedMsg should not update status, got %q", statusText(got))
				}
			},
		},
		{
			name: "connected",
			act: func(m Model) (tea.Model, tea.Cmd) {
				return m.onConnected(ConnectedMsg{
					source:        mockDataSource{},
					connectionSeq: 1,
					name:          "x",
					readOnly:      false,
				})
			},
			assert: func(t *testing.T, got Model, cmd tea.Cmd) {
				if cmd != nil {
					t.Errorf("cmd = %v, want nil", cmd)
				}
				if got.HasConnection() {
					t.Error("stale ConnectedMsg should not set source")
				}
			},
		},
		{
			name: "connecting",
			act: func(m Model) (tea.Model, tea.Cmd) {
				return m.onConnecting(ConnectingMsg{
					cfg:           database.Config{Name: "db"},
					connectionSeq: 1,
				})
			},
			assert: func(t *testing.T, got Model, cmd tea.Cmd) {
				if cmd != nil {
					t.Errorf("cmd = %v, want nil", cmd)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			m.connectionSeq = 2
			next, cmd := tt.act(m)
			tt.assert(t, next.(Model), cmd)
		})
	}
}

func TestRedactSecret(t *testing.T) {
	tests := []struct {
		name         string
		dsn          string
		wantRedacted string
	}{
		{
			name:         "standard_postgres_dsn",
			dsn:          "postgres://user:password@host:5432/dbname",
			wantRedacted: "postgres://user:[redacted]@host:5432/dbname",
		},
		{
			name:         "standard_mysql_dsn",
			dsn:          "mysql://user:password@host:3306/dbname",
			wantRedacted: "mysql://user:[redacted]@host:3306/dbname",
		},
		{
			name:         "dsn_embedded_in_message",
			dsn:          "dial error: postgres://user:password@host:5432/dbname: connection refused",
			wantRedacted: "dial error: postgres://user:[redacted]@host:5432/dbname: connection refused",
		},
		{
			name:         "dsn_with_complex_password",
			dsn:          "mysql://user:p@$$w0rd!@host:3306/dbname",
			wantRedacted: "mysql://user:[redacted]@host:3306/dbname",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := redactSecret(tt.dsn)
			if got != tt.wantRedacted {
				t.Errorf("redactSecret(%q) = %q, want %q", tt.dsn, got, tt.wantRedacted)
			}
		})
	}
}
