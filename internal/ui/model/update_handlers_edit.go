package model

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/jxdones/stoat/internal/database"
	"github.com/jxdones/stoat/internal/ui/components/statusbar"
	"github.com/jxdones/stoat/internal/ui/components/table"
)

// handleUpdateFromCell handles the update from cell key press.
func (m Model) handleUpdateFromCell(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	if m.view.focus != FocusTable || msg.String() != "enter" {
		return m, nil, false
	}
	if m.tabs.ActiveTab() != "Records" {
		return m, nil, false
	}
	if m.viewingQueryResult {
		cmd := m.statusbar.SetStatusWithTTL(" Query results are read-only", statusbar.Warning, 2*time.Second)
		return m, cmd, true
	}

	if m.readOnly {
		cmd := m.statusbar.SetStatusWithTTL(" Read-only mode: write queries are not allowed", statusbar.Warning, 3*time.Second)
		return m, cmd, true
	}

	db := m.sidebar.EffectiveDB()
	tableName := m.sidebar.SelectedTable()
	_, value, ok := m.table.ActiveCell()
	if !ok || db == "" || tableName == "" {
		return m, nil, false
	}
	m.inlineEditMode = true
	if value == table.NullValue {
		value = ""
	}
	m.editbox.SetValue(value)
	m.applyViewState()
	return m, nil, true
}

// handleInlineEditConfirm handles the inline edit confirm key press.
// It runs the update query and reloads the table if the query was successful.
func (m Model) handleInlineEditConfirm(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	if !m.inlineEditMode || msg.String() != "enter" {
		return m, nil, false
	}
	newValue := m.editbox.Value()
	m.inlineEditMode = false
	m.applyViewState()

	db := m.sidebar.EffectiveDB()
	tableName := m.sidebar.SelectedTable()
	col, oldValue, ok := m.table.ActiveCell()
	if !ok || db == "" || tableName == "" {
		return m, nil, true
	}
	if newValue == oldValue {
		return m, nil, true
	}
	activeRow, _ := m.table.ActiveRow()
	colTypeByKey := make(map[string]string)
	for _, c := range m.table.Columns() {
		colTypeByKey[c.Key] = c.Type
	}
	var pkColumns []string
	target := database.DatabaseTarget{Database: db, Table: tableName}
	if target == m.tablePKTarget && len(m.tablePKColumns) > 0 {
		pkColumns = m.tablePKColumns
	}
	q := BuildUpdateQueryFromCell(m.schemaForWrites(), tableName, col.Key, col.Type, newValue, pkColumns, activeRow, colTypeByKey)
	m.pendingTableReload = true
	spinnerCmd := m.statusbar.StartSpinner("Running update", statusbar.Info)
	return m, tea.Batch(spinnerCmd, RequestQueryRunCmd(q)), true
}

// handleKeyPressInEditMode handles key events while inline edit mode is active.
// Only esc (cancel), enter (confirm), and raw text input are processed —
// all global navigation shortcuts are suppressed so keystrokes reach the input.
func (m Model) handleKeyPressInEditMode(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		m.inlineEditMode = false
		m.applyViewState()
		return m, nil
	}
	if next, cmd, handled := m.handleInlineEditConfirm(msg); handled {
		return next, cmd
	}
	return m.handleUpdateFocused(msg)
}

// handleOpenCellEditor opens the cell editor for the active cell.
func (m Model) handleOpenCellEditor(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	if msg.String() != "e" || m.view.focus != FocusTable {
		return m, nil, false
	}
	if m.tabs.ActiveTab() != "Records" {
		return m, nil, false
	}
	if m.viewingQueryResult {
		cmd := m.statusbar.SetStatusWithTTL(" Query results are read-only", statusbar.Warning, 2*time.Second)
		return m, cmd, true
	}

	column, value, ok := m.table.ActiveCell()
	if !ok {
		return m, nil, false
	}
	return m, OpenEditorWithCellValueCmd(value, column.Type), true
}

