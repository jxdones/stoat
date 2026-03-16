package model

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/jxdones/stoat/internal/database"
	"github.com/jxdones/stoat/internal/ui/components/connectionpicker"
	"github.com/jxdones/stoat/internal/ui/components/sidebar"
	"github.com/jxdones/stoat/internal/ui/components/statusbar"
	"github.com/jxdones/stoat/internal/ui/components/table"
	"github.com/jxdones/stoat/internal/ui/datasource"
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
	m.table.SetRows(dbRowsToTable(pr.Result.Rows))
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
		m.table.SetRows(dbRowsToTable(msg.Result.Rows))
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
	spinnerCmd := m.statusbar.StartSpinner("Running query", statusbar.Info)
	return m, tea.Batch(spinnerCmd, RequestQueryRunCmd(query))
}

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
	db := m.sidebar.EffectiveDB()
	tableName := m.sidebar.SelectedTable()
	_, value, ok := m.table.ActiveCell()
	if !ok || db == "" || tableName == "" {
		return m, nil, false
	}
	m.inlineEditMode = true
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
	q := BuildUpdateQueryFromCell(tableName, col.Key, col.Type, newValue, pkColumns, activeRow, colTypeByKey)
	m.pendingTableReload = true
	spinnerCmd := m.statusbar.StartSpinner("Running update", statusbar.Info)
	return m, tea.Batch(spinnerCmd, RequestQueryRunCmd(q)), true
}

// keyHandler is a function that handles a key press and returns whether it was handled.
type keyHandler func(tea.KeyPressMsg) (tea.Model, tea.Cmd, bool)

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

// handleKeyPress handles the KeyPressMsg and updates the focused component.
func (m Model) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.inlineEditMode {
		return m.handleKeyPressInEditMode(msg)
	}

	if m.activeModal == modalConnectionPicker {
		return m.handleKeyPressInConnectionPicker(msg)
	}

	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "ctrl+e":
		return m.handleOpenEditor()
	case "ctrl+r":
		return m.handleReload(msg)
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
		m.helpExpanded = false
		m.applyViewState()
		return m, nil
	case "/":
		m.view.focus = FocusFilterbox
		m.applyViewState()
		return m, nil
	case "?":
		m.helpExpanded = !m.helpExpanded
		m.applyViewState()
		return m, nil
	case "c":
		if m.view.focus == FocusNone {
			m.OpenConnectionPicker()
			return m, nil
		}
		return m.handleUpdateFocused(msg)
	default:
		handlers := []keyHandler{
			m.handleApplyFilter,
			m.handleTabSwitch,
			m.handleUpdateFromCell,
			m.handleQueryShortcut,
			m.handlePagingShortcut,
			m.handleExpandSavedQuery,
			m.handleCopyCellValueFromTable,
		}
		for _, h := range handlers {
			if next, cmd, handled := h(msg); handled {
				return next, cmd
			}
		}
		return m.handleUpdateFocused(msg)
	}
}

