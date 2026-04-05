package model

import (
	tea "charm.land/bubbletea/v2"

	"github.com/jxdones/stoat/internal/database"
	"github.com/jxdones/stoat/internal/ui/components/statusbar"
)

// pagingState tracks cursor-based page navigation for the active table.
// Cursors are opaque strings owned by the DB adapter (e.g. rowid:200).
type pagingState struct {
	// currentAfter is the cursor used to request the next page.
	currentAfter string
	// currentHasMore indicates if another next page exists.
	currentHasMore bool
	// afterStack stores visited page-start cursors for backward navigation.
	// Convention: first entry is the initial empty cursor.
	afterStack []string
	// pendingNav records which navigation intent produced the in-flight request.
	pendingNav pageNav
	// requestAfter is the cursor we sent for the in-flight request; used when applying the result.
	requestAfter string
}

// setPendingPageNav tracks which navigation intent is tied to the in-flight page load.
// Call this right before dispatching a rows-page request.
func (m *Model) setPendingPageNav(nav pageNav) {
	m.paging.pendingNav = nav
}

// applyPageResult updates pagination state after a rows-page response.
// History transition rules:
// - next: push startAfter if it is not already the top entry
// - prev: pop one history entry (if possible)
// - none: reset history to only the returned startAfter
func (m *Model) applyPageResult(startAfter, nextAfter string, hasMore bool) {
	switch m.paging.pendingNav {
	case pageNavNext:
		if len(m.paging.afterStack) == 0 {
			m.paging.afterStack = []string{""}
		}
		if m.paging.afterStack[len(m.paging.afterStack)-1] != startAfter {
			m.paging.afterStack = append(m.paging.afterStack, startAfter)
		}
	case pageNavPrev:
		if len(m.paging.afterStack) > 1 {
			m.paging.afterStack = m.paging.afterStack[:len(m.paging.afterStack)-1]
		}
	default:
		m.paging.afterStack = []string{startAfter}
	}

	m.paging.currentAfter = nextAfter
	m.paging.currentHasMore = hasMore
	m.paging.pendingNav = pageNavNone
}

// resetPaging clears all paging state to the initial cursor.
// Use on connection switch, table switch, or query result replacement.
func (m *Model) resetPaging() {
	m.paging.currentAfter = ""
	m.paging.currentHasMore = false
	m.paging.afterStack = []string{""}
	m.paging.pendingNav = pageNavNone
}

// handlePagingKey handles ctrl+n (next page) and ctrl+b (previous page) when the table is focused.
func (m Model) handlePagingKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
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
		}, m.connectionSeq)), true
	}
	if msg.String() == "ctrl+b" && len(m.paging.afterStack) > 1 {
		prevCursor := m.paging.afterStack[len(m.paging.afterStack)-2]
		m.setPendingPageNav(pageNavPrev)
		m.paging.requestAfter = prevCursor
		spinnerCmd := m.statusbar.StartSpinner("Loading page", statusbar.Info)
		return m, tea.Batch(spinnerCmd, LoadTableRowsCmd(m.source, target, database.PageRequest{
			Limit: DefaultPageLimit,
			After: prevCursor,
		}, m.connectionSeq)), true
	}
	return m, nil, false
}
