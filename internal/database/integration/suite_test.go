//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/jxdones/stoat/internal/database"
)

func testDatabases(t *testing.T, conn database.Connection, expected string) {
	t.Helper()
	ctx := context.Background()

	dbs, err := conn.Databases(ctx)
	if err != nil {
		t.Fatalf("Databases() error: %v", err)
	}

	for _, db := range dbs {
		if db == expected {
			return
		}
	}
	t.Fatalf("expected database %q in list, got: %v", expected, dbs)
}

func testTables(t *testing.T, conn database.Connection, database string, expected []string) {
	t.Helper()
	ctx := context.Background()

	tables, err := conn.Tables(ctx, database)
	if err != nil {
		t.Fatalf("Tables() error: %v", err)
	}

	for _, e := range expected {
		found := false
		for _, table := range tables {
			if table == e {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected table %q in list, got: %v", e, tables)
		}
	}
}

func testRows(t *testing.T, conn database.Connection, target database.DatabaseTarget, expectedCount int) {
	t.Helper()
	ctx := context.Background()

	result, err := conn.Rows(ctx, target, database.PageRequest{Limit: 10})
	if err != nil {
		t.Fatalf("Rows() error: %v", err)
	}

	if len(result.Result.Rows) != expectedCount {
		t.Fatalf("expected %d rows, got %d", expectedCount, len(result.Result.Rows))
	}
}

func testForeignKeys(t *testing.T, conn database.Connection, target database.DatabaseTarget, expectedCount int) {
	t.Helper()
	ctx := context.Background()

	fks, err := conn.ForeignKeys(ctx, target)
	if err != nil {
		t.Fatalf("ForeignKeys() error: %v", err)
	}

	if len(fks) != expectedCount {
		t.Fatalf("expected %d foreign keys, got %d: %v", expectedCount, len(fks), fks)
	}
}

func testIndexes(t *testing.T, conn database.Connection, target database.DatabaseTarget, expectedCount int) {
	t.Helper()
	ctx := context.Background()

	indexes, err := conn.Indexes(ctx, target)
	if err != nil {
		t.Fatalf("Indexes() error: %v", err)
	}

	if len(indexes) != expectedCount {
		t.Fatalf("expected %d indexes, got %d: %v", expectedCount, len(indexes), indexes)
	}
}

func testConstraints(t *testing.T, conn database.Connection, target database.DatabaseTarget, expectedCount int) {
	t.Helper()
	ctx := context.Background()

	constraints, err := conn.Constraints(ctx, target)
	if err != nil {
		t.Fatalf("Constraints() error: %v", err)
	}

	if len(constraints) != expectedCount {
		t.Fatalf("expected %d constraints, got %d: %v", expectedCount, len(constraints), constraints)
	}
}
