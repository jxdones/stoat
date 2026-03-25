package database

import (
	"context"
	"database/sql"
	"sort"
	"strings"
)

// PaginationMode is the strategy for paginating a table (integer PK, or offset).
type PaginationMode int

const (
	PaginationByIntegerPK PaginationMode = iota
	PaginationByOffset
)

// RowsPagePlan holds the SQL and metadata for one page of table rows.
type RowsPagePlan struct {
	Mode       PaginationMode
	Query      string
	ScanOffset int
	AfterValue int64
}

// ColumnInfo holds column metadata for one table column.
type ColumnInfo struct {
	Name    string
	Type    string
	PKOrder int
}

// OrderedPrimaryKeyColumns returns the primary key column names in table order.
// Non-PK columns and columns with empty names are ignored.
func OrderedPrimaryKeyColumns(columnsInfo []ColumnInfo) []string {
	orderToColumn := make(map[int]string, len(columnsInfo))
	for _, column := range columnsInfo {
		if column.PKOrder <= 0 {
			continue
		}
		name := strings.TrimSpace(column.Name)
		if name == "" {
			continue
		}
		orderToColumn[column.PKOrder] = name
	}

	ordered := make([]int, 0, len(orderToColumn))
	for order := range orderToColumn {
		ordered = append(ordered, order)
	}
	sort.Ints(ordered)
	orderedColumns := make([]string, 0, len(ordered))
	for _, order := range ordered {
		orderedColumns = append(orderedColumns, orderToColumn[order])
	}
	return orderedColumns
}

// ColumnType returns the declared type of the given column.
func ColumnType(columnsInfo []ColumnInfo, name string) string {
	for _, column := range columnsInfo {
		if strings.EqualFold(strings.TrimSpace(column.Name), name) {
			return strings.TrimSpace(column.Type)
		}
	}
	return ""
}

// BuildOutputColumns builds the display column list for the given table columns.
func BuildOutputColumns(columnsInfo []ColumnInfo) []Column {
	columns := make([]Column, 0, len(columnsInfo))
	for i, column := range columnsInfo {
		declaredType := strings.TrimSpace(column.Type)
		if declaredType == "" {
			declaredType = "text"
		}
		columns = append(columns, Column{
			Key:      column.Name,
			Title:    column.Name,
			Type:     strings.ToLower(declaredType),
			MinWidth: ColumnMinWidth(len([]rune(column.Name))),
			Order:    i + 1,
		})
	}
	return columns
}

// ScanRowsPageResult executes the plan's query and returns up to pageLimit rows.
// The boolean reports whether more rows exist; the string is an opaque cursor for the next page.
func ScanRowsPageResult(ctx context.Context, db *sql.DB, plan RowsPagePlan, columnNames []string, pageLimit int) ([]Row, bool, string, error) {
	if db == nil {
		return nil, false, "", ErrNoConnection
	}

	rows, err := db.QueryContext(ctx, plan.Query)
	if err != nil {
		return nil, false, "", err
	}
	defer rows.Close()

	values, targets := MakeScanBuffers(len(columnNames) + plan.ScanOffset)
	outputRows := make([]Row, 0, pageLimit)
	hasMore := false
	nextAfter := ""

	for rows.Next() {
		if err := rows.Scan(targets...); err != nil {
			return nil, false, "", err
		}
		if len(outputRows) >= pageLimit {
			hasMore = true
			break
		}
		switch plan.Mode {
		case PaginationByOffset:
			nextAfter = FormatCursor("off", plan.AfterValue+int64(len(outputRows))+OffsetCursorSkipCurrentRow)
		case PaginationByIntegerPK:
			nextAfter = FormatCursor("pk", AsInt64(values[0]))
		}
		row := make(Row, len(columnNames))
		for i, name := range columnNames {
			row[name] = AsString(values[i+plan.ScanOffset])
		}
		outputRows = append(outputRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, false, "", err
	}
	return outputRows, hasMore, nextAfter, nil
}
