package datasource

import (
	"context"

	"github.com/jxdones/stoat/internal/database"
)

// DataSource is the abstraction the UI model uses to load data.
type DataSource interface {
	Databases(ctx context.Context) ([]string, error)
	DatabaseLabel() string
	Tables(ctx context.Context, database string) ([]string, error)
	Rows(ctx context.Context, target database.DatabaseTarget, page database.PageRequest) (database.PageResult, error)
	Query(ctx context.Context, query string) (database.QueryResult, error)
	Indexes(ctx context.Context, target database.DatabaseTarget) ([]database.Index, error)
	Constraints(ctx context.Context, target database.DatabaseTarget) ([]database.Constraint, error)
	ForeignKeys(ctx context.Context, target database.DatabaseTarget) ([]database.ForeignKey, error)
	DefaultDatabase(ctx context.Context) (string, error)
	UsesSchemaQualification() bool
	Close() error
}

// FromConnection returns a DataSource that forwards all calls to the given database.Connection.
func FromConnection(conn database.Connection) DataSource {
	if conn == nil {
		return nil
	}
	return &connectionSource{conn: conn}
}

// connectionSource wraps database.Connection to implement DataSource.
type connectionSource struct {
	conn database.Connection
}

// Databases loads the list of databases from the connection.
func (s *connectionSource) Databases(ctx context.Context) ([]string, error) {
	return s.conn.Databases(ctx)
}

// DatabaseLabel returns the label for the database.
func (s *connectionSource) DatabaseLabel() string {
	return s.conn.DatabaseLabel()
}

// Tables loads the list of tables from the connection.
func (s *connectionSource) Tables(ctx context.Context, database string) ([]string, error) {
	return s.conn.Tables(ctx, database)
}

// Rows loads a page of rows from the connection.
func (s *connectionSource) Rows(ctx context.Context, target database.DatabaseTarget, page database.PageRequest) (database.PageResult, error) {
	return s.conn.Rows(ctx, target, page)
}

// Query executes a query and returns the result.
func (s *connectionSource) Query(ctx context.Context, query string) (database.QueryResult, error) {
	return s.conn.Query(ctx, query)
}

// Indexes loads the list of indexes from the connection.
func (s *connectionSource) Indexes(ctx context.Context, target database.DatabaseTarget) ([]database.Index, error) {
	return s.conn.Indexes(ctx, target)
}

// Constraints loads the list of constraints from the connection.
func (s *connectionSource) Constraints(ctx context.Context, target database.DatabaseTarget) ([]database.Constraint, error) {
	return s.conn.Constraints(ctx, target)
}

// ForeignKeys loads the list of foreign keys from the connection.
func (s *connectionSource) ForeignKeys(ctx context.Context, target database.DatabaseTarget) ([]database.ForeignKey, error) {
	return s.conn.ForeignKeys(ctx, target)
}

// Close closes the connection.
func (s *connectionSource) Close() error {
	return s.conn.Close()
}

// DefaultDatabase returns the default database name.
func (s *connectionSource) DefaultDatabase(ctx context.Context) (string, error) {
	return s.conn.DefaultDatabase(ctx)
}

// UsesSchemaQualification delegates to the underlying connection.
func (s *connectionSource) UsesSchemaQualification() bool {
	return s.conn.UsesSchemaQualification()
}
