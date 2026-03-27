package model

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jxdones/stoat/internal/ui/components/table"
)

// formatSQLValue returns the value formatted as a SQL literal for the given column type.
// Integer/real types are left unquoted; text and unknown types are single-quoted with
// single quotes escaped by doubling. Empty value is rendered as NULL.
func formatSQLValue(colType, value string) string {
	if value == table.NullValue {
		return "NULL"
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "NULL"
	}
	upper := strings.ToUpper(strings.TrimSpace(colType))
	switch {
	case strings.Contains(upper, "INT"), strings.Contains(upper, "NUMERIC"):
		if _, err := strconv.ParseInt(value, 10, 64); err == nil {
			return value
		}
		return "'" + strings.ReplaceAll(value, "'", "''") + "'"
	case strings.Contains(upper, "REAL"), strings.Contains(upper, "FLOAT"), strings.Contains(upper, "DOUBLE"):
		if _, err := strconv.ParseFloat(value, 64); err == nil {
			return value
		}
		return "'" + strings.ReplaceAll(value, "'", "''") + "'"
	default:
		return "'" + strings.ReplaceAll(value, "'", "''") + "'"
	}
}

// quoteIdentifier quotes a SQL identifier so it is safe to use in generated SQL.
// It wraps the name in double quotes and escapes any internal " by doubling it.
// This prevents invalid syntax for names with spaces/special chars and SQL keywords.
func quoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// tableRef returns a safe SQL table reference. When schema is non-empty the
// result is "schema"."table" (used for Postgres); otherwise just "table".
func tableRef(schema, table string) string {
	if schema == "" {
		return quoteIdentifier(table)
	}
	return quoteIdentifier(schema) + "." + quoteIdentifier(table)
}

// BuildUpdateQueryFromCell builds a SQL UPDATE query from the selected cell.
// schema is the SQL schema prefix; pass empty string for databases that do not
// use schema qualification (SQLite). setColumn/setColType/setValue are the column
// being edited and its new value. When pkColumns is non-empty and row has the
// current row, WHERE is built from primary key so only one row is updated.
// Otherwise WHERE uses the edited column (may match multiple rows).
// colTypeByKey maps column key to type for formatting literals (e.g. "id" -> "integer").
func BuildUpdateQueryFromCell(schema, table, setColumn, setColType, setValue string, pkColumns []string, row map[string]string, colTypeByKey map[string]string) string {
	setLiteral := formatSQLValue(setColType, setValue)
	tbl := tableRef(schema, table)
	col := quoteIdentifier(setColumn)

	var whereClause string
	usePK := len(pkColumns) > 0 && row != nil && colTypeByKey != nil
	if usePK {
		parts := make([]string, 0, len(pkColumns))
		for _, pk := range pkColumns {
			val := row[pk]
			typ := colTypeByKey[pk]
			parts = append(parts, quoteIdentifier(pk)+" = "+formatSQLValue(typ, val))
		}
		if len(parts) == len(pkColumns) {
			whereClause = "WHERE " + strings.Join(parts, " AND ") + ";"
		}
	}
	if whereClause == "" {
		oldLiteral := formatSQLValue(setColType, row[setColumn])
		whereClause = fmt.Sprintf("WHERE %s = %s;", col, oldLiteral)
	}

	lines := []string{
		fmt.Sprintf("UPDATE %s", tbl),
		fmt.Sprintf("SET %s = %s", col, setLiteral),
		whereClause,
		"",
	}
	if !usePK {
		lines = append([]string{"-- WARNING: WHERE uses the edited column; may match multiple rows. Use primary key for a single row.", ""}, lines...)
	}
	return strings.Join(lines, "\n")
}

// BuildDeleteQuery builds a SQL DELETE query for the active row.
// schema is the SQL schema prefix; pass empty string for databases that do not
// use schema qualification (SQLite). When pkColumns is non-empty, WHERE is built
// from primary key columns so only one row is deleted. Otherwise WHERE matches
// all column values in the row (may match multiple rows if data is not unique).
// colTypeByKey maps column key to type for formatting literals (e.g. "id" -> "integer").
func BuildDeleteQuery(schema, tableName string, pkColumns []string, row map[string]string, colTypeByKey map[string]string) string {
	tbl := tableRef(schema, tableName)

	var whereClause string
	usePK := len(pkColumns) > 0 && row != nil && colTypeByKey != nil
	if usePK {
		parts := make([]string, 0, len(pkColumns))
		for _, pk := range pkColumns {
			val := row[pk]
			typ := colTypeByKey[pk]
			parts = append(parts, quoteIdentifier(pk)+" = "+formatSQLValue(typ, val))
		}
		if len(parts) == len(pkColumns) {
			whereClause = "WHERE " + strings.Join(parts, " AND ") + ";"
		}
	}
	if whereClause == "" {
		parts := make([]string, 0, len(row))
		for col, val := range row {
			typ := colTypeByKey[col]
			parts = append(parts, quoteIdentifier(col)+" = "+formatSQLValue(typ, val))
		}
		whereClause = "WHERE " + strings.Join(parts, " AND ") + ";"
	}

	lines := []string{
		fmt.Sprintf("DELETE FROM %s", tbl),
		whereClause,
		"",
	}
	if !usePK {
		lines = append([]string{"-- WARNING: No primary key found; WHERE matches all column values. May delete multiple rows.", ""}, lines...)
	}
	return strings.Join(lines, "\n")
}
