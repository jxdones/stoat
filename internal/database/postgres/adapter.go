package postgres

import (
	"context"
	"database/sql"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jxdones/stoat/internal/database"
)

// connection implements the database.Connection interface for PostgreSQL.
type connection struct {
	name string
	dsn  string
	db   *sql.DB
}

// NewConnection creates a new PostgreSQL database connection from the given configuration.
// config.Values must contain a "dsn" key with a valid PostgreSQL connection string.
// Optional SSL fields: "sslmode", "sslcert", "sslkey", "sslrootcert".
func NewConnection(config database.Config) (database.Connection, error) {
	if config.DBMS != database.DBMSPostgres {
		if strings.TrimSpace(string(config.DBMS)) == "" {
			return nil, database.ErrInvalidConfig
		}
		return nil, database.ErrNotSupported
	}

	dsn := strings.TrimSpace(config.Values["dsn"])
	if dsn == "" {
		return nil, database.ErrInvalidConfig
	}

	dbConn, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	if err := dbConn.PingContext(context.Background()); err != nil {
		dbConn.Close()
		return nil, err
	}

	if config.ReadOnly {
		readOnlyQuery := "SET SESSION CHARACTERISTICS AS TRANSACTION READ ONLY"
		if _, err := dbConn.ExecContext(context.Background(), readOnlyQuery); err != nil {
			dbConn.Close()
			return nil, err
		}
	}

	return &connection{
		name: strings.TrimSpace(config.Name),
		dsn:  dsn,
		db:   dbConn,
	}, nil
}

// Close closes the connection.
func (c *connection) Close() error {
	if c.db == nil {
		return nil
	}
	err := c.db.Close()
	c.db = nil
	return err
}

// Databases returns the list of schemas in the PostgreSQL database.
func (c *connection) Databases(ctx context.Context) ([]string, error) {
	return Schemas(ctx, c.db)
}

// DatabaseLabel returns the label for the database.
func (c *connection) DatabaseLabel() string {
	return "Schemas"
}

// Tables returns the list of table names in the given schema.
func (c *connection) Tables(ctx context.Context, schema string) ([]string, error) {
	return Tables(ctx, c.db, schema)
}

// Rows returns a page of rows for the given table.
func (c *connection) Rows(ctx context.Context, target database.DatabaseTarget, page database.PageRequest) (database.PageResult, error) {
	return Rows(ctx, c.db, target, page)
}

// Query executes a query and returns the result.
func (c *connection) Query(ctx context.Context, query string) (database.QueryResult, error) {
	return Query(ctx, c.db, query)
}

// Indexes returns the list of indexes on the given table.
func (c *connection) Indexes(ctx context.Context, target database.DatabaseTarget) ([]database.Index, error) {
	return Indexes(ctx, c.db, target)
}

// Constraints returns the list of constraints on the given table.
func (c *connection) Constraints(ctx context.Context, target database.DatabaseTarget) ([]database.Constraint, error) {
	return Constraints(ctx, c.db, target)
}

// ForeignKeys returns the list of foreign keys on the given table.
func (c *connection) ForeignKeys(ctx context.Context, target database.DatabaseTarget) ([]database.ForeignKey, error) {
	return ForeignKeys(ctx, c.db, target)
}

// DefaultDatabase returns the default database name.
func (c *connection) DefaultDatabase(ctx context.Context) (string, error) {
	rows, err := c.db.QueryContext(ctx, "SELECT current_schema()")
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var schemaName string
	if rows.Next() {
		err := rows.Scan(&schemaName)
		if err != nil {
			return "", err
		}
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	return schemaName, nil
}

// UsesSchemaQualification reports that Postgres write queries must prefix table
// names with the active schema (e.g. "public"."users").
func (c *connection) UsesSchemaQualification() bool {
	return true
}
