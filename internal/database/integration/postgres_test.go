//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/jxdones/stoat/internal/database"
	"github.com/jxdones/stoat/internal/database/provider"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// pgConn is the PostgreSQL connection for integration tests.
var pgConn database.Connection

func TestMain(m *testing.M) {
	ctx := context.Background()
	pgContainer, err := tcpostgres.Run(ctx,
		"postgres:17",
		tcpostgres.WithDatabase("stoat_test"),
		tcpostgres.WithUsername("stoat"),
		tcpostgres.WithPassword("stoat"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		),
	)
	if err != nil {
		panic(err)
	}
	defer pgContainer.Terminate(ctx)

	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic(err)
	}

	pgConn, err = provider.FromConfig(database.Config{
		DBMS: database.DBMSPostgres,
		Values: map[string]string{
			"dsn": dsn,
		},
	})
	if err != nil {
		panic(err)
	}
	defer pgConn.Close()

	seed := `
	CREATE TABLE users (
		id SERIAL PRIMARY KEY,
		username TEXT NOT NULL UNIQUE,
		email TEXT NOT NULL UNIQUE
	);
  
	CREATE TABLE habits (
		id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(id),
		name TEXT NOT NULL,
		description TEXT
	);
  
	CREATE TABLE habit_logs (
		id SERIAL PRIMARY KEY,
		habit_id INTEGER NOT NULL REFERENCES habits(id),
		logged_at TIMESTAMP NOT NULL DEFAULT NOW()
	);
  
	INSERT INTO users (username, email) VALUES
		('alice', 'alice@example.com'),
		('bob', 'bob@example.com');
  
	INSERT INTO habits (user_id, name, description) VALUES
		(1, 'Morning run', 'Run 5km every morning'),
		(1, 'Read', NULL),
		(2, 'Meditate', '10 minutes daily');
  
	INSERT INTO habit_logs (habit_id, logged_at) VALUES
		(1, '2026-01-01 07:00:00'),
		(1, '2026-01-02 07:15:00'),
		(2, '2026-01-01 08:00:00'),
		(3, '2026-01-01 06:30:00');
	`

	if _, err = pgConn.Query(ctx, seed); err != nil {
		panic(err)
	}

	m.Run()
}

func TestPostgresDatabases(t *testing.T) {
	testDatabases(t, pgConn, "public")
}

func TestPostgresTables(t *testing.T) {
	testTables(t, pgConn, "public", []string{"users", "habits", "habit_logs"})
}

func TestPostgresRows(t *testing.T) {
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
			testRows(t, pgConn, database.DatabaseTarget{Database: "public", Table: tt.table}, tt.expectedCount)
		})
	}
}

func TestPostgresForeignKeys(t *testing.T) {
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
			testForeignKeys(t, pgConn, database.DatabaseTarget{Database: "public", Table: tt.table}, tt.expectedCount)
		})
	}
}

func TestPostgresIndexes(t *testing.T) {
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
			testIndexes(t, pgConn, database.DatabaseTarget{Database: "public", Table: tt.table}, tt.expectedCount)
		})
	}
}

func TestPostgresQuery(t *testing.T) {
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
		{
			name:           "explain_analyze_returns_rows",
			query:          "EXPLAIN ANALYZE SELECT * FROM users",
			wantMinRows:    1,
			wantMinColumns: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testQuery(t, pgConn, tt.query, tt.wantMinRows, tt.wantMinColumns)
		})
	}
}

func TestPostgresConstraints(t *testing.T) {
	tests := []struct {
		table         string
		expectedCount int
	}{
		{"users", 7},
		{"habits", 5},
		{"habit_logs", 6},
	}

	for _, tt := range tests {
		t.Run(tt.table, func(t *testing.T) {
			testConstraints(t, pgConn, database.DatabaseTarget{Database: "public", Table: tt.table}, tt.expectedCount)
		})
	}
}
