package model

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/atotto/clipboard"
	"github.com/jxdones/stoat/internal/database"
	"github.com/jxdones/stoat/internal/ui/components/statusbar"
	"github.com/jxdones/stoat/internal/ui/components/table"
	"github.com/jxdones/stoat/internal/ui/datasource"
	"github.com/jxdones/stoat/internal/ui/theme"
)

// DefaultPageLimit is the number of rows loaded per page for table data.
const DefaultPageLimit = 200

// DatabasesLoadedMsg is sent when a list of databases is loaded.
type DatabasesLoadedMsg struct {
	Databases     []string
	ConnectionSeq int
	Err           error
}

// TablesLoadedMsg is sent when a list of tables is loaded.
type TablesLoadedMsg struct {
	Database      string
	Tables        []string
	ConnectionSeq int
	Err           error
}

// RowsLoadedMsg is sent when a page of rows is loaded.
type RowsLoadedMsg struct {
	Result        database.PageResult
	ConnectionSeq int
	Err           error
}

// TableConstraintsLoadedMsg is sent when constraints for a table have been loaded.
type TableConstraintsLoadedMsg struct {
	Target        database.DatabaseTarget
	Constraints   []database.Constraint
	ConnectionSeq int
	Err           error
}

// IndexesLoadedMsg is sent when indexes for a table have been loaded.
type IndexesLoadedMsg struct {
	Target        database.DatabaseTarget
	Indexes       []database.Index
	ConnectionSeq int
	Err           error
}

// ForeignKeysLoadedMsg is sent when foreign keys for a table have been loaded.
type ForeignKeysLoadedMsg struct {
	Target        database.DatabaseTarget
	ForeignKeys   []database.ForeignKey
	ConnectionSeq int
	Err           error
}

// CopyDoneMsg is sent when the copy operation is done.
type CopyDoneMsg struct {
	Err error
}

