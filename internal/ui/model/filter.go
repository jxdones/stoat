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

// handleFilterKey handles the apply filter shortcut key press.
func (m Model) handleFilterKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
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
		if m.viewingQueryResult {
			m.table.SetRows(m.unfilteredRows)
			cmd := m.statusbar.SetStatusWithTTL(
				fmt.Sprintf(" Query ok: %d row(s) returned", len(m.unfilteredRows)),
				statusbar.Success,
				3*time.Second,
			)
			return m, cmd, true
		}
		m.resetPaging()
		m.setPendingPageNav(pageNavNone)
		m.paging.requestAfter = ""
		target := database.DatabaseTarget{Database: db, Table: tableName}
		page := database.PageRequest{Limit: DefaultPageLimit, After: ""}
		spinnerCmd := m.statusbar.StartSpinner("Loading "+tableName, statusbar.Info)
		return m, tea.Batch(spinnerCmd, LoadTableRowsCmd(m.source, target, page, m.connectionSeq)), true
	}

	if strings.Contains(expression, "=") {
		if _, _, _, ok := parseColumnFilterExpression(expression); !ok {
			cmd := m.statusbar.SetStatusWithTTL(" Invalid filter: missing value after '='", statusbar.Warning, 2*time.Second)
			return m, cmd, true
		}
	}

	filtered := filterRowsByExpression(m.unfilteredRows, m.table.Columns(), expression)
	m.table.SetRows(filtered)
	cmd := m.statusbar.SetStatusWithTTL(
		fmt.Sprintf(" Filter: %d row(s) match", len(filtered)),
		statusbar.Info,
		2*time.Second,
	)
	return m, cmd, true
}

// filterRowsByExpression filters the rows by the expression.
func filterRowsByExpression(rows []table.Row, columns []table.Column, expr string) []table.Row {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return rows
	}

	filtered := make([]table.Row, 0, len(rows))
	// filtering by column e.g: name = 'John Doe', id = 204
	column, value, quoted, ok := parseColumnFilterExpression(expr)
	if ok {
		colFound := false
		for _, c := range columns {
			if c.Key == column {
				colFound = true
				for _, r := range rows {
					if quoted && strings.Contains(r[c.Key], value) {
						filtered = append(filtered, r)
					} else if strings.EqualFold(r[c.Key], value) {
						filtered = append(filtered, r)
					}
				}
				break
			}
		}
		if colFound {
			return filtered
		}
	}
	// filtering by value e.g: John Doe
	needle := strings.ToLower(expr)
	keys := make([]string, 0, len(columns))
	for _, c := range columns {
		keys = append(keys, c.Key)
	}

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

// parseColumnFilterExpression parses the column filter expression and returns
// the column, value, quoted, and true if the expression is valid.
func parseColumnFilterExpression(expr string) (column, value string, quoted bool, ok bool) {
	column, value, ok = strings.Cut(expr, "=")
	if !ok {
		return "", "", false, false
	}
	column = strings.TrimSpace(column)
	value = strings.TrimSpace(value)
	if column == "" || value == "" {
		return "", "", false, false
	}

	// check if the value is quoted
	if (strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) ||
		(strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`)) {
		value = value[1 : len(value)-1]
		quoted = true
	}
	return column, value, quoted, true
}
