package model

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/jxdones/stoat/internal/database"
	"github.com/jxdones/stoat/internal/ui/components/statusbar"
	"github.com/jxdones/stoat/internal/ui/components/table"
)

// handleDatabasesLoaded handles the DatabasesLoadedMsg and updates the sidebar.
func (m Model) handleDatabasesLoaded(msg DatabasesLoadedMsg) (tea.Model, tea.Cmd) {
	m.statusbar.StopSpinner()
	if msg.Err != nil {
		cmd := m.statusbar.SetStatusWithTTL(" Databases: "+msg.Err.Error(), statusbar.Error, 4*time.Second)
		return m, cmd
	}
	activeDB := m.sidebar.EffectiveDB()
	m.sidebar.SetDatabases(msg.Databases)
	if len(msg.Databases) > 0 {
		if activeDB != "" {
			m.sidebar.SelectDatabase(activeDB)
		}
		m.sidebar.OpenSelectedDatabase()
	}
	return m, nil
}

// handleTablesLoaded handles the TablesLoadedMsg and updates the sidebar.
func (m Model) handleTablesLoaded(msg TablesLoadedMsg) (tea.Model, tea.Cmd) {
	m.statusbar.StopSpinner()
	if msg.Err != nil {
		cmd := m.statusbar.SetStatusWithTTL(" Tables: "+msg.Err.Error(), statusbar.Error, 4*time.Second)
		return m, cmd
	}
	m.sidebar.SetTables(msg.Database, msg.Tables)
	if len(msg.Tables) == 0 {
		m.table.SetColumns(nil)
		m.table.SetRows(nil)
	}
	m.statusbar.SetStatus(" Ready", statusbar.Info)
	return m, nil
}

// handleRowsLoaded handles the RowsLoadedMsg and updates the table.
func (m Model) handleRowsLoaded(msg RowsLoadedMsg) (tea.Model, tea.Cmd) {
	m.statusbar.StopSpinner()
	if msg.Err != nil {
		cmd := m.statusbar.SetStatusWithTTL(" Rows: "+msg.Err.Error(), statusbar.Error, 4*time.Second)
		return m, cmd
	}
	m.viewingQueryResult = false
	m.queryResultPreview = ""
	m.statusbar.SetStatus(" Ready", statusbar.Info)
	pr := msg.Result
	m.applyPageResult(m.paging.requestAfter, pr.NextAfter, pr.HasMore)
	if len(pr.Result.Columns) > 0 {
		m.table.SetColumns(dbColumnsToTable(pr.Result.Columns))
		m.tableSchema.columns = pr.Result.Columns

		if m.tabs.ActiveTab() == "Columns" {
			m.schemaTable = table.New(schemaColumnsToTable(m.tableSchema.columns))
		}
	}
	m.unfilteredRows = dbRowsToTable(pr.Result.Rows)
	m.table.SetRows(m.unfilteredRows)
	m.table.GotoTop()
	m.applyViewState()
	target := database.DatabaseTarget{Database: m.sidebar.EffectiveDB(), Table: m.sidebar.SelectedTable()}
	return m, tea.Batch(
		LoadTableConstraintsCmd(m.source, target),
		LoadTableIndexesCmd(m.source, target),
		LoadTableForeignKeysCmd(m.source, target),
	)
}

// handleTableConstraintsLoaded stores primary key columns for the table so UPDATE-from-cell can build a safe WHERE.
func (m Model) handleTableConstraintsLoaded(msg TableConstraintsLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		return m, nil
	}
	m.tablePKTarget = msg.Target
	m.tableSchema.constraints = msg.Constraints
	m.tablePKColumns = nil

	if m.tabs.ActiveTab() == "Constraints" {
		m.schemaTable = table.New(schemaConstraintsToTable(m.tableSchema.constraints))
	}

	for _, c := range msg.Constraints {
		if c.Type == "PRIMARY KEY" && len(c.Columns) > 0 {
			m.tablePKColumns = append([]string(nil), c.Columns...)
			break
		}
	}
	return m, nil
}

// handleIndexesLoaded handles the IndexesLoadedMsg and updates the table schema.
func (m Model) handleIndexesLoaded(msg IndexesLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		return m, nil
	}
	target := database.DatabaseTarget{Database: m.sidebar.EffectiveDB(), Table: m.sidebar.SelectedTable()}

	if m.tabs.ActiveTab() == "Indexes" {
		m.schemaTable = table.New(schemaIndexesToTable(msg.Indexes))
	}

	if msg.Target == target {
		m.tableSchema.indexes = msg.Indexes
	}
	return m, nil
}

