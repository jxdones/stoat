package model

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/jxdones/stoat/internal/database"
	"github.com/jxdones/stoat/internal/ui/components/statusbar"
	"github.com/jxdones/stoat/internal/ui/datasource"
)

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

// handleQueryKey handles ctrl+s when the querybox is focused: submits the query for execution.
func (m Model) handleQueryKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
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

	if m.readOnly && m.isWriteQuery(query) {
		cmd := m.statusbar.SetStatusWithTTL(" Read-only mode: write queries are not allowed", statusbar.Warning, 3*time.Second)
		m.querybox.SetValue("")
		return m, cmd, true
	}

	spinnerCmd := m.statusbar.StartSpinner("Running query", statusbar.Info)
	return m, tea.Batch(spinnerCmd, RequestQueryRunCmd(query)), true
}

// onQueryExecuted handles the QueryExecutedMsg and updates the status bar.
// Queries that return a result set (SELECT, or INSERT/UPDATE/DELETE ... RETURNING)
// replace the table with that result. Plain DML with no result set only shows
// the affected row count in the status bar and leaves the table as-is.
func (m Model) onQueryExecuted(msg QueryExecutedMsg) (tea.Model, tea.Cmd) {
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

// onEditorQueryDone handles the EditorQueryMsg after the user closes the editor.
// If the query is non-empty, it is run via the same path as the query box.
func (m Model) onEditorQueryDone(msg EditorQueryMsg) (tea.Model, tea.Cmd) {
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

// openEditor opens $EDITOR with a SQL comment template. When the user
// saves and closes, onEditorQueryDone runs whatever was written.
func (m Model) openEditor() (tea.Model, tea.Cmd) {
	if !m.HasConnection() {
		cmd := m.statusbar.SetStatusWithTTL(" No active connection", statusbar.Warning, 2*time.Second)
		return m, cmd
	}
	template := "-- Write your SQL here, then save and close the editor to run it.\n\n"
	return m, OpenEditorWithQueryCmd(template)
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
