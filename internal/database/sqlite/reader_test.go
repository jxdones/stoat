package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/jxdones/stoat/internal/database"
	_ "github.com/mattn/go-sqlite3"
)

func TestReaderDatabases(t *testing.T) {
	tests := []struct {
		name              string
		dbPath            string
		expectedDatabases []string
	}{
		{
			name:              "test_databases_name",
			dbPath:            "./test.db",
			expectedDatabases: []string{"test.db"},
		},
		{
			name:              "test_databases_path",
			dbPath:            "./testdata/anothertest.db",
			expectedDatabases: []string{"anothertest.db"},
		},
		{
			name:              "test_databases_empty",
			dbPath:            "",
			expectedDatabases: []string{"."},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			databases, err := Databases(context.Background(), "", test.dbPath)
			if err != nil {
				t.Fatalf("failed to get databases: %v", err)
			}
			if len(databases) != len(test.expectedDatabases) {
				t.Fatalf("expected %d databases, got %d", len(test.expectedDatabases), len(databases))
			}
			for i, database := range databases {
				if database != test.expectedDatabases[i] {
					t.Fatalf("expected database %s, got %s", test.expectedDatabases[i], database)
				}
			}
		})
	}
}

func TestTables(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		setupDB    func(t *testing.T) *sql.DB
		wantTables []string
		wantErr    error
	}{
		{
			name: "nil_db_returns_err_no_connection",
			setupDB: func(t *testing.T) *sql.DB {
				return nil
			},
			wantTables: nil,
			wantErr:    database.ErrNoConnection,
		},
		{
			name: "empty_db_returns_no_tables",
			setupDB: func(t *testing.T) *sql.DB {
				db, err := sql.Open("sqlite3", ":memory:")
				if err != nil {
					t.Fatalf("open in-memory db: %v", err)
				}
				t.Cleanup(func() { _ = db.Close() })
				return db
			},
			wantTables: []string{},
			wantErr:    nil,
		},
		{
			name: "single_table",
			setupDB: func(t *testing.T) *sql.DB {
				db, err := sql.Open("sqlite3", ":memory:")
				if err != nil {
					t.Fatalf("open in-memory db: %v", err)
				}
				t.Cleanup(func() { _ = db.Close() })
				_, err = db.ExecContext(ctx, "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT);")
				if err != nil {
					t.Fatalf("create table: %v", err)
				}
				return db
			},
			wantTables: []string{"users"},
			wantErr:    nil,
		},
		{
			name: "multiple_tables_sorted_by_name",
			setupDB: func(t *testing.T) *sql.DB {
				db, err := sql.Open("sqlite3", ":memory:")
				if err != nil {
					t.Fatalf("open in-memory db: %v", err)
				}
				t.Cleanup(func() { _ = db.Close() })
				for _, stmt := range []string{
					"CREATE TABLE zebra (id INTEGER);",
					"CREATE TABLE alpha (id INTEGER);",
					"CREATE TABLE beta (id INTEGER);",
				} {
					if _, err := db.ExecContext(ctx, stmt); err != nil {
						t.Fatalf("create table: %v", err)
					}
				}
				return db
			},
			wantTables: []string{"alpha", "beta", "zebra"},
			wantErr:    nil,
		},
		{
			name: "excludes_sqlite_internal_tables",
			setupDB: func(t *testing.T) *sql.DB {
				db, err := sql.Open("sqlite3", ":memory:")
				if err != nil {
					t.Fatalf("open in-memory db: %v", err)
				}
				t.Cleanup(func() { _ = db.Close() })
				// Only user table; sqlite_master/sqlite_sequence etc. are excluded by query
				_, err = db.ExecContext(ctx, "CREATE TABLE my_table (id INTEGER PRIMARY KEY AUTOINCREMENT);")
				if err != nil {
					t.Fatalf("create table: %v", err)
				}
				return db
			},
			wantTables: []string{"my_table"},
			wantErr:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := tt.setupDB(t)
			got, err := Tables(ctx, db)
			if err != nil {
				if tt.wantErr == nil {
					t.Fatalf("Tables() unexpected error: %v", err)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Tables() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			if tt.wantErr != nil {
				t.Fatalf("Tables() err = nil, wantErr %v", tt.wantErr)
			}
			if len(got) != len(tt.wantTables) {
				t.Fatalf("Tables() len = %d, want %d; got %v", len(got), len(tt.wantTables), got)
			}
			for i := range got {
				if got[i] != tt.wantTables[i] {
					t.Fatalf("Tables()[%d] = %q, want %q", i, got[i], tt.wantTables[i])
				}
			}
		})
	}
}