// handleUpdateCellFromEditor handles the update from cell editor.
func (m Model) handleUpdateCellFromEditor(msg EditorCellMsg) (tea.Model, tea.Cmd, bool) {
	if msg.Err != nil {
		cmd := m.statusbar.SetStatusWithTTL(" Edit failed: "+msg.Err.Error(), statusbar.Error, 2*time.Second)
		return m, cmd, true
	}
	if m.tabs.ActiveTab() != "Records" {
		return m, nil, false
	}
	if m.readOnly {
		cmd := m.statusbar.SetStatusWithTTL(" Read-only mode: write queries are not allowed", statusbar.Warning, 3*time.Second)
		return m, cmd, true
	}
	if m.viewingQueryResult {
		cmd := m.statusbar.SetStatusWithTTL(" Query results are read-only", statusbar.Warning, 2*time.Second)
		return m, cmd, true
	}

	db := m.sidebar.EffectiveDB()
	tableName := m.sidebar.SelectedTable()
	activeRow, _ := m.table.ActiveRow()
	column, value, ok := m.table.ActiveCell()
	if !ok || db == "" || tableName == "" || msg.Value == value {
		return m, nil, false
	}
	colTypeByKey := make(map[string]string)
	for _, c := range m.table.Columns() {
		colTypeByKey[c.Key] = c.Type
	}

	m.pendingTableReload = true
	query := BuildUpdateQueryFromCell(m.schemaForWrites(), tableName, column.Key, column.Type, msg.Value, m.tablePKColumns, activeRow, colTypeByKey)
	spinnerCmd := m.statusbar.StartSpinner("Running update", statusbar.Info)
	return m, tea.Batch(spinnerCmd, RequestQueryRunCmd(query)), true
}

// handleOpenEditor opens $EDITOR with a SQL comment template. When the user
// saves and closes, handleEditorQueryDone runs whatever was written.
func (m Model) handleOpenEditor() (tea.Model, tea.Cmd) {
	if !m.HasConnection() {
		cmd := m.statusbar.SetStatusWithTTL(" No active connection", statusbar.Warning, 2*time.Second)
		return m, cmd
	}
	template := "-- Write your SQL here, then save and close the editor to run it.\n\n"
	return m, OpenEditorWithQueryCmd(template)
}

// handleDeleteRow handles the dd key sequence for deleting a row.
// prevKey is the key pressed immediately before this one, used to detect the dd sequence.
func (m Model) handleDeleteRow(msg tea.KeyPressMsg, prevKey string) (tea.Model, tea.Cmd, bool) {
	if m.view.focus != FocusTable || msg.String() != "d" {
		return m, nil, false
	}
	if m.tabs.ActiveTab() != "Records" {
		return m, nil, false
	}
	if m.viewingQueryResult {
		cmd := m.statusbar.SetStatusWithTTL(" Query results are read-only", statusbar.Warning, 2*time.Second)
		return m, cmd, true
	}
	if m.readOnly {
		cmd := m.statusbar.SetStatusWithTTL(" Read-only mode: write queries are not allowed", statusbar.Warning, 3*time.Second)
		return m, cmd, true
	}
	if prevKey != "d" {
		m.lastKey = "d"
		return m, nil, true
	}
	_, ok := m.table.ActiveRow()
	if !ok {
		return m, nil, false
	}
	m.pendingDeleteConfirm = true
	m.applyViewState()
	return m, nil, true
}

// handleDeleteConfirm handles the y key press when a delete confirmation is pending.
func (m Model) handleDeleteConfirm(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	if !m.pendingDeleteConfirm || msg.String() != "y" {
		return m, nil, false
	}
	m.pendingDeleteConfirm = false
	m.applyViewState()

	db := m.sidebar.EffectiveDB()
	tableName := m.sidebar.SelectedTable()
	activeRow, ok := m.table.ActiveRow()
	if !ok || db == "" || tableName == "" {
		return m, nil, true
	}
	colTypeByKey := make(map[string]string)
	for _, c := range m.table.Columns() {
		colTypeByKey[c.Key] = c.Type
	}
	var pkColumns []string
	target := database.DatabaseTarget{Database: db, Table: tableName}
	if target == m.tablePKTarget && len(m.tablePKColumns) > 0 {
		pkColumns = m.tablePKColumns
	}
	q := BuildDeleteQuery(m.schemaForWrites(), tableName, pkColumns, activeRow, colTypeByKey)
	m.pendingTableReload = true
	spinnerCmd := m.statusbar.StartSpinner("Deleting row", statusbar.Info)
	return m, tea.Batch(spinnerCmd, RequestQueryRunCmd(q)), true
}

// handleDeleteCancel handles the n or esc key press when a delete confirmation is pending.
func (m Model) handleDeleteCancel(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	if !m.pendingDeleteConfirm {
		return m, nil, false
	}
	if msg.String() != "n" && msg.String() != "esc" {
		return m, nil, false
	}
	m.pendingDeleteConfirm = false
	m.applyViewState()
	return m, nil, true
}

// schemaForWrites returns the SQL schema to use when generating write queries
// (UPDATE, DELETE). For Postgres, this is the active sidebar schema so that
// generated queries use "schema"."table" and are not sensitive to search_path.
// For SQLite (and any source that does not use schema qualification) it returns
// empty string, leaving the table reference unqualified.
func (m Model) schemaForWrites() string {
	if m.source == nil || !m.source.UsesSchemaQualification() {
		return ""
	}
	return m.sidebar.EffectiveDB()
}
