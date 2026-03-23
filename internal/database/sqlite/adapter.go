package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"

	"github.com/jxdones/stoat/internal/database"
	_ "github.com/mattn/go-sqlite3"
)

// connection implements the database.Connection interface for SQLite.
type connection struct {
	name string
	path string
	db   *sql.DB
}

// NewConnection creates a new SQLite database connection from the given configuration.
func NewConnection(config database.Config) (database.Connection, error) {
	if config.DBMS != database.DBMSSQLite {
		if strings.TrimSpace(string(config.DBMS)) == "" {
			return nil, database.ErrInvalidConfig
		}
		return nil, database.ErrNotSupported
	}
	path := strings.TrimSpace(config.Values["path"])
	if path == "" {
		return nil, database.ErrInvalidConfig
	}
	dsn := path
	if config.ReadOnly {
		dsn = "file:" + path + "?mode=ro"
	}
	dbConn, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	if err := dbConn.PingContext(context.Background()); err != nil {
		dbConn.Close()
		return nil, err
	}

	return &connection{
		name: strings.TrimSpace(config.Name),
		path: path,
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

// Databases returns the list of database names in the given path.
func (c *connection) Databases(ctx context.Context) ([]string, error) {
	return Databases(ctx, c.name, c.path)
}

// DatabaseLabel returns the label for the database.
func (c *connection) DatabaseLabel() string {
	return "Databases"
}

// Tables returns the list of table names in the given database.
func (c *connection) Tables(ctx context.Context, db string) ([]string, error) {
	return Tables(ctx, c.db)
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
func (c *connection) DefaultDatabase(_ context.Context) (string, error) {
	if c.name != "" {
		return c.name, nil
	}
	return filepath.Base(c.path), nil
}

// UsesSchemaQualification reports that SQLite write queries do not prefix table
// names with a schema. SQLite has no named schema concept in generated queries.
func (c *connection) UsesSchemaQualification() bool {
	return false
}