func TestRows(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name            string
		setupDB         func(t *testing.T) *sql.DB
		target          database.DatabaseTarget
		page            database.PageRequest
		wantErr         bool
		wantErrContains string
		wantColumnCount int
		wantRowCount    int
		wantHasMore     bool
		checkFirstRow   map[string]string // optional: assert first row has these key-value pairs
	}{
		{
			name: "nonexistent_table_returns_error",
			setupDB: func(t *testing.T) *sql.DB {
				db, err := sql.Open("sqlite3", ":memory:")
				if err != nil {
					t.Fatalf("open in-memory db: %v", err)
				}
				t.Cleanup(func() { _ = db.Close() })
				return db
			},
			target:          database.DatabaseTarget{Table: "nonexistent"},
			page:            database.PageRequest{},
			wantErr:         true,
			wantErrContains: "table has no columns",
		},
		{
			name: "empty_table_returns_columns_no_rows",
			setupDB: func(t *testing.T) *sql.DB {
				db, err := sql.Open("sqlite3", ":memory:")
				if err != nil {
					t.Fatalf("open in-memory db: %v", err)
				}
				t.Cleanup(func() { _ = db.Close() })
				_, err = db.ExecContext(ctx, "CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT);")
				if err != nil {
					t.Fatalf("create table: %v", err)
				}
				return db
			},
			target:          database.DatabaseTarget{Table: "items"},
			page:            database.PageRequest{},
			wantErr:         false,
			wantColumnCount: 2,
			wantRowCount:    0,
			wantHasMore:     false,
		},
		{
			name: "single_row_returns_one_row",
			setupDB: func(t *testing.T) *sql.DB {
				db, err := sql.Open("sqlite3", ":memory:")
				if err != nil {
					t.Fatalf("open in-memory db: %v", err)
				}
				t.Cleanup(func() { _ = db.Close() })
				_, err = db.ExecContext(ctx, "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT);")
				if err != nil {
					t.Fatalf("create table: %v", err)
				}
				_, err = db.ExecContext(ctx, "INSERT INTO users (id, name) VALUES (1, 'alice');")
				if err != nil {
					t.Fatalf("insert: %v", err)
				}
				return db
			},
			target:          database.DatabaseTarget{Table: "users"},
			page:            database.PageRequest{},
			wantErr:         false,
			wantColumnCount: 2,
			wantRowCount:    1,
			wantHasMore:     false,
			checkFirstRow:   map[string]string{"id": "1", "name": "alice"},
		},
		{
			name: "multiple_rows_respects_limit_and_has_more",
			setupDB: func(t *testing.T) *sql.DB {
				db, err := sql.Open("sqlite3", ":memory:")
				if err != nil {
					t.Fatalf("open in-memory db: %v", err)
				}
				t.Cleanup(func() { _ = db.Close() })
				_, err = db.ExecContext(ctx, "CREATE TABLE nums (id INTEGER PRIMARY KEY, val INTEGER);")
				if err != nil {
					t.Fatalf("create table: %v", err)
				}
				for i := 1; i <= 5; i++ {
					_, err = db.ExecContext(ctx, "INSERT INTO nums (id, val) VALUES (?, ?);", i, i*10)
					if err != nil {
						t.Fatalf("insert: %v", err)
					}
				}
				return db
			},
			target:          database.DatabaseTarget{Table: "nums"},
			page:            database.PageRequest{Limit: 2},
			wantErr:         false,
			wantColumnCount: 2,
			wantRowCount:    2,
			wantHasMore:     true,
		},
		{
			name: "second_page_using_cursor",
			setupDB: func(t *testing.T) *sql.DB {
				db, err := sql.Open("sqlite3", ":memory:")
				if err != nil {
					t.Fatalf("open in-memory db: %v", err)
				}
				t.Cleanup(func() { _ = db.Close() })
				_, err = db.ExecContext(ctx, "CREATE TABLE nums (id INTEGER PRIMARY KEY, val INTEGER);")
				if err != nil {
					t.Fatalf("create table: %v", err)
				}
				for i := 1; i <= 5; i++ {
					_, err = db.ExecContext(ctx, "INSERT INTO nums (id, val) VALUES (?, ?);", i, i*10)
					if err != nil {
						t.Fatalf("insert: %v", err)
					}
				}
				return db
			},
			target:          database.DatabaseTarget{Table: "nums"},
			page:            database.PageRequest{Limit: 2, After: "rowid:2"},
			wantErr:         false,
			wantColumnCount: 2,
			wantRowCount:    2,
			wantHasMore:     true,
			checkFirstRow:   map[string]string{"id": "3", "val": "30"},
		},
		{
			name: "zero_limit_uses_default_page_size",
			setupDB: func(t *testing.T) *sql.DB {
				db, err := sql.Open("sqlite3", ":memory:")
				if err != nil {
					t.Fatalf("open in-memory db: %v", err)
				}
				t.Cleanup(func() { _ = db.Close() })
				_, err = db.ExecContext(ctx, "CREATE TABLE small (id INTEGER PRIMARY KEY);")
				if err != nil {
					t.Fatalf("create table: %v", err)
				}
				_, err = db.ExecContext(ctx, "INSERT INTO small (id) VALUES (1);")
				if err != nil {
					t.Fatalf("insert: %v", err)
				}
				return db
			},
			target:          database.DatabaseTarget{Table: "small"},
			page:            database.PageRequest{Limit: 0},
			wantErr:         false,
			wantColumnCount: 1,
			wantRowCount:    1,
			wantHasMore:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := tt.setupDB(t)
			got, err := Rows(ctx, db, tt.target, tt.page)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Rows() expected error containing %q, got nil", tt.wantErrContains)
				}
				if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Fatalf("Rows() error = %v, want containing %q", err, tt.wantErrContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("Rows() unexpected error: %v", err)
			}
			if len(got.Result.Columns) != tt.wantColumnCount {
				t.Fatalf("Rows() column count = %d, want %d", len(got.Result.Columns), tt.wantColumnCount)
			}
			if len(got.Result.Rows) != tt.wantRowCount {
				t.Fatalf("Rows() row count = %d, want %d", len(got.Result.Rows), tt.wantRowCount)
			}
			if got.HasMore != tt.wantHasMore {
				t.Fatalf("Rows() HasMore = %v, want %v", got.HasMore, tt.wantHasMore)
			}
			for key, wantVal := range tt.checkFirstRow {
				if len(got.Result.Rows) == 0 {
					t.Fatalf("Rows() checkFirstRow specified but no rows returned")
				}
				if v, ok := got.Result.Rows[0][key]; !ok || v != wantVal {
					t.Fatalf("Rows() first row[%q] = %q, want %q", key, got.Result.Rows[0][key], wantVal)
				}
			}
		})
	}
}

