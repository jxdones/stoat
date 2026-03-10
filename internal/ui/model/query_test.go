package model

import (
	"strings"
	"testing"
)

func TestFormatSQLValue(t *testing.T) {
	tests := []struct {
		name    string
		colType string
		value   string
		want    string
	}{
		{
			name:    "empty_value_returns_NULL",
			colType: "text",
			value:   "",
			want:    "NULL",
		},
		{
			name:    "whitespace_only_returns_NULL",
			colType: "text",
			value:   "   \t  ",
			want:    "NULL",
		},
		{
			name:    "integer_type_valid_int_unquoted",
			colType: "integer",
			value:   "42",
			want:    "42",
		},
		{
			name:    "integer_type_negative_unquoted",
			colType: "INT",
			value:   "-1",
			want:    "-1",
		},
		{
			name:    "integer_type_non_numeric_quoted",
			colType: "integer",
			value:   "abc",
			want:    "'abc'",
		},
		{
			name:    "numeric_type_valid_int_unquoted",
			colType: "NUMERIC",
			value:   "0",
			want:    "0",
		},
		{
			name:    "real_type_valid_float_unquoted",
			colType: "real",
			value:   "3.14",
			want:    "3.14",
		},
		{
			name:    "float_type_valid_unquoted",
			colType: "FLOAT",
			value:   "1e-2",
			want:    "1e-2", // original value returned when parse succeeds
		},
		{
			name:    "real_type_non_numeric_quoted",
			colType: "REAL",
			value:   "nope",
			want:    "'nope'",
		},
		{
			name:    "text_type_quoted",
			colType: "text",
			value:   "hello",
			want:    "'hello'",
		},
		{
			name:    "text_type_single_quote_escaped",
			colType: "TEXT",
			value:   "O'Brien",
			want:    "'O''Brien'",
		},
		{
			name:    "unknown_type_quoted",
			colType: "blob",
			value:   "x",
			want:    "'x'",
		},
		{
			name:    "value_trimmed",
			colType: "text",
			value:   "  hello  ",
			want:    "'hello'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSQLValue(tt.colType, tt.value)
			if got != tt.want {
				t.Errorf("formatSQLValue(%q, %q) = %q, want %q", tt.colType, tt.value, got, tt.want)
			}
		})
	}
}

func TestQuoteIdentifier(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "simple",
			in:   "users",
			want: `"users"`,
		},
		{
			name: "reserved",
			in:   "order",
			want: `"order"`,
		},
		{
			name: "with_space",
			in:   "my column",
			want: `"my column"`,
		},
		{
			name: "double_quote_escaped",
			in:   `foo"bar`,
			want: `"foo""bar"`,
		},
		{
			name: "empty",
			in:   "",
			want: `""`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := quoteIdentifier(tt.in)
			if got != tt.want {
				t.Errorf("quoteIdentifier(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestBuildUpdateQueryFromCell(t *testing.T) {
	tests := []struct {
		name         string
		table        string
		setColumn    string
		setColType   string
		setValue     string
		pkColumns    []string
		row          map[string]string
		colTypeByKey map[string]string
		want         []string
		notWant      []string
	}{
		{
			name:       "basic_update_fallback_when_no_pk",
			table:      "users",
			setColumn:  "name",
			setColType: "text",
			setValue:   "alice",
			want: []string{
				"WARNING",
				"WHERE uses the edited column",
				`UPDATE "users"`,
				`SET "name" = 'alice'`,
				`WHERE "name" = 'alice';`,
			},
		},
		{
			name:       "integer_column_unquoted_literal",
			table:      "items",
			setColumn:  "id",
			setColType: "integer",
			setValue:   "1",
			want: []string{
				"WARNING",
				`UPDATE "items"`,
				`SET "id" = 1`,
				`WHERE "id" = 1;`,
			},
		},
		{
			name:       "empty_value_produces_NULL",
			table:      "t",
			setColumn:  "opt",
			setColType: "text",
			setValue:   "",
			want: []string{
				"WARNING",
				`SET "opt" = NULL`,
				`WHERE "opt" = NULL;`,
			},
		},
		{
			name:       "real_column_unquoted",
			table:      "t",
			setColumn:  "price",
			setColType: "real",
			setValue:   "9.99",
			want: []string{
				"WARNING",
				`SET "price" = 9.99`,
				`WHERE "price" = 9.99;`,
			},
		},
		{
			name:       "text_with_quote_escaped",
			table:      "t",
			setColumn:  "name",
			setColType: "text",
			setValue:   "O'Brien",
			want: []string{
				"WARNING",
				`SET "name" = 'O''Brien'`,
				`WHERE "name" = 'O''Brien';`,
			},
		},
		{
			name:         "where_uses_primary_key_when_provided",
			table:        "users",
			setColumn:    "name",
			setColType:   "text",
			setValue:     "bob",
			pkColumns:    []string{"id"},
			row:          map[string]string{"id": "1", "name": "alice"},
			colTypeByKey: map[string]string{"id": "integer", "name": "text"},
			want: []string{
				`UPDATE "users"`,
				`SET "name" = 'bob'`,
				`WHERE "id" = 1`,
			},
			notWant: []string{`WHERE "name"`},
		},
		{
			name:       "reserved_keyword_table_and_column_quoted",
			table:      "order",
			setColumn:  "select",
			setColType: "text",
			setValue:   "x",
			want: []string{
				"WARNING",
				`UPDATE "order"`,
				`SET "select" = 'x'`,
				`WHERE "select" = 'x';`,
			},
		},
		{
			name:       "identifier_with_double_quote_escaped",
			table:      "t",
			setColumn:  `foo"bar`,
			setColType: "text",
			setValue:   "v",
			want: []string{
				"WARNING",
				`"foo""bar"`,
				`SET "foo""bar" = 'v'`,
			},
		},
		{
			name:       "identifier_with_space_quoted",
			table:      "my table",
			setColumn:  "my column",
			setColType: "integer",
			setValue:   "1",
			want: []string{
				"WARNING",
				`UPDATE "my table"`,
				`SET "my column" = 1`,
				`WHERE "my column" = 1;`,
			},
		},
		{
			name:       "malicious_identifier_quoted_not_injection",
			table:      "t",
			setColumn:  `"; DROP TABLE t; --`,
			setColType: "text",
			setValue:   "x",
			want: []string{
				"WARNING",
				`SET "`,
				`= 'x'`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildUpdateQueryFromCell(tt.table, tt.setColumn, tt.setColType, tt.setValue, tt.pkColumns, tt.row, tt.colTypeByKey)
			for _, sub := range tt.want {
				if !strings.Contains(got, sub) {
					t.Errorf("BuildUpdateQueryFromCell(...) result must contain %q.\nGot:\n%s", sub, got)
				}
			}
			for _, sub := range tt.notWant {
				if strings.Contains(got, sub) {
					t.Errorf("BuildUpdateQueryFromCell(...) result must not contain %q.\nGot:\n%s", sub, got)
				}
			}
		})
	}
}
