package database

import (
	"fmt"
	"strconv"
	"strings"
)

// Constants for column display.
const (
	MinColumnWidth = 8
	MaxColumnWidth = 24
	ColumnNamePad  = 2
)

// Constants for cursor parsing and conversion.
const (
	decimalBase  = 10
	int64BitSize = 64
)

// offsetCursorSkipCurrentRow is added to the offset when building the next-page cursor
// so the next page starts after the current row.
const OffsetCursorSkipCurrentRow = 1

// makeScanBuffers allocates n value slots and n pointers (targets[i] == &values[i]) for rows.Scan.
// After calling rows.Scan(targets...), the row data lives in values.
func MakeScanBuffers(n int) ([]any, []any) {
	values := make([]any, n)
	targets := make([]any, n)
	for i := range values {
		targets[i] = &values[i]
	}
	return values, targets
}

// AsString converts a scanned value to a string for display.
func AsString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case []byte:
		return string(t)
	case float64:
		if float64(int64(t)) == t {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case int64:
		return strconv.FormatInt(t, 10)
	case int:
		return strconv.Itoa(t)
	case bool:
		if t {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", t)
	}
}

// AsInt64 converts a scanned value to int64 for cursor use.
func AsInt64(v any) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case int64:
		return t
	case int:
		return int64(t)
	case []byte:
		n, _ := strconv.ParseInt(strings.TrimSpace(string(t)), 10, 64)
		return n
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		return n
	default:
		return 0
	}
}

// HasIntegerAffinity checks if the declared type has integer affinity.
func HasIntegerAffinity(declaredType string) bool {
	return strings.Contains(strings.ToUpper(strings.TrimSpace(declaredType)), "INT")
}

// QuoteIdentifier quotes an identifier for use in an SQL query.
func QuoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// QuoteSQLString quotes a string for use in an SQL query.
func QuoteSQLString(value string) string {
	return `'` + strings.ReplaceAll(value, `'`, `''`) + `'`
}

// UniqueCursorAlias returns an alias for the cursor column that does not conflict with existing column names.
func UniqueCursorAlias(columns []string) string {
	used := make(map[string]struct{}, len(columns))
	for _, column := range columns {
		used[strings.ToLower(strings.TrimSpace(column))] = struct{}{}
	}
	alias := "__cursor"
	for {
		if _, ok := used[strings.ToLower(alias)]; !ok {
			return alias
		}
		alias += "_"
	}
}

// ParseCursor parses the cursor string and returns the integer value for the given prefix.
// An empty cursor returns 0, nil.
func ParseCursor(cursor string, expectedPrefix string) (int64, error) {
	raw := strings.TrimSpace(cursor)
	if raw == "" {
		return 0, nil
	}
	if n, err := strconv.ParseInt(raw, decimalBase, int64BitSize); err == nil {
		return n, nil
	}

	prefix := expectedPrefix + ":"
	if !strings.HasPrefix(raw, prefix) {
		return 0, fmt.Errorf("invalid cursor %q for mode %s", raw, expectedPrefix)
	}
	n, err := strconv.ParseInt(strings.TrimSpace(strings.TrimPrefix(raw, prefix)), decimalBase, int64BitSize)
	if err != nil {
		return 0, fmt.Errorf("invalid cursor %q for mode %s: %w", raw, expectedPrefix, err)
	}
	return n, nil
}

// FormatCursor formats a cursor as "prefix:n".
func FormatCursor(prefix string, n int64) string {
	return fmt.Sprintf("%s:%d", prefix, n)
}

// PrimaryKeyOrderExpr returns an ORDER BY expression for the primary key columns, or "1" if none.
func PrimaryKeyOrderExpr(primaryKeyColumns []string) string {
	if len(primaryKeyColumns) == 0 {
		return "1"
	}
	out := make([]string, 0, len(primaryKeyColumns))
	for _, column := range primaryKeyColumns {
		out = append(out, QuoteIdentifier(column))
	}
	return strings.Join(out, ", ")
}

// ColumnMinWidth returns the minimum display width for a column based on the
// header name length, clamped between MinColumnWidth and MaxColumnWidth.
func ColumnMinWidth(headerLen int) int {
	return max(MinColumnWidth, min(MaxColumnWidth, headerLen+ColumnNamePad))
}