func TestQuery(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name             string
		setupDB          func(t *testing.T) *sql.DB
		query            string
		wantErr          bool
		wantErrContains  string
		wantColumnCount  int
		wantRowCount     int
		wantRowsAffected *int64 // nil = don't check (SELECT leaves connection's changes() from prior stmt)
		checkFirstRow    map[string]string
	}{
		{
			name: "invalid_sql_returns_error",
			setupDB: func(t *testing.T) *sql.DB {
				db, err := sql.Open("sqlite3", ":memory:")
				if err != nil {
					t.Fatalf("open in-memory db: %v", err)
				}
				t.Cleanup(func() { _ = db.Close() })
				return db
			},
			query:           "SELECT FROM broken",
			wantErr:         true,
			wantErrContains: "",
		},
		{
			name: "select_empty_table",
			setupDB: func(t *testing.T) *sql.DB {
				db, err := sql.Open("sqlite3", ":memory:")
				if err != nil {
					t.Fatalf("open in-memory db: %v", err)
				}
				t.Cleanup(func() { _ = db.Close() })
				_, err = db.ExecContext(ctx, "CREATE TABLE empty_t (id INTEGER);")
				if err != nil {
					t.Fatalf("create table: %v", err)
				}
				return db
			},
			query:            "SELECT * FROM empty_t",
			wantErr:          false,
			wantColumnCount:  1,
			wantRowCount:     0,
			wantRowsAffected: nil,
		},
		{
			name: "select_returns_rows",
			setupDB: func(t *testing.T) *sql.DB {
				db, err := sql.Open("sqlite3", ":memory:")
				if err != nil {
					t.Fatalf("open in-memory db: %v", err)
				}
				t.Cleanup(func() { _ = db.Close() })
				_, err = db.ExecContext(ctx, "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT);")
				if err != nil {
					t.Fatalf("create table: %v", err)
				}
				_, err = db.ExecContext(ctx, "INSERT INTO users (id, name) VALUES (1, 'alice');")
				if err != nil {
					t.Fatalf("insert: %v", err)
				}
				return db
			},
			query:            "SELECT * FROM users",
			wantErr:          false,
			wantColumnCount:  2,
			wantRowCount:     1,
			wantRowsAffected: nil,
			checkFirstRow:    map[string]string{"id": "1", "name": "alice"},
		},
		{
			name: "select_literal",
			setupDB: func(t *testing.T) *sql.DB {
				db, err := sql.Open("sqlite3", ":memory:")
				if err != nil {
					t.Fatalf("open in-memory db: %v", err)
				}
				t.Cleanup(func() { _ = db.Close() })
				return db
			},
			query:            "SELECT 1 AS one, 'hello' AS msg",
			wantErr:          false,
			wantColumnCount:  2,
			wantRowCount:     1,
			wantRowsAffected: nil,
			checkFirstRow:    map[string]string{"one": "1", "msg": "hello"},
		},
		{
			name: "insert_sets_rows_affected",
			setupDB: func(t *testing.T) *sql.DB {
				db, err := sql.Open("sqlite3", ":memory:")
				if err != nil {
					t.Fatalf("open in-memory db: %v", err)
				}
				t.Cleanup(func() { _ = db.Close() })
				_, err = db.ExecContext(ctx, "CREATE TABLE ins (id INTEGER);")
				if err != nil {
					t.Fatalf("create table: %v", err)
				}
				return db
			},
			query:            "INSERT INTO ins (id) VALUES (1)",
			wantErr:          false,
			wantColumnCount:  0,
			wantRowCount:     0,
			wantRowsAffected: int64Ptr(1),
		},
		{
			name: "update_sets_rows_affected",
			setupDB: func(t *testing.T) *sql.DB {
				db, err := sql.Open("sqlite3", ":memory:")
				if err != nil {
					t.Fatalf("open in-memory db: %v", err)
				}
				t.Cleanup(func() { _ = db.Close() })
				_, err = db.ExecContext(ctx, "CREATE TABLE upd (id INTEGER, x TEXT);")
				if err != nil {
					t.Fatalf("create table: %v", err)
				}
				_, err = db.ExecContext(ctx, "INSERT INTO upd (id, x) VALUES (1, 'a'), (2, 'b'), (3, 'a');")
				if err != nil {
					t.Fatalf("insert: %v", err)
				}
				return db
			},
			query:            "UPDATE upd SET x = 'updated' WHERE id = 1",
			wantErr:          false,
			wantColumnCount:  0,
			wantRowCount:     0,
			wantRowsAffected: int64Ptr(1),
		},
		{
			name: "group_by_with_aggregate",
			setupDB: func(t *testing.T) *sql.DB {
				db, err := sql.Open("sqlite3", ":memory:")
				if err != nil {
					t.Fatalf("open in-memory db: %v", err)
				}
				t.Cleanup(func() { _ = db.Close() })
				_, err = db.ExecContext(ctx, "CREATE TABLE payment (customer_id INTEGER, amount REAL);")
				if err != nil {
					t.Fatalf("create table: %v", err)
				}
				_, err = db.ExecContext(ctx, "INSERT INTO payment (customer_id, amount) VALUES (1, 10.5), (1, 20), (2, 5);")
				if err != nil {
					t.Fatalf("insert: %v", err)
				}
				return db
			},
			query:           "SELECT customer_id, SUM(amount) AS total_spent FROM payment GROUP BY customer_id ORDER BY total_spent DESC",
			wantErr:         false,
			wantColumnCount: 2,
			wantRowCount:    2,
			checkFirstRow:   map[string]string{"customer_id": "1", "total_spent": "30.5"},
		},
		{
			name: "explain_query_plan_returns_rows",
			setupDB: func(t *testing.T) *sql.DB {
				db, err := sql.Open("sqlite3", ":memory:")
				if err != nil {
					t.Fatalf("open in-memory db: %v", err)
				}
				t.Cleanup(func() { _ = db.Close() })
				_, err = db.ExecContext(ctx, "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT);")
				if err != nil {
					t.Fatalf("create table: %v", err)
				}
				return db
			},
			query:           "EXPLAIN QUERY PLAN SELECT * FROM users",
			wantErr:         false,
			wantColumnCount: 4,
			wantRowCount:    1,
		},
		{
			name: "group_by_with_trailing_semicolon",
			setupDB: func(t *testing.T) *sql.DB {
				db, err := sql.Open("sqlite3", ":memory:")
				if err != nil {
					t.Fatalf("open in-memory db: %v", err)
				}
				t.Cleanup(func() { _ = db.Close() })
				_, err = db.ExecContext(ctx, "CREATE TABLE payment (customer_id INTEGER, amount REAL);")
				if err != nil {
					t.Fatalf("create table: %v", err)
				}
				_, err = db.ExecContext(ctx, "INSERT INTO payment (customer_id, amount) VALUES (1, 10.5), (1, 20);")
				if err != nil {
					t.Fatalf("insert: %v", err)
				}
				return db
			},
			query:           "SELECT customer_id, SUM(amount) AS total_spent FROM payment GROUP BY customer_id ORDER BY total_spent DESC LIMIT 10;",
			wantErr:         false,
			wantColumnCount: 2,
			wantRowCount:    1,
			checkFirstRow:   map[string]string{"customer_id": "1", "total_spent": "30.5"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := tt.setupDB(t)
			got, err := Query(ctx, db, tt.query)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Query() expected error, got nil")
				}
				if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Fatalf("Query() error = %v, want containing %q", err, tt.wantErrContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("Query() unexpected error: %v", err)
			}
			if len(got.Columns) != tt.wantColumnCount {
				t.Fatalf("Query() column count = %d, want %d", len(got.Columns), tt.wantColumnCount)
			}
			if len(got.Rows) != tt.wantRowCount {
				t.Fatalf("Query() row count = %d, want %d", len(got.Rows), tt.wantRowCount)
			}
			if tt.wantRowsAffected != nil && got.RowsAffected != *tt.wantRowsAffected {
				t.Fatalf("Query() RowsAffected = %d, want %d", got.RowsAffected, *tt.wantRowsAffected)
			}
			for key, wantVal := range tt.checkFirstRow {
				if len(got.Rows) == 0 {
					t.Fatalf("Query() checkFirstRow specified but no rows returned")
				}
				if v, ok := got.Rows[0][key]; !ok || v != wantVal {
					t.Fatalf("Query() first row[%q] = %q, want %q", key, got.Rows[0][key], wantVal)
				}
			}
		})
	}
}

