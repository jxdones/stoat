package model

import (
	"reflect"
	"testing"

	"github.com/jxdones/stoat/internal/database"
	"github.com/jxdones/stoat/internal/ui/components/table"
)

func TestDBColumnsToTable(t *testing.T) {
	tests := []struct {
		name string
		in   []database.Column
		want []table.Column
	}{
		{
			name: "empty_slice",
			in:   []database.Column{},
			want: []table.Column{},
		},
		{
			name: "single_column_maps_all_fields",
			in: []database.Column{
				{
					Key:      "id",
					Title:    "ID",
					Type:     "INTEGER",
					MinWidth: 6,
					Order:    1,
				},
			},
			want: []table.Column{
				{
					Key:      "id",
					Title:    "ID",
					Type:     "INTEGER",
					MinWidth: 6,
					Order:    1,
				},
			},
		},
		{
			name: "multiple_columns_preserve_order",
			in: []database.Column{
				{Key: "name", Title: "Name", Type: "TEXT", MinWidth: 12, Order: 2},
				{Key: "email", Title: "Email", Type: "TEXT", MinWidth: 16, Order: 3},
			},
			want: []table.Column{
				{Key: "name", Title: "Name", Type: "TEXT", MinWidth: 12, Order: 2},
				{Key: "email", Title: "Email", Type: "TEXT", MinWidth: 16, Order: 3},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dbColumnsToTable(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("dbColumnsToTable() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestDBRowsToTable(t *testing.T) {
	tests := []struct {
		name string
		in   []database.Row
		want []table.Row
	}{
		{
			name: "empty_slice",
			in:   []database.Row{},
			want: []table.Row{},
		},
		{
			name: "single_row_maps_fields",
			in: []database.Row{
				{"id": "1", "name": "alice"},
			},
			want: []table.Row{
				{"id": "1", "name": "alice"},
			},
		},
		{
			name: "multiple_rows_preserve_order",
			in: []database.Row{
				{"id": "1", "name": "alice"},
				{"id": "2", "name": "bob"},
			},
			want: []table.Row{
				{"id": "1", "name": "alice"},
				{"id": "2", "name": "bob"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dbRowsToTable(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("dbRowsToTable() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestSchemaIndexesToTable(t *testing.T) {
	tests := []struct {
		name     string
		in       []database.Index
		wantCols []table.Column
		wantRows []table.Row
	}{
		{
			name: "empty_slice",
			in:   []database.Index{},
			wantCols: []table.Column{
				{Key: "name", Title: "Name", MinWidth: 20, Order: 1},
				{Key: "unique", Title: "Unique", MinWidth: 10, Order: 2},
				{Key: "columns", Title: "Columns", MinWidth: 20, Order: 3},
			},
			wantRows: []table.Row{},
		},
		{
			name: "index_fields_mapped_to_rows",
			in: []database.Index{
				{Name: "idx_users_email", Unique: true, Columns: []string{"email"}},
				{Name: "idx_users_name_role", Unique: false, Columns: []string{"name", "role"}},
			},
			wantCols: []table.Column{
				{Key: "name", Title: "Name", MinWidth: 20, Order: 1},
				{Key: "unique", Title: "Unique", MinWidth: 10, Order: 2},
				{Key: "columns", Title: "Columns", MinWidth: 20, Order: 3},
			},
			wantRows: []table.Row{
				{"name": "idx_users_email", "unique": "YES", "columns": "email"},
				{"name": "idx_users_name_role", "unique": "NO", "columns": "name, role"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCols, gotRows := schemaIndexesToTable(tt.in)
			if !reflect.DeepEqual(gotCols, tt.wantCols) {
				t.Fatalf("schemaIndexesToTable() cols = %#v, want %#v", gotCols, tt.wantCols)
			}
			if !reflect.DeepEqual(gotRows, tt.wantRows) {
				t.Fatalf("schemaIndexesToTable() rows = %#v, want %#v", gotRows, tt.wantRows)
			}
		})
	}
}

func TestSchemaConstraintsToTable(t *testing.T) {
	tests := []struct {
		name     string
		in       []database.Constraint
		wantCols []table.Column
		wantRows []table.Row
	}{
		{
			name: "empty_slice",
			in:   []database.Constraint{},
			wantCols: []table.Column{
				{Key: "name", Title: "Name", MinWidth: 20, Order: 1},
				{Key: "type", Title: "Type", MinWidth: 10, Order: 2},
				{Key: "columns", Title: "Columns", MinWidth: 20, Order: 3},
			},
			wantRows: []table.Row{},
		},
		{
			name: "constraint_fields_mapped_to_rows",
			in: []database.Constraint{
				{Name: "pk_users", Type: "PRIMARY KEY", Columns: []string{"id"}},
				{Name: "uq_users_email_org", Type: "UNIQUE", Columns: []string{"email", "org_id"}},
			},
			wantCols: []table.Column{
				{Key: "name", Title: "Name", MinWidth: 20, Order: 1},
				{Key: "type", Title: "Type", MinWidth: 10, Order: 2},
				{Key: "columns", Title: "Columns", MinWidth: 20, Order: 3},
			},
			wantRows: []table.Row{
				{"name": "pk_users", "type": "PRIMARY KEY", "columns": "id"},
				{"name": "uq_users_email_org", "type": "UNIQUE", "columns": "email, org_id"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCols, gotRows := schemaConstraintsToTable(tt.in)
			if !reflect.DeepEqual(gotCols, tt.wantCols) {
				t.Fatalf("schemaConstraintsToTable() cols = %#v, want %#v", gotCols, tt.wantCols)
			}
			if !reflect.DeepEqual(gotRows, tt.wantRows) {
				t.Fatalf("schemaConstraintsToTable() rows = %#v, want %#v", gotRows, tt.wantRows)
			}
		})
	}
}

func TestSchemaColumnsToTable(t *testing.T) {
	tests := []struct {
		name     string
		in       []database.Column
		wantCols []table.Column
		wantRows []table.Row
	}{
		{
			name: "empty_slice",
			in:   []database.Column{},
			wantCols: []table.Column{
				{Key: "name", Title: "Name", MinWidth: 20, Order: 1},
				{Key: "type", Title: "Type", MinWidth: 10, Order: 2},
			},
			wantRows: []table.Row{},
		},
		{
			name: "column_key_and_type_mapped_to_rows",
			in: []database.Column{
				{Key: "id", Type: "INTEGER"},
				{Key: "email", Type: "TEXT"},
			},
			wantCols: []table.Column{
				{Key: "name", Title: "Name", MinWidth: 20, Order: 1},
				{Key: "type", Title: "Type", MinWidth: 10, Order: 2},
			},
			wantRows: []table.Row{
				{"name": "id", "type": "INTEGER"},
				{"name": "email", "type": "TEXT"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCols, gotRows := schemaColumnsToTable(tt.in)
			if !reflect.DeepEqual(gotCols, tt.wantCols) {
				t.Fatalf("schemaColumnsToTable() cols = %#v, want %#v", gotCols, tt.wantCols)
			}
			if !reflect.DeepEqual(gotRows, tt.wantRows) {
				t.Fatalf("schemaColumnsToTable() rows = %#v, want %#v", gotRows, tt.wantRows)
			}
		})
	}
}