// handleForeignKeysLoaded handles the ForeignKeysLoadedMsg and updates the table schema.
func (m Model) handleForeignKeysLoaded(msg ForeignKeysLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		return m, nil
	}

	target := database.DatabaseTarget{Database: m.sidebar.EffectiveDB(), Table: m.sidebar.SelectedTable()}
	if msg.Target == target {
		m.tableSchema.foreignKeys = msg.ForeignKeys
		m.fkViewport.SetContent(m.fkViewportContent())
		m.fkViewport.GotoTop()
	}
	return m, nil
}

// handleQueryExecuted handles the QueryExecutedMsg and updates the status bar.
// Queries that return a result set (SELECT, or INSERT/UPDATE/DELETE ... RETURNING)
// replace the table with that result. Plain DML with no result set only shows
// the affected row count in the status bar and leaves the table as-is.
func (m Model) handleQueryExecuted(msg QueryExecutedMsg) (tea.Model, tea.Cmd) {
	m.statusbar.StopSpinner()
	if msg.Err != nil {
		cmd := m.statusbar.SetStatusWithTTL(" Query failed: "+msg.Err.Error(), statusbar.Error, 4*time.Second)
		return m, cmd
	}
	m.querybox.SetValue("")
	m.view.focus = FocusTable

	hasResultSet := len(msg.Result.Columns) > 0 || len(msg.Result.Rows) > 0
	if hasResultSet {
		m.viewingQueryResult = true
		m.queryResultPreview = queryPreviewForHeader(msg.Query)
		m.resetPaging()
		m.tablePKColumns = nil
		m.tablePKTarget = database.DatabaseTarget{}
		m.table.SetColumns(dbColumnsToTable(msg.Result.Columns))
		m.unfilteredRows = dbRowsToTable(msg.Result.Rows)
		m.table.SetRows(m.unfilteredRows)
		m.table.GotoTop()
		m.applyViewState()
		cmd := m.statusbar.SetStatusWithTTL(
			fmt.Sprintf(" Query ok: %d row(s) returned", len(msg.Result.Rows)),
			statusbar.Success,
			3*time.Second,
		)
		return m, cmd
	}

	// DML with no result set: show affected count; reload the table if the
	// query came from an inline edit so the new value is visible immediately.
	statusCmd := m.statusbar.SetStatusWithTTL(
		fmt.Sprintf(" Query ok: %d row(s) affected", msg.Result.RowsAffected),
		statusbar.Success,
		3*time.Second,
	)
	if m.pendingTableReload {
		m.pendingTableReload = false
		db := m.sidebar.EffectiveDB()
		tableName := m.sidebar.SelectedTable()
		if db != "" && tableName != "" {
			target := database.DatabaseTarget{Database: db, Table: tableName}
			page := database.PageRequest{Limit: DefaultPageLimit, After: m.paging.requestAfter}
			m.applyViewState()
			return m, tea.Batch(statusCmd, LoadTableRowsCmd(m.source, target, page))
		}
	}
	m.applyViewState()
	return m, statusCmd
}

// handleEditorQueryDone handles the EditorQueryMsg after the user closes the editor.
// If the query is non-empty, it is run via the same path as the query box.
func (m Model) handleEditorQueryDone(msg EditorQueryMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		cmd := m.statusbar.SetStatusWithTTL(" Edit cancelled: "+msg.Err.Error(), statusbar.Warning, 3*time.Second)
		return m, cmd
	}
	query := strings.TrimSpace(msg.Query)
	if query == "" {
		cmd := m.statusbar.SetStatusWithTTL(" Empty query", statusbar.Warning, 2*time.Second)
		return m, cmd
	}
	if !m.HasConnection() {
		cmd := m.statusbar.SetStatusWithTTL(" No active connection", statusbar.Warning, 2*time.Second)
		return m, cmd
	}
	if m.readOnly && m.isWriteQuery(query) {
		cmd := m.statusbar.SetStatusWithTTL(" Read-only mode: write queries are not allowed", statusbar.Warning, 3*time.Second)
		return m, cmd
	}
	spinnerCmd := m.statusbar.StartSpinner("Running query", statusbar.Info)
	return m, tea.Batch(spinnerCmd, RequestQueryRunCmd(query))
}

// queryPreviewForHeader returns a one-line, truncated preview of the query for the header.
func queryPreviewForHeader(query string) string {
	const queryPreviewMaxLen = 52
	line := strings.TrimSpace(query)
	if line == "" {
		return ""
	}

	fields := strings.Fields(line)
	line = strings.Join(fields, " ")
	if len(line) <= queryPreviewMaxLen {
		return line
	}
	return line[:queryPreviewMaxLen-1] + "…"
}
