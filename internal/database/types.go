package database

// DBMS is the type of database management system.
type DBMS string

const (
	DBMSSQLite   DBMS = "sqlite"
	DBMSPostgres DBMS = "postgres"
)

// Config is the configuration for a database connection.
type Config struct {
	Name     string
	DBMS     DBMS
	Values   map[string]string
	ReadOnly bool
}

// Column is a column in a table.
type Column struct {
	Key      string
	Title    string
	Type     string
	MinWidth int
	Order    int
}

// Row stores one record keyed by column key.
type Row map[string]string

// Index is an index on a table.
type Index struct {
	Name    string
	Columns []string
	Unique  bool
}

// Constraint is a constraint on a table.
type Constraint struct {
	Name    string
	Type    string
	Columns []string
	Detail  string
}

// ForeignKey is a foreign key on a table.
type ForeignKey struct {
	Name           string
	Column         string
	RefTable       string
	RefColumn      string
	OnUpdateAction string
	OnDeleteAction string
}

// DatabaseTarget is a target database and table.
type DatabaseTarget struct {
	Database string
	Table    string
}

// PageRequest describes keyset-style pagination input.
type PageRequest struct {
	Limit int
	After string // cursor value in format "rowid:123" or "pk:123" or "offset:123"
}

// QueryResult provides a normalized response for query execution.
type QueryResult struct {
	Columns      []Column
	Rows         []Row
	RowsAffected int64
}

// PageResult describes a keyset page plus cursor metadata.
type PageResult struct {
	Result     QueryResult
	StartAfter int64
	HasMore    bool
	NextAfter  string
}
