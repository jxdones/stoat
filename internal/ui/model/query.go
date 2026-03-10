package model

import (
	"fmt"
	"strconv"
	"strings"
)

// formatSQLValue returns the value formatted as a SQL literal for the given column type.
// Integer/real types are left unquoted; text and unknown types are single-quoted with
// single quotes escaped by doubling. Empty value is rendered as NULL.
func formatSQLValue(colType, value string) string {
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

// quoteIdentifier quotes a SQLite identifier so it is safe to use in generated SQL.
// It wraps the name in double quotes and escapes any internal " by doubling it.
// This prevents invalid syntax for names with spaces/special chars and SQL keywords.
func quoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// BuildUpdateQueryFromCell builds a SQL UPDATE query from the selected cell.
// setColumn/setColType/setValue are the column being edited and its new value.
// When pkColumns is non-empty and row has the current row, WHERE is built from primary key
// so only one row is updated. Otherwise WHERE uses the edited column (may match multiple rows).
// colTypeByKey maps column key to type for formatting literals (e.g. "id" -> "integer").
func BuildUpdateQueryFromCell(table, setColumn, setColType, setValue string, pkColumns []string, row map[string]string, colTypeByKey map[string]string) string {
	setLiteral := formatSQLValue(setColType, setValue)
	tbl := quoteIdentifier(table)
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
		whereClause = fmt.Sprintf("WHERE %s = %s;", col, setLiteral)
	}

	lines := []string{
		"-- Generated from selected cell. Adjust as needed, then run it.",
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