func TestIndexes(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		setupDB    func(t *testing.T) *sql.DB
		target     database.DatabaseTarget
		wantErr    bool
		wantCount  int
		checkFirst *database.Index
	}{
		{
			name: "nonexistent_table_returns_empty",
			setupDB: func(t *testing.T) *sql.DB {
				db, err := sql.Open("sqlite3", ":memory:")
				if err != nil {
					t.Fatalf("open in-memory db: %v", err)
				}
				t.Cleanup(func() { _ = db.Close() })
				return db
			},
			target:    database.DatabaseTarget{Table: "nonexistent"},
			wantErr:   false,
			wantCount: 0,
		},
		{
			name: "table_with_no_explicit_indexes",
			setupDB: func(t *testing.T) *sql.DB {
				db, err := sql.Open("sqlite3", ":memory:")
				if err != nil {
					t.Fatalf("open in-memory db: %v", err)
				}
				t.Cleanup(func() { _ = db.Close() })
				_, err = db.ExecContext(ctx, "CREATE TABLE t (id INTEGER PRIMARY KEY);")
				if err != nil {
					t.Fatalf("create table: %v", err)
				}
				return db
			},
			target:    database.DatabaseTarget{Table: "t"},
			wantErr:   false,
			wantCount: 0, // INTEGER PRIMARY KEY uses rowid, no separate index in index_list
		},
		{
			name: "table_with_explicit_index",
			setupDB: func(t *testing.T) *sql.DB {
				db, err := sql.Open("sqlite3", ":memory:")
				if err != nil {
					t.Fatalf("open in-memory db: %v", err)
				}
				t.Cleanup(func() { _ = db.Close() })
				_, err = db.ExecContext(ctx, "CREATE TABLE items (id INTEGER, name TEXT);")
				if err != nil {
					t.Fatalf("create table: %v", err)
				}
				_, err = db.ExecContext(ctx, "CREATE INDEX idx_name ON items(name);")
				if err != nil {
					t.Fatalf("create index: %v", err)
				}
				return db
			},
			target:     database.DatabaseTarget{Table: "items"},
			wantErr:    false,
			wantCount:  1,
			checkFirst: &database.Index{Name: "idx_name", Columns: []string{"name"}, Unique: false},
		},
		{
			name: "table_with_unique_index",
			setupDB: func(t *testing.T) *sql.DB {
				db, err := sql.Open("sqlite3", ":memory:")
				if err != nil {
					t.Fatalf("open in-memory db: %v", err)
				}
				t.Cleanup(func() { _ = db.Close() })
				_, err = db.ExecContext(ctx, "CREATE TABLE u (id INTEGER, code TEXT);")
				if err != nil {
					t.Fatalf("create table: %v", err)
				}
				_, err = db.ExecContext(ctx, "CREATE UNIQUE INDEX uq_code ON u(code);")
				if err != nil {
					t.Fatalf("create unique index: %v", err)
				}
				return db
			},
			target:     database.DatabaseTarget{Table: "u"},
			wantErr:    false,
			wantCount:  1,
			checkFirst: &database.Index{Name: "uq_code", Columns: []string{"code"}, Unique: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := tt.setupDB(t)
			got, err := Indexes(ctx, db, tt.target)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Indexes() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Indexes() unexpected error: %v", err)
			}
			if len(got) != tt.wantCount {
				t.Fatalf("Indexes() count = %d, want %d", len(got), tt.wantCount)
			}
			if tt.checkFirst != nil && len(got) > 0 {
				first := got[0]
				if first.Name != tt.checkFirst.Name {
					t.Fatalf("Indexes()[0].Name = %q, want %q", first.Name, tt.checkFirst.Name)
				}
				if first.Unique != tt.checkFirst.Unique {
					t.Fatalf("Indexes()[0].Unique = %v, want %v", first.Unique, tt.checkFirst.Unique)
				}
				if len(first.Columns) != len(tt.checkFirst.Columns) {
					t.Fatalf("Indexes()[0].Columns len = %d, want %d", len(first.Columns), len(tt.checkFirst.Columns))
				}
				for i, c := range tt.checkFirst.Columns {
					if i >= len(first.Columns) || first.Columns[i] != c {
						t.Fatalf("Indexes()[0].Columns[%d] = %v, want %q", i, first.Columns, c)
					}
				}
			}
		})
	}
}

