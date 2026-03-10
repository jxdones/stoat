package model

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/jxdones/stoat/internal/database"
	"github.com/jxdones/stoat/internal/ui/components/sidebar"
	"github.com/jxdones/stoat/internal/ui/components/statusbar"
)

// handleDatabasesLoaded handles the DatabasesLoadedMsg and updates the sidebar.
func (m Model) handleDatabasesLoaded(msg DatabasesLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		cmd := m.statusbar.SetStatusWithTTL(" Databases: "+msg.Err.Error(), statusbar.Error, 4*time.Second)
		return m, cmd
	}
	m.sidebar.SetDatabases(msg.Databases)
	if len(msg.Databases) > 0 {
		m.sidebar.OpenSelectedDatabase()
		return m, LoadTablesCmd(m.source, msg.Databases[0])
	}
	return m, nil
}

// handleTablesLoaded handles the TablesLoadedMsg and updates the sidebar.
func (m Model) handleTablesLoaded(msg TablesLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		cmd := m.statusbar.SetStatusWithTTL(" Tables: "+msg.Err.Error(), statusbar.Error, 4*time.Second)
		return m, cmd
	}
	m.sidebar.SetTables(msg.Database, msg.Tables)
	return m, nil
}

// handleRowsLoaded handles the RowsLoadedMsg and updates the table.
func (m Model) handleRowsLoaded(msg RowsLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		cmd := m.statusbar.SetStatusWithTTL(" Rows: "+msg.Err.Error(), statusbar.Error, 4*time.Second)
		return m, cmd
	}
	pr := msg.Result
	m.applyPageResult(m.paging.requestAfter, pr.NextAfter, pr.HasMore)
	if len(pr.Result.Columns) > 0 {
		m.table.SetColumns(dbColumnsToTable(pr.Result.Columns))
	}
	m.table.SetRows(dbRowsToTable(pr.Result.Rows))
	m.applyViewState()
	target := database.DatabaseTarget{Database: m.sidebar.EffectiveDB(), Table: m.sidebar.SelectedTable()}
	return m, LoadTableConstraintsCmd(m.source, target)
}

// handleTableConstraintsLoaded stores primary key columns for the table so UPDATE-from-cell can build a safe WHERE.
func (m Model) handleTableConstraintsLoaded(msg TableConstraintsLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		return m, nil
	}
	m.tablePKTarget = msg.Target
	m.tablePKColumns = nil
	for _, c := range msg.Constraints {
		if c.Type == "PRIMARY KEY" && len(c.Columns) > 0 {
			m.tablePKColumns = append([]string(nil), c.Columns...)
			break
		}
	}
	return m, nil
}

// handleQueryExecuted handles the QueryExecutedMsg and updates the status bar.
// Queries that return a result set (SELECT, or INSERT/UPDATE/DELETE ... RETURNING)
// replace the table with that result. Plain DML with no result set only shows
// the affected row count in the status bar and leaves the table as-is.
func (m Model) handleQueryExecuted(msg QueryExecutedMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		cmd := m.statusbar.SetStatusWithTTL(" Query failed: "+msg.Err.Error(), statusbar.Error, 4*time.Second)
		return m, cmd
	}
	m.querybox.SetValue("")
	m.view.focus = FocusTable

	hasResultSet := len(msg.Result.Columns) > 0 || len(msg.Result.Rows) > 0
	if hasResultSet {
		m.resetPaging()
		m.tablePKColumns = nil
		m.tablePKTarget = database.DatabaseTarget{}
		m.table.SetColumns(dbColumnsToTable(msg.Result.Columns))
		m.table.SetRows(dbRowsToTable(msg.Result.Rows))
		m.applyViewState()
		cmd := m.statusbar.SetStatusWithTTL(
			fmt.Sprintf(" Query ok: %d row(s) returned", len(msg.Result.Rows)),
			statusbar.Success,
			3*time.Second,
		)
		return m, cmd
	}

	// DML with no result set: show affected count only; table stays as-is.
	m.applyViewState()
	return m, m.statusbar.SetStatusWithTTL(
		fmt.Sprintf(" Query ok: %d row(s) affected", msg.Result.RowsAffected),
		statusbar.Success,
		3*time.Second,
	)
}

// handleWindowSize handles the WindowSizeMsg and updates the view state.
func (m Model) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.view.width = msg.Width
	m.view.height = msg.Height
	m.applyViewState()
	return m, nil
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
	m.statusbar.SetStatus(" Running query...", statusbar.Info)
	return m, RequestQueryRunCmd(query)
}