// handleExpandSavedQuery handles Ctrl+N when the querybox is focused: tries to expand a saved query.
// If expansion happens it returns the updated model and true
func (m Model) handleExpandSavedQuery(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	if msg.String() != "ctrl+n" || !m.isFocused(FocusQuerybox) {
		return m, nil, false
	}
	next, expanded := m.ExpandSavedQuery()
	if !expanded {
		return m, nil, false
	}
	next.applyViewState()
	return next, nil, true
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
	spinnerCmd := m.statusbar.StartSpinner("Running query", statusbar.Info)
	return m, tea.Batch(spinnerCmd, RequestQueryRunCmd(query)), true
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
		spinnerCmd := m.statusbar.StartSpinner("Loading page", statusbar.Info)
		return m, tea.Batch(spinnerCmd, LoadTableRowsCmd(m.source, target, database.PageRequest{
			Limit: DefaultPageLimit,
			After: m.paging.currentAfter,
		})), true
	}
	if msg.String() == "ctrl+b" && len(m.paging.afterStack) > 1 {
		prevCursor := m.paging.afterStack[len(m.paging.afterStack)-2]
		m.setPendingPageNav(pageNavPrev)
		m.paging.requestAfter = prevCursor
		spinnerCmd := m.statusbar.StartSpinner("Loading page", statusbar.Info)
		return m, tea.Batch(spinnerCmd, LoadTableRowsCmd(m.source, target, database.PageRequest{
			Limit: DefaultPageLimit,
			After: prevCursor,
		})), true
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
						spinnerCmd := m.statusbar.StartSpinner("Loading tables", statusbar.Info)
						return m, tea.Batch(spinnerCmd, LoadTablesCmd(m.source, db))
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
			m.tableSchema = tableSchema{}
			m.tablePKColumns = nil
			m.tablePKTarget = database.DatabaseTarget{}
			target := database.DatabaseTarget{
				Database: db,
				Table:    tableName,
			}
			page := database.PageRequest{
				Limit: DefaultPageLimit,
				After: "",
			}
			spinnerCmd := m.statusbar.StartSpinner("Loading "+tableName, statusbar.Info)
			return m, tea.Batch(spinnerCmd, LoadTableRowsCmd(m.source, target, page))
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
		if m.inlineEditMode {
			next, cmd := m.editbox.Update(msg)
			m.editbox = next
			return m, cmd
		}
		if m.tabs.ActiveTab() != "Records" && m.tabs.ActiveTab() != "Foreign Keys" {
			next, cmd := m.schemaTable.Update(msg)
			m.schemaTable = next
			return m, cmd
		}
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

// handleApplyFilter handles the apply filter shortcut key press.
func (m Model) handleApplyFilter(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	if m.view.focus != FocusFilterbox || msg.String() != "enter" {
		return m, nil, false
	}
	if !m.HasConnection() {
		cmd := m.statusbar.SetStatusWithTTL(" No active connection", statusbar.Warning, 2*time.Second)
		return m, cmd, true
	}
	db := m.sidebar.EffectiveDB()
	tableName := m.sidebar.SelectedTable()
	if db == "" || tableName == "" || tableName == "(none)" {
		cmd := m.statusbar.SetStatusWithTTL(" No table selected", statusbar.Warning, 2*time.Second)
		return m, cmd, true
	}

	expression := strings.TrimSpace(m.filterbox.Value())
	m.view.focus = FocusTable
	m.applyViewState()

	if expression == "" {
		m.resetPaging()
		m.setPendingPageNav(pageNavNone)
		m.paging.requestAfter = ""
		target := database.DatabaseTarget{Database: db, Table: tableName}
		page := database.PageRequest{Limit: DefaultPageLimit, After: ""}
		spinnerCmd := m.statusbar.StartSpinner("Loading "+tableName, statusbar.Info)
		return m, tea.Batch(spinnerCmd, LoadTableRowsCmd(m.source, target, page)), true
	}

	filtered := filterRowsByExpression(m.table.Rows(), m.table.Columns(), expression)
	m.table.SetRows(filtered)
	cmd := m.statusbar.SetStatusWithTTL(
		fmt.Sprintf(" Filter: %d row(s) match", len(filtered)),
		statusbar.Info,
		2*time.Second,
	)
	return m, cmd, true
}

// handleReload handles the reload shortcut key press.
func (m Model) handleReload(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.String() != "ctrl+r" {
		return m, nil
	}
	db := m.sidebar.EffectiveDB()
	tableName := m.sidebar.SelectedTable()
	if db == "" || tableName == "" || tableName == "(none)" {
		return m, nil
	}
	target := database.DatabaseTarget{Database: db, Table: tableName}
	page := database.PageRequest{Limit: DefaultPageLimit, After: ""}
	spinnerCmd := m.statusbar.StartSpinner("Loading "+tableName, statusbar.Info)
	return m, tea.Batch(spinnerCmd, LoadTableRowsCmd(m.source, target, page))
}

// filterRowsByExpression filters the rows by the expression.
func filterRowsByExpression(rows []table.Row, columns []table.Column, expr string) []table.Row {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return rows
	}
	needle := strings.ToLower(expr)
	keys := make([]string, 0, len(columns))
	for _, c := range columns {
		keys = append(keys, c.Key)
	}
	filtered := make([]table.Row, 0, len(rows))
	for _, r := range rows {
		for _, k := range keys {
			if strings.Contains(strings.ToLower(r[k]), needle) {
				filtered = append(filtered, r)
				break
			}
		}
	}
	return filtered
}

