package mysql

import (
	"context"
	"database/sql"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jxdones/stoat/internal/database"
)

// connection implements the database.Connection interface for MySQL.
type connection struct {
	name         string
	dsn          string
	databaseName string
	db           *sql.DB
}

// NewConnection creates a new MySQL database connection from the given configuration.
// config.Values must contain a "dsn" key with a valid MySQL connection string.
// Optional TLS mode: "false", "skip-verify", "true".
func NewConnection(config database.Config) (database.Connection, error) {
	if config.DBMS != database.DBMSMySQL {
		if strings.TrimSpace(string(config.DBMS)) == "" {
			return nil, database.ErrInvalidConfig
		}
		return nil, database.ErrNotSupported
	}

	dsn := strings.TrimSpace(config.Values["dsn"])
	if dsn == "" {
		return nil, database.ErrInvalidConfig
	}

	dbConn, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	if err := dbConn.PingContext(context.Background()); err != nil {
		dbConn.Close()
		return nil, err
	}

	if config.ReadOnly {
		readOnlyQuery := "SET SESSION TRANSACTION READ ONLY"
		if _, err := dbConn.ExecContext(context.Background(), readOnlyQuery); err != nil {
			dbConn.Close()
			return nil, err
		}
	}

	dbName := strings.TrimSpace(config.Values["database"])
	if dbName == "" {
		return nil, database.ErrInvalidDatabase
	}

	return &connection{
		name:         strings.TrimSpace(config.Name),
		dsn:          dsn,
		databaseName: dbName,
		db:           dbConn,
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

// Databases returns the list of databases in the MySQL database.
func (c *connection) Databases(ctx context.Context) ([]string, error) {
	return Databases(ctx, c.db)
}

// DatabaseLabel returns the label for the database.
func (c *connection) DatabaseLabel() string {
	return "Databases"
}

// Tables returns the list of table names in the given database.
func (c *connection) Tables(ctx context.Context, databaseName string) ([]string, error) {
	return Tables(ctx, c.db, databaseName)
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
	return c.databaseName, nil
}

// UsesSchemaQualification reports that MySQL write queries must prefix table
// names with the active database (e.g. "database"."users").
func (c *connection) UsesSchemaQualification() bool {
	return true
}
