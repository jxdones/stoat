package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/jxdones/stoat/internal/database"
	"github.com/jxdones/stoat/internal/ui/components/statusbar"
	"github.com/jxdones/stoat/internal/ui/components/table"
)

// EditorCellMsg is sent when the user closes the cell editor.
type EditorCellMsg struct {
	Value string
	Err   error
}

// handleEditKey handles the enter key to start inline cell editing when the table is focused.
func (m Model) handleEditKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
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

// handleDeleteKey handles the d key (and dd sequence) for deleting a row.
// prevKey is the key pressed immediately before this one, used to detect the dd sequence.
func (m Model) handleDeleteKey(msg tea.KeyPressMsg, prevKey string) (tea.Model, tea.Cmd, bool) {
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

// handleCellEditorKey handles the e key to open the external cell editor when the table is focused.
func (m Model) handleCellEditorKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
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

// confirmInlineEdit handles the enter key when in inline edit mode.
// It runs the update query and reloads the table if the value changed.
func (m Model) confirmInlineEdit(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
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

// handleEditModeKey dispatches key events while inline edit mode is active.
// Only esc (cancel), enter (confirm), and raw text input are processed.
func (m Model) handleEditModeKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		m.inlineEditMode = false
		m.applyViewState()
		return m, nil
	}
	if next, cmd, handled := m.confirmInlineEdit(msg); handled {
		return next, cmd
	}
	return m.delegateToFocused(msg)
}

// confirmDelete handles the y key press when a delete confirmation is pending.
func (m Model) confirmDelete(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
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

// cancelDelete handles the n or esc key press when a delete confirmation is pending.
func (m Model) cancelDelete(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
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

// onCellEditorDone handles the result of an external cell editor session.
func (m Model) onCellEditorDone(msg EditorCellMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		cmd := m.statusbar.SetStatusWithTTL(" Edit failed: "+msg.Err.Error(), statusbar.Error, 2*time.Second)
		return m, cmd
	}
	if m.tabs.ActiveTab() != "Records" {
		return m, nil
	}
	if m.readOnly {
		cmd := m.statusbar.SetStatusWithTTL(" Read-only mode: write queries are not allowed", statusbar.Warning, 3*time.Second)
		return m, cmd
	}
	if m.viewingQueryResult {
		cmd := m.statusbar.SetStatusWithTTL(" Query results are read-only", statusbar.Warning, 2*time.Second)
		return m, cmd
	}

	db := m.sidebar.EffectiveDB()
	tableName := m.sidebar.SelectedTable()
	activeRow, _ := m.table.ActiveRow()
	column, value, ok := m.table.ActiveCell()
	if !ok || db == "" || tableName == "" || msg.Value == value {
		return m, nil
	}
	colTypeByKey := make(map[string]string)
	for _, c := range m.table.Columns() {
		colTypeByKey[c.Key] = c.Type
	}

	m.pendingTableReload = true
	query := BuildUpdateQueryFromCell(m.schemaForWrites(), tableName, column.Key, column.Type, msg.Value, m.tablePKColumns, activeRow, colTypeByKey)
	spinnerCmd := m.statusbar.StartSpinner("Running update", statusbar.Info)
	return m, tea.Batch(spinnerCmd, RequestQueryRunCmd(query))
}

// OpenEditorWithCellValueCmd opens the editor with the given value and type name.
func OpenEditorWithCellValueCmd(value, typeName string) tea.Cmd {
	fileName := "stoat-cell-editor-*"
	isJson := false
	if strings.Contains(typeName, "json") {
		isJson = true
		fileName += ".json"
	} else {
		fileName += ".txt"
	}

	f, err := os.CreateTemp("", fileName)
	if err != nil {
		return func() tea.Msg {
			return EditorCellMsg{Err: err}
		}
	}

	if isJson {
		var v any
		if err := json.Unmarshal([]byte(value), &v); err == nil {
			if jsonValue, err := json.MarshalIndent(v, "", "  "); err == nil {
				value = string(jsonValue)
			}
		}
	}

	_, err = f.WriteString(value)
	if err != nil {
		f.Close()
		_ = os.Remove(f.Name())
		return func() tea.Msg {
			return EditorCellMsg{Err: err}
		}
	}

	path := f.Name()
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return func() tea.Msg {
			return EditorCellMsg{Err: err}
		}
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	cmd := exec.Command(editor, path)
	return tea.ExecProcess(cmd, func(execErr error) tea.Msg {
		content, readErr := os.ReadFile(path)
		_ = os.Remove(path)
		if execErr != nil {
			return EditorCellMsg{Err: execErr}
		}
		if readErr != nil {
			return EditorCellMsg{Err: readErr}
		}

		contentStr := strings.TrimRight(string(content), "\n")
		if isJson {
			var buf bytes.Buffer
			if err := json.Compact(&buf, []byte(contentStr)); err == nil {
				contentStr = buf.String()
			}
		}
		return EditorCellMsg{Value: contentStr, Err: nil}
	})
}

// schemaForWrites returns the SQL schema to use when generating write queries.
func (m Model) schemaForWrites() string {
	if m.source == nil || !m.source.UsesSchemaQualification() {
		return ""
	}
	return m.sidebar.EffectiveDB()
}

// BuildUpdateQueryFromCell builds a SQL UPDATE query from the selected cell.
func BuildUpdateQueryFromCell(schema, tableName, setColumn, setColType, setValue string, pkColumns []string, row map[string]string, colTypeByKey map[string]string) string {
	setLiteral := formatSQLValue(setColType, setValue)
	tbl := tableRef(schema, tableName)
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

// formatSQLValue returns the value formatted as a SQL literal for the given column type.
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
func quoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// tableRef returns a safe SQL table reference. When schema is non-empty the
// result is "schema"."table" (used for Postgres); otherwise just "table".
func tableRef(schema, tbl string) string {
	if schema == "" {
		return quoteIdentifier(tbl)
	}
	return quoteIdentifier(schema) + "." + quoteIdentifier(tbl)
}