// handleCopyCellValueFromTable handles the copy of active cell value from table shortcut key press.
func (m Model) handleCopyCellValueFromTable(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	if msg.String() != "y" || m.view.focus != FocusTable {
		return m, nil, false
	}

	_, cellValue, ok := m.table.ActiveCell()
	if !ok {
		return m, nil, false
	}

	return m, CopyToClipboardCmd(cellValue), true
}

// handleTabSwitch handles the tab switch shortcut key press.
func (m Model) handleTabSwitch(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	validKeys := []string{"ctrl+1", "ctrl+2", "ctrl+3", "ctrl+4", "ctrl+5"}
	if !slices.Contains(validKeys, msg.String()) || m.view.focus != FocusTable {
		return m, nil, false
	}
	var index int
	switch msg.String() {
	case "ctrl+1":
		index = 0
	case "ctrl+2":
		index = 1
	case "ctrl+3":
		index = 2
	case "ctrl+4":
		index = 3
	case "ctrl+5":
		index = 4
	default:
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

// handleConnecting sets the status bar to "Connecting…" and fires the async
// ConnectCmd. The extra message hop gives the UI one render cycle to paint
// the status before the blocking provider call starts.
func (m Model) handleConnecting(msg ConnectingMsg) (tea.Model, tea.Cmd) {
	spinnerCmd := m.statusbar.StartSpinner("Connecting to "+msg.cfg.Name, statusbar.Info)
	return m, tea.Batch(spinnerCmd, ConnectCmd(msg.cfg))
}

// handleConnected stores the established data source and begins loading databases.
// It immediately populates the sidebar with the default database so the user
// sees something as soon as the connection is established, without waiting for
// the full Databases() round-trip to complete. The real schema list loads in
// parallel and replaces this placeholder when it arrives.
func (m Model) handleConnected(msg ConnectedMsg) (tea.Model, tea.Cmd) {
	m.source = msg.source
	if m.debugOutput != nil {
		m.source = datasource.WithTiming(m.source, m.debugOutput)
	}
	defaultDB, err := m.source.DefaultDatabase(context.Background())
	if err != nil || defaultDB == "" {
		spinnerCmd := m.statusbar.StartSpinner("Loading databases", statusbar.Info)
		return m, tea.Batch(spinnerCmd, LoadDatabasesCmd(m.source))
	}
	m.sidebar.SetDatabases([]string{defaultDB})
	m.sidebar.OpenSelectedDatabase()
	spinnerCmd := m.statusbar.StartSpinner("Loading tables", statusbar.Info)
	return m, tea.Batch(
		spinnerCmd,
		LoadDatabasesCmd(m.source),
		LoadTablesCmd(m.source, defaultDB),
	)
}

// handleConnectionFailed shows a sticky error in the status bar.
func (m Model) handleConnectionFailed(msg ConnectionFailedMsg) (tea.Model, tea.Cmd) {
	m.statusbar.StopSpinner()
	m.statusbar.SetStatus(" Connection failed: "+msg.err.Error(), statusbar.Error)
	return m, nil
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

func (m Model) handleKeyPressInConnectionPicker(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	next, event := m.connectionPicker.Update(msg)
	m.connectionPicker = next

	switch event {
	case connectionpicker.EventSelected:
		selected := m.connectionPicker.Selected()
		cfg := database.Config{Name: selected.Name}
		switch database.DBMS(strings.ToLower(selected.Type)) {
		case database.DBMSPostgres:
			cfg.DBMS = database.DBMSPostgres
			cfg.Values = map[string]string{"dsn": selected.DSN}
		default:
			cfg.DBMS = database.DBMSSQLite
			cfg.Values = map[string]string{"path": selected.DSN}
		}
		m.SetPendingConfig(cfg)
		m.view.focus = FocusSidebar
		m.activeModal = modalNone
		return m, func() tea.Msg { return ConnectingMsg{cfg: cfg} }
	case connectionpicker.EventClosed:
		m.activeModal = modalNone
	}

	return m, nil
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
