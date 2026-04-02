package model

import (
	"errors"
	"strings"
	"testing"
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
			next, cmd := m.onConnectionFailed(ConnectionFailedMsg{err: tt.err})
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

func TestHandleConnected(t *testing.T) {
	tests := []struct {
		name           string
		wantSourceSet  bool
		wantStatusText string
		wantCmd        bool
	}{
		{
			name:           "sets_source_and_triggers_parallel_load",
			wantSourceSet:  true,
			wantStatusText: "Loading tables",
			wantCmd:        true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			next, cmd := m.onConnected(ConnectedMsg{source: mockDataSource{}})
			got := next.(Model)
			if tt.wantSourceSet && !got.HasConnection() {
				t.Error("onConnected() source not set on model")
			}
			if !strings.Contains(statusText(got), tt.wantStatusText) {
				t.Errorf("status %q does not contain %q", statusText(got), tt.wantStatusText)
			}
			if tt.wantCmd && cmd == nil {
				t.Error("onConnected() cmd = nil, want LoadDatabasesCmd")
			}
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
