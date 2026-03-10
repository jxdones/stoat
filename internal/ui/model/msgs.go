package model

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/jxdones/stoat/internal/database"
	"github.com/jxdones/stoat/internal/ui/datasource"
)

// DefaultPageLimit is the number of rows loaded per page for table data.
const DefaultPageLimit = 200

// Messages returned by async load commands. Handle these in Model.Update to
// update the sidebar, table, and paging state.
type (
	DatabasesLoadedMsg struct {
		Databases []string
		Err       error
	}
	TablesLoadedMsg struct {
		Database string
		Tables   []string
		Err      error
	}
	RowsLoadedMsg struct {
		Result database.PageResult
		Err    error
	}
	QueryExecutedMsg struct {
		Result database.QueryResult
		Err    error
	}
	QueryRunRequestedMsg struct {
		Query string
	}
	CellEditDoneMsg struct {
		RowIndex int
		ColIndex int
		Value    string
		Err      error
	}
)

// LoadDatabasesCmd returns a command that loads the list of databases from the
// data source. On completion it sends a DatabasesLoadedMsg.
func LoadDatabasesCmd(source datasource.DataSource) tea.Cmd {
	return func() tea.Msg {
		if source == nil {
			return DatabasesLoadedMsg{Err: database.ErrNoConnection}
		}
		dbs, err := source.Databases(context.Background())
		return DatabasesLoadedMsg{Databases: dbs, Err: err}
	}
}

// LoadTablesCmd returns a command that loads the list of tables for the given
// database. On completion it sends a TablesLoadedMsg.
func LoadTablesCmd(source datasource.DataSource, dbName string) tea.Cmd {
	return func() tea.Msg {
		if source == nil {
			return TablesLoadedMsg{Database: dbName, Err: database.ErrNoConnection}
		}
		tables, err := source.Tables(context.Background(), dbName)
		return TablesLoadedMsg{Database: dbName, Tables: tables, Err: err}
	}
}

// LoadTableRowsCmd returns a command that loads one page of rows for the given
// target. Use page.After for keyset pagination (empty for first page). On
// completion it sends a RowsLoadedMsg.
func LoadTableRowsCmd(source datasource.DataSource, target database.DatabaseTarget, page database.PageRequest) tea.Cmd {
	return func() tea.Msg {
		if source == nil {
			return RowsLoadedMsg{Err: database.ErrNoConnection}
		}
		if page.Limit <= 0 {
			page.Limit = DefaultPageLimit
		}
		result, err := source.Rows(context.Background(), target, page)
		return RowsLoadedMsg{Result: result, Err: err}
	}
}

// RunQueryCmd returns a command that executes one SQL query and sends a
// QueryExecutedMsg with either result rows/columns or an error.
func RunQueryCmd(source datasource.DataSource, query string) tea.Cmd {
	return func() tea.Msg {
		if source == nil {
			return QueryExecutedMsg{Err: database.ErrNoConnection}
		}
		result, err := source.Query(context.Background(), query)
		return QueryExecutedMsg{Result: result, Err: err}
	}
}

// RequestQueryRunCmd returns a command that schedules query execution on a
// near-future tick. This gives the UI one render cycle to show the
// "Running query..." status before results replace it.
func RequestQueryRunCmd(query string) tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(time.Time) tea.Msg {
		return QueryRunRequestedMsg{Query: query}
	})
}
