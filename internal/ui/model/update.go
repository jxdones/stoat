package model

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/jxdones/stoat/internal/ui/components/statusbar"
)

// keyHandler is a function that handles a key press and returns whether it was handled.
type keyHandler func(tea.KeyPressMsg) (tea.Model, tea.Cmd, bool)

// Update handles window resize, async load results,
// and key messages for the focused component.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ConnectingMsg:
		return m.onConnecting(msg)
	case ConnectedMsg:
		return m.onConnected(msg)
	case ConnectionFailedMsg:
		return m.onConnectionFailed(msg)
	case DatabasesLoadedMsg:
		return m.onDatabasesLoaded(msg)
	case TablesLoadedMsg:
		return m.onTablesLoaded(msg)
	case RowsLoadedMsg:
		return m.onRowsLoaded(msg)
	case QueryExecutedMsg:
		return m.onQueryExecuted(msg)
	case QueryRunRequestedMsg:
		return m, RunQueryCmd(m.source, msg.Query)
	case EditorQueryMsg:
		return m.onEditorQueryDone(msg)
	case TableConstraintsLoadedMsg:
		return m.onTableConstraintsLoaded(msg)
	case IndexesLoadedMsg:
		return m.onIndexesLoaded(msg)
	case ForeignKeysLoadedMsg:
		return m.onForeignKeysLoaded(msg)
	case tea.WindowSizeMsg:
		return m.onWindowSize(msg)
	case statusbar.ExpiredMsg:
		m.statusbar.HandleExpired(msg)
		return m, nil
	case tea.KeyPressMsg:
		return m.handleKeyPress(msg)
	case tea.PasteMsg:
		return m.delegatePaste(msg)
	case CopyDoneMsg:
		return m.onCopyDone(msg)
	case EditorCellMsg:
		return m.onCellEditorDone(msg)
	default:
		cmd := m.statusbar.Update(msg)
		return m, cmd
	}
}

// onWindowSize handles the WindowSizeMsg and updates the view state.
func (m Model) onWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.view.width = msg.Width
	m.view.height = msg.Height
	m.applyViewState()
	return m, nil
}

// handleKeyPress routes key presses to the active mode, modal, global shortcuts,
// or feature-specific key handlers.
func (m Model) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.inlineEditMode {
		return m.handleEditModeKey(msg)
	}
	prevKey := m.lastKey
	m.lastKey = ""

	if m.pendingDeleteConfirm {
		if next, cmd, handled := m.confirmDelete(msg); handled {
			return next, cmd
		}
		if next, cmd, handled := m.cancelDelete(msg); handled {
			return next, cmd
		}
		return m, nil
	}

	if m.activeModal == modalConnectionPicker {
		return m.handleConnectionPickerKey(msg)
	}

	if m.activeModal == modalCellDetail {
		return m.handleCellDetailModalKey(msg)
	}

	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "ctrl+e":
		return m.openEditor()
	case "ctrl+r":
		return m.reload()
	case "q":
		if m.view.focus == FocusNone {
			return m, tea.Quit
		}
		return m.delegateToFocused(msg)
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
		return m.delegateToFocused(msg)
	default:
		handlers := []keyHandler{
			m.handleFilterKey,
			m.handleSchemaTabKey,
			m.handleEditKey,
			func(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
				return m.handleDeleteKey(msg, prevKey)
			},
			m.handleCellEditorKey,
			m.handleQueryKey,
			m.handlePagingKey,
			m.handleSavedQueryKey,
			m.handleCopyKey,
			m.handleCellDetailKey,
		}
		for _, h := range handlers {
			if next, cmd, handled := h(msg); handled {
				return next, cmd
			}
		}
		return m.delegateToFocused(msg)
	}
}

// onCopyDone handles the result of a clipboard copy operation.
func (m Model) onCopyDone(msg CopyDoneMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		cmd := m.statusbar.SetStatusWithTTL(" Copy failed: "+msg.Err.Error(), statusbar.Error, 2*time.Second)
		return m, cmd
	}
	cmd := m.statusbar.SetStatusWithTTL(" Copied to clipboard", statusbar.Success, 2*time.Second)
	return m, cmd
}
