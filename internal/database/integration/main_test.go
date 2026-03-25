//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/jxdones/stoat/internal/database"
	"github.com/jxdones/stoat/internal/database/provider"
	"github.com/testcontainers/testcontainers-go"
	tcmysql "github.com/testcontainers/testcontainers-go/modules/mysql"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	pgConn    database.Connection
	mysqlConn database.Connection
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	setupPostgres(ctx)
	setupMySQL(ctx)

	m.Run()
}

func setupPostgres(ctx context.Context) {
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
}

func setupMySQL(ctx context.Context) {
	mysqlContainer, err := tcmysql.Run(ctx,
		"mysql:8",
		tcmysql.WithDatabase("stoat_test"),
		tcmysql.WithUsername("stoat"),
		tcmysql.WithPassword("stoat"),
	)
	if err != nil {
		panic(err)
	}

	dsn, err := mysqlContainer.ConnectionString(ctx)
	if err != nil {
		panic(err)
	}

	mysqlConn, err = provider.FromConfig(database.Config{
		DBMS: database.DBMSMySQL,
		Values: map[string]string{
			"dsn":      dsn,
			"database": "stoat_test",
		},
	})
	if err != nil {
		panic(err)
	}

	// MySQL driver does not support multiple statements in a single call — seed one at a time.
	statements := []string{
		`CREATE TABLE users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			username VARCHAR(255) NOT NULL UNIQUE,
			email VARCHAR(255) NOT NULL UNIQUE
		)`,
		`CREATE TABLE habits (
			id INT AUTO_INCREMENT PRIMARY KEY,
			user_id INT NOT NULL,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)`,
		`CREATE TABLE habit_logs (
			id INT AUTO_INCREMENT PRIMARY KEY,
			habit_id INT NOT NULL,
			logged_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (habit_id) REFERENCES habits(id)
		)`,
		`INSERT INTO users (username, email) VALUES ('alice', 'alice@example.com'), ('bob', 'bob@example.com')`,
		`INSERT INTO habits (user_id, name, description) VALUES (1, 'Morning run', 'Run 5km every morning'), (1, 'Read', NULL), (2, 'Meditate', '10 minutes daily')`,
		`INSERT INTO habit_logs (habit_id, logged_at) VALUES (1, '2026-01-01 07:00:00'), (1, '2026-01-02 07:15:00'), (2, '2026-01-01 08:00:00'), (3, '2026-01-01 06:30:00')`,
	}
	for _, stmt := range statements {
		if _, err = mysqlConn.Query(ctx, stmt); err != nil {
			panic(err)
		}
	}
}