func TestConstraints(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		setupDB   func(t *testing.T) *sql.DB
		target    database.DatabaseTarget
		wantErr   bool
		wantCount int
		wantTypes []string
	}{
		{
			name: "nonexistent_table_returns_empty",
			setupDB: func(t *testing.T) *sql.DB {
				db, err := sql.Open("sqlite3", ":memory:")
				if err != nil {
					t.Fatalf("open in-memory db: %v", err)
				}
				t.Cleanup(func() { _ = db.Close() })
				return db
			},
			target:    database.DatabaseTarget{Table: "nonexistent"},
			wantErr:   false,
			wantCount: 0,
		},
		{
			name: "primary_key_constraint",
			setupDB: func(t *testing.T) *sql.DB {
				db, err := sql.Open("sqlite3", ":memory:")
				if err != nil {
					t.Fatalf("open in-memory db: %v", err)
				}
				t.Cleanup(func() { _ = db.Close() })
				_, err = db.ExecContext(ctx, "CREATE TABLE pk_t (id INTEGER PRIMARY KEY);")
				if err != nil {
					t.Fatalf("create table: %v", err)
				}
				return db
			},
			target:    database.DatabaseTarget{Table: "pk_t"},
			wantErr:   false,
			wantCount: 1,
			wantTypes: []string{"PRIMARY KEY"},
		},
		{
			name: "not_null_and_default",
			setupDB: func(t *testing.T) *sql.DB {
				db, err := sql.Open("sqlite3", ":memory:")
				if err != nil {
					t.Fatalf("open in-memory db: %v", err)
				}
				t.Cleanup(func() { _ = db.Close() })
				_, err = db.ExecContext(ctx, "CREATE TABLE nn (id INTEGER PRIMARY KEY, x TEXT NOT NULL, y TEXT DEFAULT 'hi');")
				if err != nil {
					t.Fatalf("create table: %v", err)
				}
				return db
			},
			target:    database.DatabaseTarget{Table: "nn"},
			wantErr:   false,
			wantTypes: []string{"PRIMARY KEY", "NOT NULL", "DEFAULT"},
		},
		{
			name: "unique_constraint_via_index",
			setupDB: func(t *testing.T) *sql.DB {
				db, err := sql.Open("sqlite3", ":memory:")
				if err != nil {
					t.Fatalf("open in-memory db: %v", err)
				}
				t.Cleanup(func() { _ = db.Close() })
				_, err = db.ExecContext(ctx, "CREATE TABLE uq_t (id INTEGER PRIMARY KEY, code TEXT);")
				if err != nil {
					t.Fatalf("create table: %v", err)
				}
				_, err = db.ExecContext(ctx, "CREATE UNIQUE INDEX uq_code ON uq_t(code);")
				if err != nil {
					t.Fatalf("create unique index: %v", err)
				}
				return db
			},
			target:    database.DatabaseTarget{Table: "uq_t"},
			wantErr:   false,
			wantTypes: []string{"PRIMARY KEY", "UNIQUE"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := tt.setupDB(t)
			got, err := Constraints(ctx, db, tt.target)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Constraints() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Constraints() unexpected error: %v", err)
			}
			if tt.wantCount > 0 && len(got) != tt.wantCount {
				t.Fatalf("Constraints() count = %d, want %d", len(got), tt.wantCount)
			}
			if len(tt.wantTypes) > 0 {
				gotTypes := make(map[string]int)
				for _, c := range got {
					gotTypes[c.Type] = gotTypes[c.Type] + 1
				}
				for _, wantType := range tt.wantTypes {
					if gotTypes[wantType] == 0 {
						t.Fatalf("Constraints() missing type %q; got %v", wantType, got)
					}
				}
			}
		})
	}
}

