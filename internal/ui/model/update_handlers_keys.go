package model

import (
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/jxdones/stoat/internal/database"
	"github.com/jxdones/stoat/internal/ui/components/celldetail"
	"github.com/jxdones/stoat/internal/ui/components/statusbar"
	"github.com/jxdones/stoat/internal/ui/components/table"
)

// keyHandler is a function that handles a key press and returns whether it was handled.
type keyHandler func(tea.KeyPressMsg) (tea.Model, tea.Cmd, bool)

// handleKeyPress handles the KeyPressMsg and updates the focused component.
func (m Model) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.inlineEditMode {
		return m.handleKeyPressInEditMode(msg)
	}
	prevKey := m.lastKey
	m.lastKey = ""

	if m.pendingDeleteConfirm {
		if next, cmd, handled := m.handleDeleteConfirm(msg); handled {
			return next, cmd
		}
		if next, cmd, handled := m.handleDeleteCancel(msg); handled {
			return next, cmd
		}
		return m, nil
	}

	if m.activeModal == modalConnectionPicker {
		return m.handleKeyPressInConnectionPicker(msg)
	}

	if m.activeModal == modalCellDetail {
		return m.handleKeyPressInCellDetail(msg)
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
			func(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
				return m.handleDeleteRow(msg, prevKey)
			},
			m.handleOpenCellEditor,
			m.handleQueryShortcut,
			m.handlePagingShortcut,
			m.handleExpandSavedQuery,
			m.handleCopyCellValueFromTable,
			m.handleViewCellDetail,
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

	if m.readOnly && m.isWriteQuery(query) {
		cmd := m.statusbar.SetStatusWithTTL(" Read-only mode: write queries are not allowed", statusbar.Warning, 3*time.Second)
		m.querybox.SetValue("")
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

// handleCopyCellValueFromTable handles the copy of active cell value from table shortcut key press.
func (m Model) handleCopyCellValueFromTable(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	if msg.String() != "y" || m.view.focus != FocusTable {
		return m, nil, false
	}

	_, cellValue, ok := m.table.ActiveCell()
	if !ok {
		return m, nil, false
	}

	if cellValue == table.NullValue {
		cellValue = ""
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

// handleViewCellDetail handles the view cell detail shortcut key press.
func (m Model) handleViewCellDetail(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	if msg.String() != "v" || m.view.focus != FocusTable {
		return m, nil, false
	}
	column, value, ok := m.table.ActiveCell()
	if !ok {
		return m, nil, false
	}
	cd := m.cellDetail.SetContent(column.Key, column.Type, value)
	m.cellDetail = cd
	m.activeModal = modalCellDetail
	return m, nil, true
}

// handleKeyPressInCellDetail handles the key press in the cell detail modal.
func (m Model) handleKeyPressInCellDetail(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	next, event := m.cellDetail.Update(msg)
	m.cellDetail = next

	switch event {
	case celldetail.EventClosed:
		m.activeModal = modalNone
	case celldetail.EventNone:
		return m, nil
	}
	return m, nil
}
