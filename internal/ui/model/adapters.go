package model

import (
	"strings"

	"github.com/jxdones/stoat/internal/database"
	"github.com/jxdones/stoat/internal/ui/components/table"
)

// dbColumnsToTable converts a database.Column slice to a table.Column slice.
func dbColumnsToTable(cols []database.Column) []table.Column {
	out := make([]table.Column, len(cols))
	for i, c := range cols {
		out[i] = table.Column{
			Key:      c.Key,
			Title:    c.Title,
			Type:     c.Type,
			MinWidth: c.MinWidth,
			Order:    c.Order,
		}
	}
	return out
}

// dbRowsToTable converts a database.Row slice to a table.Row slice.
func dbRowsToTable(rows []database.Row) []table.Row {
	out := make([]table.Row, len(rows))
	for i, r := range rows {
		out[i] = table.Row(r)
	}
	return out
}

// schemaIndexesToTable converts a database.Index slice to a table.Column and table.Row slice.
func schemaIndexesToTable(indexes []database.Index) ([]table.Column, []table.Row) {
	columns := []table.Column{
		{Key: "name", Title: "Name", MinWidth: 20, Order: 1},
		{Key: "unique", Title: "Unique", MinWidth: 10, Order: 2},
		{Key: "columns", Title: "Columns", MinWidth: 20, Order: 3},
	}

	// Helper function to convert a boolean to a string.
	unique := func(b bool) string {
		if b {
			return "YES"
		}
		return "NO"
	}

	rows := make([]table.Row, len(indexes))
	for i, idx := range indexes {
		rows[i] = table.Row{
			"name":    idx.Name,
			"unique":  unique(idx.Unique),
			"columns": strings.Join(idx.Columns, ", "),
		}
	}
	return columns, rows
}

// schemaConstraintsToTable converts a database.Constraint slice to a table.Column and table.Row slice.
func schemaConstraintsToTable(constraints []database.Constraint) ([]table.Column, []table.Row) {
	columns := []table.Column{
		{Key: "name", Title: "Name", MinWidth: 20, Order: 1},
		{Key: "type", Title: "Type", MinWidth: 10, Order: 2},
		{Key: "columns", Title: "Columns", MinWidth: 20, Order: 3},
	}

	rows := make([]table.Row, len(constraints))
	for i, c := range constraints {
		rows[i] = table.Row{
			"name":    c.Name,
			"type":    c.Type,
			"columns": strings.Join(c.Columns, ", "),
		}
	}
	return columns, rows
}

// schemaColumnsToTable converts a database.Column slice to a table.Column and table.Row slice.
func schemaColumnsToTable(cols []database.Column) ([]table.Column, []table.Row) {
	columns := []table.Column{
		{Key: "name", Title: "Name", MinWidth: 20, Order: 1},
		{Key: "type", Title: "Type", MinWidth: 10, Order: 2},
	}

	rows := make([]table.Row, len(cols))
	for i, col := range cols {
		rows[i] = table.Row{
			"name": col.Key,
			"type": col.Type,
		}
	}
	return columns, rows
}