func TestForeignKeys(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		setupDB    func(t *testing.T) *sql.DB
		target     database.DatabaseTarget
		wantErr    bool
		wantCount  int
		checkFirst *database.ForeignKey
	}{
		{
			name: "table_with_no_foreign_keys",
			setupDB: func(t *testing.T) *sql.DB {
				db, err := sql.Open("sqlite3", ":memory:")
				if err != nil {
					t.Fatalf("open in-memory db: %v", err)
				}
				t.Cleanup(func() { _ = db.Close() })
				_, err = db.ExecContext(ctx, "CREATE TABLE t (id INTEGER PRIMARY KEY);")
				if err != nil {
					t.Fatalf("create table: %v", err)
				}
				return db
			},
			target:    database.DatabaseTarget{Table: "t"},
			wantErr:   false,
			wantCount: 0,
		},
		{
			name: "table_with_one_foreign_key",
			setupDB: func(t *testing.T) *sql.DB {
				db, err := sql.Open("sqlite3", ":memory:")
				if err != nil {
					t.Fatalf("open in-memory db: %v", err)
				}
				t.Cleanup(func() { _ = db.Close() })
				_, err = db.ExecContext(ctx, "CREATE TABLE parent (id INTEGER PRIMARY KEY);")
				if err != nil {
					t.Fatalf("create parent: %v", err)
				}
				_, err = db.ExecContext(ctx, "CREATE TABLE child (id INTEGER PRIMARY KEY, parent_id INTEGER REFERENCES parent(id));")
				if err != nil {
					t.Fatalf("create child: %v", err)
				}
				return db
			},
			target:    database.DatabaseTarget{Table: "child"},
			wantErr:   false,
			wantCount: 1,
			checkFirst: &database.ForeignKey{
				Name:      "fk_child_parent_id",
				Column:    "parent_id",
				RefTable:  "parent",
				RefColumn: "id",
			},
		},
		{
			name: "table_with_multiple_foreign_keys",
			setupDB: func(t *testing.T) *sql.DB {
				db, err := sql.Open("sqlite3", ":memory:")
				if err != nil {
					t.Fatalf("open in-memory db: %v", err)
				}
				t.Cleanup(func() { _ = db.Close() })
				_, err = db.ExecContext(ctx, "CREATE TABLE a (id INTEGER PRIMARY KEY);")
				if err != nil {
					t.Fatalf("create a: %v", err)
				}
				_, err = db.ExecContext(ctx, "CREATE TABLE b (id INTEGER PRIMARY KEY);")
				if err != nil {
					t.Fatalf("create b: %v", err)
				}
				_, err = db.ExecContext(ctx, "CREATE TABLE link (id INTEGER PRIMARY KEY, a_id INTEGER REFERENCES a(id), b_id INTEGER REFERENCES b(id));")
				if err != nil {
					t.Fatalf("create link: %v", err)
				}
				return db
			},
			target:    database.DatabaseTarget{Table: "link"},
			wantErr:   false,
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := tt.setupDB(t)
			got, err := ForeignKeys(ctx, db, tt.target)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ForeignKeys() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ForeignKeys() unexpected error: %v", err)
			}
			if len(got) != tt.wantCount {
				t.Fatalf("ForeignKeys() count = %d, want %d", len(got), tt.wantCount)
			}
			if tt.checkFirst != nil && len(got) > 0 {
				first := got[0]
				if first.Name != tt.checkFirst.Name {
					t.Fatalf("ForeignKeys()[0].Name = %q, want %q", first.Name, tt.checkFirst.Name)
				}
				if first.Column != tt.checkFirst.Column {
					t.Fatalf("ForeignKeys()[0].Column = %q, want %q", first.Column, tt.checkFirst.Column)
				}
				if first.RefTable != tt.checkFirst.RefTable {
					t.Fatalf("ForeignKeys()[0].RefTable = %q, want %q", first.RefTable, tt.checkFirst.RefTable)
				}
				if first.RefColumn != tt.checkFirst.RefColumn {
					t.Fatalf("ForeignKeys()[0].RefColumn = %q, want %q", first.RefColumn, tt.checkFirst.RefColumn)
				}
			}
		})
	}
}

// Helper functions for tests in this file.
func int64Ptr(n int64) *int64 {
	return &n
}
