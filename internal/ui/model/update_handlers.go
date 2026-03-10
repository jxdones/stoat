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
		m.table.SetColumns(dbColumnsToTable(msg.Result.Columns))
		m.table.SetRows(dbRowsToTable(msg.Result.Rows))
		m.applyViewState()
		cmd := m.statusbar.SetStatusWithTTL(
			fmt.Sprintf(" Query ok: %d rows (%d affected)", len(msg.Result.Rows), msg.Result.RowsAffected),
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
