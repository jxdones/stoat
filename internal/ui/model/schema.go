package model

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/jxdones/stoat/internal/database"
	"github.com/jxdones/stoat/internal/ui/components/celldetail"
	"github.com/jxdones/stoat/internal/ui/components/table"
)

// handleSchemaTabKey handles ctrl+1..5 to switch between schema tabs when the table is focused.
func (m Model) handleSchemaTabKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	validKeys := map[string]int{
		"ctrl+1": 0, "ctrl+2": 1, "ctrl+3": 2, "ctrl+4": 3, "ctrl+5": 4,
	}
	index, ok := validKeys[msg.String()]
	if !ok || m.view.focus != FocusTable {
		return m, nil, false
	}

	m.tabs.SetActive(index)
	switch index {
	case 1:
		cols, rows := schemaColumnsToTable(m.tableSchema.columns)
		m.schemaTable = table.New(cols, rows)
	case 2:
		cols, rows := schemaConstraintsToTable(m.tableSchema.constraints)
		m.schemaTable = table.New(cols, rows)
	case 4:
		cols, rows := schemaIndexesToTable(m.tableSchema.indexes)
		m.schemaTable = table.New(cols, rows)
	}
	m.applyViewState()
	return m, nil, true
}

// handleCopyKey handles the y key to copy the active cell value when the table is focused.
func (m Model) handleCopyKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	if msg.String() != "y" || m.view.focus != FocusTable {
		return m, nil, false
	}

	_, cellValue, ok := m.table.ActiveCell()
	if !ok {
		return m, nil, false
	}

	if cellValue == table.NullValue {
		cellValue = ""
	}
	return m, CopyToClipboardCmd(cellValue), true
}

// handleCellDetailKey handles the v key to open the cell detail modal when the table is focused.
func (m Model) handleCellDetailKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	if msg.String() != "v" || m.view.focus != FocusTable {
		return m, nil, false
	}
	column, value, ok := m.table.ActiveCell()
	if !ok {
		return m, nil, false
	}
	cd := m.cellDetail.SetContent(column.Key, column.Type, value)
	m.cellDetail = cd
	m.activeModal = modalCellDetail
	return m, nil, true
}

// handleCellDetailModalKey handles key presses while the cell detail modal is active.
func (m Model) handleCellDetailModalKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	next, event := m.cellDetail.Update(msg)
	m.cellDetail = next

	switch event {
	case celldetail.EventClosed:
		m.activeModal = modalNone
	case celldetail.EventNone:
		return m, nil
	}
	return m, nil
}

// schemaIndexesToTable converts a database.Index slice to table columns and rows.
func schemaIndexesToTable(indexes []database.Index) ([]table.Column, []table.Row) {
	columns := []table.Column{
		{Key: "name", Title: "Name", MinWidth: 20, Order: 1},
		{Key: "unique", Title: "Unique", MinWidth: 10, Order: 2},
		{Key: "columns", Title: "Columns", MinWidth: 20, Order: 3},
	}

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

// schemaConstraintsToTable converts a database.Constraint slice to table columns and rows.
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

// schemaColumnsToTable converts a database.Column slice to table columns and rows.
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