// onDatabasesLoaded handles the DatabasesLoadedMsg and updates the sidebar.
func (m Model) onDatabasesLoaded(msg DatabasesLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.ConnectionSeq != m.connectionSeq {
		return m, nil
	}

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

// onTablesLoaded handles the TablesLoadedMsg and updates the sidebar.
func (m Model) onTablesLoaded(msg TablesLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.ConnectionSeq != m.connectionSeq {
		return m, nil
	}

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

// onRowsLoaded handles the RowsLoadedMsg and updates the table.
func (m Model) onRowsLoaded(msg RowsLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.ConnectionSeq != m.connectionSeq {
		return m, nil
	}

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
		LoadTableConstraintsCmd(m.source, target, m.connectionSeq),
		LoadTableIndexesCmd(m.source, target, m.connectionSeq),
		LoadTableForeignKeysCmd(m.source, target, m.connectionSeq),
	)
}

// onTableConstraintsLoaded stores primary key columns for the table so UPDATE-from-cell can build a safe WHERE.
func (m Model) onTableConstraintsLoaded(msg TableConstraintsLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.ConnectionSeq != m.connectionSeq {
		return m, nil
	}

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

// onIndexesLoaded handles the IndexesLoadedMsg and updates the table schema.
func (m Model) onIndexesLoaded(msg IndexesLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.ConnectionSeq != m.connectionSeq {
		return m, nil
	}

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

// onForeignKeysLoaded handles the ForeignKeysLoadedMsg and updates the table schema.
func (m Model) onForeignKeysLoaded(msg ForeignKeysLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.ConnectionSeq != m.connectionSeq {
		return m, nil
	}

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

// reload reloads the current table's rows from the first page.
func (m Model) reload() (tea.Model, tea.Cmd) {
	db := m.sidebar.EffectiveDB()
	tableName := m.sidebar.SelectedTable()
	if db == "" || tableName == "" || tableName == "(none)" {
		return m, nil
	}
	target := database.DatabaseTarget{Database: db, Table: tableName}
	page := database.PageRequest{Limit: DefaultPageLimit, After: ""}
	spinnerCmd := m.statusbar.StartSpinner("Loading "+tableName, statusbar.Info)
	return m, tea.Batch(spinnerCmd, LoadTableRowsCmd(m.source, target, page, m.connectionSeq))
}

// LoadDatabasesCmd returns a command that loads the list of databases from the
// data source. On completion it sends a DatabasesLoadedMsg.
func LoadDatabasesCmd(source datasource.DataSource, seq int) tea.Cmd {
	return func() tea.Msg {
		if source == nil {
			return DatabasesLoadedMsg{Err: database.ErrNoConnection}
		}
		dbs, err := source.Databases(context.Background())
		return DatabasesLoadedMsg{Databases: dbs, Err: err, ConnectionSeq: seq}
	}
}

// LoadTablesCmd returns a command that loads the list of tables for the given
// database. On completion it sends a TablesLoadedMsg.
func LoadTablesCmd(source datasource.DataSource, dbName string, seq int) tea.Cmd {
	return func() tea.Msg {
		if source == nil {
			return TablesLoadedMsg{Database: dbName, Err: database.ErrNoConnection}
		}
		tables, err := source.Tables(context.Background(), dbName)
		return TablesLoadedMsg{Database: dbName, Tables: tables, Err: err, ConnectionSeq: seq}
	}
}

// LoadTableConstraintsCmd returns a command that loads constraints for the given target.
func LoadTableConstraintsCmd(source datasource.DataSource, target database.DatabaseTarget, seq int) tea.Cmd {
	return func() tea.Msg {
		if source == nil {
			return TableConstraintsLoadedMsg{Err: database.ErrNoConnection}
		}
		constraints, err := source.Constraints(context.Background(), target)
		return TableConstraintsLoadedMsg{Target: target, Constraints: constraints, Err: err, ConnectionSeq: seq}
	}
}

// LoadTableIndexesCmd returns a command that loads indexes for the given target.
func LoadTableIndexesCmd(source datasource.DataSource, target database.DatabaseTarget, seq int) tea.Cmd {
	return func() tea.Msg {
		if source == nil {
			return IndexesLoadedMsg{Err: database.ErrNoConnection}
		}
		indexes, err := source.Indexes(context.Background(), target)
		return IndexesLoadedMsg{Target: target, Indexes: indexes, Err: err, ConnectionSeq: seq}
	}
}

// LoadTableForeignKeysCmd returns a command that loads foreign keys for the given target.
func LoadTableForeignKeysCmd(source datasource.DataSource, target database.DatabaseTarget, seq int) tea.Cmd {
	return func() tea.Msg {
		if source == nil {
			return ForeignKeysLoadedMsg{Err: database.ErrNoConnection}
		}
		foreignKeys, err := source.ForeignKeys(context.Background(), target)
		return ForeignKeysLoadedMsg{Target: target, ForeignKeys: foreignKeys, Err: err, ConnectionSeq: seq}
	}
}

// LoadTableRowsCmd returns a command that loads one page of rows for the given target.
func LoadTableRowsCmd(source datasource.DataSource, target database.DatabaseTarget, page database.PageRequest, seq int) tea.Cmd {
	return func() tea.Msg {
		if source == nil {
			return RowsLoadedMsg{Err: database.ErrNoConnection}
		}
		if page.Limit <= 0 {
			page.Limit = DefaultPageLimit
		}
		result, err := source.Rows(context.Background(), target, page)
		return RowsLoadedMsg{Result: result, Err: err, ConnectionSeq: seq}
	}
}

// CopyToClipboardCmd returns a command that copies the given value to the clipboard.
func CopyToClipboardCmd(value string) tea.Cmd {
	return func() tea.Msg {
		err := clipboard.WriteAll(value)
		return CopyDoneMsg{Err: err}
	}
}

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

// fkViewportContent builds the styled string content for the foreign keys viewport.
func (m Model) fkViewportContent() string {
	content := []string{}
	columnStyle := lipgloss.NewStyle().Foreground(theme.Current.TextAccent)
	arrowStyle := lipgloss.NewStyle().Foreground(theme.Current.TextMuted)
	refStyle := lipgloss.NewStyle().Foreground(theme.Current.TextPrimary)
	actionLabelStyle := lipgloss.NewStyle().Foreground(theme.Current.TextMuted)
	actionValueStyle := lipgloss.NewStyle().Foreground(theme.Current.TextWarning).Bold(true)

	for _, fk := range m.tableSchema.foreignKeys {
		indent := strings.Repeat(" ", len(fk.Column)+5)
		line := fmt.Sprintf(
			"%s %s %s.%s\n",
			columnStyle.Render(fk.Column),
			arrowStyle.Render("→"),
			refStyle.Render(fk.RefTable),
			refStyle.Render(fk.RefColumn),
		)
		if fk.OnDeleteAction != "" {
			line += fmt.Sprintf(
				"%s%s %s\n",
				indent,
				actionLabelStyle.Render("on DELETE:"),
				actionValueStyle.Render(fk.OnDeleteAction),
			)
		}
		if fk.OnUpdateAction != "" {
			line += fmt.Sprintf(
				"%s%s %s\n",
				indent,
				actionLabelStyle.Render("on UPDATE:"),
				actionValueStyle.Render(fk.OnUpdateAction),
			)
		}
		line += "\n"
		content = append(content, line)
	}
	return strings.Join(content, "\n")
}