func (m Model) handleUpdateFromCell(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	if m.view.focus != FocusTable || msg.String() != "enter" {
		return m, nil, false
	}
	db := m.sidebar.EffectiveDB()
	tableName := m.sidebar.SelectedTable()
	col, value, ok := m.table.ActiveCell()
	if !ok || db == "" || tableName == "" {
		return m, nil, false
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
	q := BuildUpdateQueryFromCell(tableName, col.Key, col.Type, value, pkColumns, activeRow, colTypeByKey)
	return m, OpenEditorWithQueryCmd(q), true
}

// handleKeyPress handles the KeyPressMsg and updates the focused component.
func (m Model) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "q":
		if m.view.focus == FocusNone {
			return m, tea.Quit
		}
		return m.handleUpdateFocused(msg)
	case "tab":
		m.focusNext()
		m.applyViewState()
		return m, nil
	case "shift+tab":
		m.focusPrevious()
		m.applyViewState()
		return m, nil
	case "esc":
		m.view.focus = FocusNone
		m.applyViewState()
		return m, nil
	default:
		if next, cmd, handled := m.handleUpdateFromCell(msg); handled {
			return next, cmd
		}
		if next, cmd, handled := m.handleQueryShortcut(msg); handled {
			return next, cmd
		}
		if next, cmd, handled := m.handlePagingShortcut(msg); handled {
			return next, cmd
		}
		return m.handleUpdateFocused(msg)
	}
}

// handleQueryShortcut handles the query shortcut key press.
func (m Model) handleQueryShortcut(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	if msg.String() != "ctrl+s" || m.view.focus != FocusQuerybox {
		return m, nil, false
	}
	if !m.HasConnection() {
		cmd := m.statusbar.SetStatusWithTTL(" No active connection", statusbar.Warning, 2*time.Second)
		return m, cmd, true
	}
	query := strings.TrimSpace(m.querybox.Value())
	if query == "" {
		cmd := m.statusbar.SetStatusWithTTL(" Query is empty", statusbar.Warning, 2*time.Second)
		return m, cmd, true
	}
	m.statusbar.SetStatus(" Running query...", statusbar.Info)
	return m, RequestQueryRunCmd(query), true
}

// handlePagingShortcut handles the paging shortcut key press.
func (m Model) handlePagingShortcut(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	// Next/prev page when table is focused (data paging).
	if m.view.focus != FocusTable || !m.HasConnection() {
		return m, nil, false
	}

	db := m.sidebar.EffectiveDB()
	tableName := m.sidebar.SelectedTable()
	target := database.DatabaseTarget{Database: db, Table: tableName}

	if msg.String() == "ctrl+n" && m.paging.currentHasMore {
		m.setPendingPageNav(pageNavNext)
		m.paging.requestAfter = m.paging.currentAfter
		return m, LoadTableRowsCmd(m.source, target, database.PageRequest{
			Limit: DefaultPageLimit,
			After: m.paging.currentAfter,
		}), true
	}
	if msg.String() == "ctrl+b" && len(m.paging.afterStack) > 1 {
		prevCursor := m.paging.afterStack[len(m.paging.afterStack)-2]
		m.setPendingPageNav(pageNavPrev)
		m.paging.requestAfter = prevCursor
		return m, LoadTableRowsCmd(m.source, target, database.PageRequest{
			Limit: DefaultPageLimit,
			After: prevCursor,
		}), true
	}
	return m, nil, false
}

// handleUpdateFocused forwards the message to the focused panel and returns the updated model and any command.
func (m Model) handleUpdateFocused(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.view.focus {
	case FocusSidebar:
		next, ev := m.sidebar.Update(msg)
		m.sidebar = next
		switch ev {
		case sidebar.EventOpenRequested:
			if !m.sidebar.InTablesSection() {
				m.sidebar.OpenSelectedDatabase()
				m.applyViewState()
				if m.HasConnection() {
					if db := m.sidebar.EffectiveDB(); db != "" {
						return m, LoadTablesCmd(m.source, db)
					}
				}
				return m, nil
			}
			tableName := m.sidebar.SelectedTable()
			if tableName == "" || tableName == "(none)" {
				m.view.focus = FocusTable
				m.applyViewState()
				return m, nil
			}
			m.view.focus = FocusTable
			m.applyViewState()
			if !m.HasConnection() {
				return m, nil
			}
			db := m.sidebar.EffectiveDB()
			m.resetPaging()
			m.setPendingPageNav(pageNavNone)
			m.paging.requestAfter = ""
			target := database.DatabaseTarget{
				Database: db,
				Table:    tableName,
			}
			page := database.PageRequest{
				Limit: DefaultPageLimit,
				After: "",
			}
			return m, LoadTableRowsCmd(m.source, target, page)
		}
	case FocusQuerybox:
		next, cmd := m.querybox.Update(msg)
		m.querybox = next
		return m, cmd
	case FocusFilterbox:
		next, cmd := m.filterbox.Update(msg)
		m.filterbox = next
		return m, cmd
	case FocusTable:
		next, cmd := m.table.Update(msg)
		m.table = next
		return m, cmd
	}
	return m, nil
}

// handlePasteMsg handles the PasteMsg and updates the focused component.
func (m Model) handlePasteMsg(msg tea.PasteMsg) (tea.Model, tea.Cmd) {
	switch m.view.focus {
	case FocusQuerybox:
		next, cmd := m.querybox.Update(msg)
		m.querybox = next
		return m, cmd
	}
	return m, nil
}
