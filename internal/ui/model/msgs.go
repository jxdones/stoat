package model

import (
	"context"
	"os"
	"os/exec"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/atotto/clipboard"
	"github.com/jxdones/stoat/internal/database"
	"github.com/jxdones/stoat/internal/ui/datasource"
)

// DefaultPageLimit is the number of rows loaded per page for table data.
const DefaultPageLimit = 200

// Messages returned by async load commands. Handle these in Model.Update to update the model.
// DatabasesLoadedMsg is sent when a list of databases is loaded.
type DatabasesLoadedMsg struct {
	Databases []string
	Err       error
}

// TablesLoadedMsg is sent when a list of tables is loaded.
type TablesLoadedMsg struct {
	Database string
	Tables   []string
	Err      error
}

// RowsLoadedMsg is sent when a page of rows is loaded.
type RowsLoadedMsg struct {
	Result database.PageResult
	Err    error
}

// QueryExecutedMsg is sent when a query execution completes.
type QueryExecutedMsg struct {
	Result database.QueryResult
	Err    error
	Query  string
}

// QueryRunRequestedMsg is sent when the user requests to run a query.
type QueryRunRequestedMsg struct {
	Query string
}

// EditorQueryMsg is sent when the user closes the editor after editing a query.
type EditorQueryMsg struct {
	Query string
	Err   error
}

// TableConstraintsLoadedMsg is sent when constraints for a table have been loaded.
// Used to store primary key columns for safe UPDATE generation.
type TableConstraintsLoadedMsg struct {
	Target      database.DatabaseTarget
	Constraints []database.Constraint
	Err         error
}

// CopyDoneMsg is sent when the copy operation is done.
type CopyDoneMsg struct {
	Err error
}

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

// LoadTableConstraintsCmd returns a command that loads constraints for the given
// target. On completion it sends a TableConstraintsLoadedMsg.
func LoadTableConstraintsCmd(source datasource.DataSource, target database.DatabaseTarget) tea.Cmd {
	return func() tea.Msg {
		if source == nil {
			return TableConstraintsLoadedMsg{Err: database.ErrNoConnection}
		}
		constraints, err := source.Constraints(context.Background(), target)
		return TableConstraintsLoadedMsg{Target: target, Constraints: constraints, Err: err}
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
		return QueryExecutedMsg{Result: result, Err: err, Query: query}
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

// OpenEditorWithQueryCmd returns a command that opens the editor with the given query.
func OpenEditorWithQueryCmd(query string) tea.Cmd {
	f, err := os.CreateTemp("", "stoat-editor-*.sql")
	if err != nil {
		return func() tea.Msg {
			return EditorQueryMsg{Err: err}
		}
	}

	_, err = f.WriteString(query)
	if err != nil {
		f.Close()
		_ = os.Remove(f.Name())
		return func() tea.Msg {
			return EditorQueryMsg{Err: err}
		}
	}

	path := f.Name()
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return func() tea.Msg {
			return EditorQueryMsg{Err: err}
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
			return EditorQueryMsg{Err: execErr}
		}
		if readErr != nil {
			return EditorQueryMsg{Err: readErr}
		}
		return EditorQueryMsg{Query: string(content), Err: nil}
	})
}

// CopyToClipboardCmd returns a command that copies the given value to the clipboard.
func CopyToClipboardCmd(value string) tea.Cmd {
	return func() tea.Msg {
		err := clipboard.WriteAll(value)
		return CopyDoneMsg{Err: err}
	}
}
