package database

import "context"

type Connection interface {
	Databases(ctx context.Context) ([]string, error)
	DatabaseLabel() string
	Tables(ctx context.Context, database string) ([]string, error)
	Rows(ctx context.Context, target DatabaseTarget, page PageRequest) (PageResult, error)
	Query(ctx context.Context, query string) (QueryResult, error)
	Indexes(ctx context.Context, target DatabaseTarget) ([]Index, error)
	Constraints(ctx context.Context, target DatabaseTarget) ([]Constraint, error)
	ForeignKeys(ctx context.Context, target DatabaseTarget) ([]ForeignKey, error)
	DefaultDatabase(ctx context.Context) (string, error)
	UsesSchemaQualification() bool
	Close() error
}
