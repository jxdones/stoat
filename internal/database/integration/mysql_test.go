//go:build integration

package integration_test

import (
	"testing"

	"github.com/jxdones/stoat/internal/database"
)

func TestMySQLDatabases(t *testing.T) {
	testDatabases(t, mysqlConn, "stoat_test")
}

func TestMySQLTables(t *testing.T) {
	testTables(t, mysqlConn, "stoat_test", []string{"users", "habits", "habit_logs"})
}

func TestMySQLRows(t *testing.T) {
	tests := []struct {
		table         string
		expectedCount int
	}{
		{"users", 2},
		{"habits", 3},
		{"habit_logs", 4},
	}

	for _, tt := range tests {
		t.Run(tt.table, func(t *testing.T) {
			testRows(t, mysqlConn, database.DatabaseTarget{Database: "stoat_test", Table: tt.table}, tt.expectedCount)
		})
	}
}

func TestMySQLForeignKeys(t *testing.T) {
	tests := []struct {
		table         string
		expectedCount int
	}{
		{"users", 0},
		{"habits", 1},
		{"habit_logs", 1},
	}

	for _, tt := range tests {
		t.Run(tt.table, func(t *testing.T) {
			testForeignKeys(t, mysqlConn, database.DatabaseTarget{Database: "stoat_test", Table: tt.table}, tt.expectedCount)
		})
	}
}

func TestMySQLIndexes(t *testing.T) {
	tests := []struct {
		table         string
		expectedCount int
	}{
		{"users", 3},
		{"habits", 2},
		{"habit_logs", 2},
	}

	for _, tt := range tests {
		t.Run(tt.table, func(t *testing.T) {
			testIndexes(t, mysqlConn, database.DatabaseTarget{Database: "stoat_test", Table: tt.table}, tt.expectedCount)
		})
	}
}

func TestMySQLQuery(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		wantMinRows    int
		wantMinColumns int
	}{
		{
			name:           "explain_returns_rows",
			query:          "EXPLAIN SELECT * FROM users",
			wantMinRows:    1,
			wantMinColumns: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testQuery(t, mysqlConn, tt.query, tt.wantMinRows, tt.wantMinColumns)
		})
	}
}

func TestMySQLConstraints(t *testing.T) {
	tests := []struct {
		table         string
		expectedCount int
	}{
		{"users", 3},
		{"habits", 1},
		{"habit_logs", 1},
	}

	for _, tt := range tests {
		t.Run(tt.table, func(t *testing.T) {
			testConstraints(t, mysqlConn, database.DatabaseTarget{Database: "stoat_test", Table: tt.table}, tt.expectedCount)
		})
	}
}
