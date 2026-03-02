package model

import (
	tea "charm.land/bubbletea/v2"

	"github.com/jxdones/stoat/internal/ui/common"
	"github.com/jxdones/stoat/internal/ui/components/filterbox"
	"github.com/jxdones/stoat/internal/ui/components/querybox"
	"github.com/jxdones/stoat/internal/ui/components/sidebar"
	"github.com/jxdones/stoat/internal/ui/components/statusbar"
	"github.com/jxdones/stoat/internal/ui/components/table"
	"github.com/jxdones/stoat/internal/ui/components/tabs"
	"github.com/jxdones/stoat/internal/ui/viewstate"
)

// FocusedPanel indicates which panel receives key input.
type FocusedPanel int

const (
	FocusNone FocusedPanel = iota
	FocusSidebar
	FocusFilterbox
	FocusTable
	FocusQuerybox
)

type screenState struct {
	width   int
	height  int
	focus   FocusedPanel
	compact bool
}

// Model is the root Bubble Tea model. It holds all component models and
// delegates Init/Update/View to them as appropriate.
type Model struct {
	view screenState

	sidebar   sidebar.Model
	statusbar statusbar.Model
	tabs      tabs.Model
	querybox  querybox.Model
	filterbox filterbox.Model
	table     table.Model
}

// New returns a new root model with default component state.
func New() Model {
	m := Model{
		sidebar:   sidebar.New(nil, nil),
		statusbar: statusbar.New(),
		filterbox: filterbox.New(),
		tabs:      tabs.New([]string{"Records", "Columns", "Constraints", "Foreign Keys", "Indexes"}),
		table:     table.New(nil, nil),
		querybox:  querybox.New(),
		view: screenState{
			width:  80,
			height: 24,
			focus:  FocusSidebar,
		},
	}
	m.applyViewState()
	return m
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// applyViewState applies the view state to the model.
func (m *Model) applyViewState() {
	frame := computeLayout(m.view.width, m.view.height)

	m.view.compact = m.view.width < 42 || m.view.height < 14
	if m.view.compact || frame.rows.mainContent <= 0 {
		return
	}

	m.sidebar.ApplyViewState(viewstate.ViewState{
		Width:   frame.columns.leftPane,
		Height:  frame.rows.mainContent,
		Focused: m.isFocused(FocusSidebar),
	})

	m.tabs.SetSize(frame.columns.mainPane)
	m.tabs.SetFocused(m.isFocused(FocusTable))

	m.querybox.SetSize(frame.columns.mainPane, frame.main.query)
	if m.isFocused(FocusQuerybox) {
		m.querybox.Focus()
	} else {
		m.querybox.Blur()
	}

	if m.isFocused(FocusFilterbox) {
		m.filterbox.Focus()
	} else {
		m.filterbox.Blur()
	}

	m.table.SetSize(
		common.BoxInnerWidth(frame.columns.mainPane),
		common.PaneInnerHeight(frame.main.table),
	)
}

// isFocused returns true if the given panel is focused.
func (m Model) isFocused(p FocusedPanel) bool {
	return m.view.focus == p
}

// Update implements tea.Model. It handles window resize and delegates key
// messages to the focused component. Sidebar events (e.g. EventOpenRequested)
// are handled here; you can extend to run commands (load tables, run query).
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.view.width = msg.Width
		m.view.height = msg.Height
		m.applyViewState()
		return m, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			m.focusNext()
			m.applyViewState()
			return m, nil
		case "shift+tab":
			m.focusPrevious()
			m.applyViewState()
			return m, nil
		default:
			return m.updateFocused(msg)
		}
	}
	return m, nil
}

// updateFocused forwards the message to the focused panel and returns the updated model and any command.
func (m Model) updateFocused(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.view.focus {
	case FocusSidebar:
		next, ev := m.sidebar.Update(msg)
		m.sidebar = next
		switch ev {
		case sidebar.EventOpenRequested:
			if !m.sidebar.InTablesSection() {
				m.sidebar.OpenSelectedDatabase()
				m.applyViewState()
				return m, nil
			}
			m.view.focus = FocusTable
			m.applyViewState()
			return m, nil
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

// focusNext focuses the next panel in the order:
// Sidebar -> Filterbox -> Table -> Querybox -> Sidebar.
func (m *Model) focusNext() {
	switch m.view.focus {
	case FocusSidebar:
		m.view.focus = FocusFilterbox
	case FocusFilterbox:
		m.view.focus = FocusTable
	case FocusTable:
		m.view.focus = FocusQuerybox
	case FocusQuerybox:
		m.view.focus = FocusSidebar
	default:
		m.view.focus = FocusSidebar
	}
}

// focusPrevious focuses the previous panel in the order:
// Sidebar -> Querybox -> Table -> Filterbox -> Sidebar.
func (m *Model) focusPrevious() {
	switch m.view.focus {
	case FocusSidebar:
		m.view.focus = FocusQuerybox
	case FocusQuerybox:
		m.view.focus = FocusTable
	case FocusTable:
		m.view.focus = FocusFilterbox
	case FocusFilterbox:
		m.view.focus = FocusSidebar
	default:
		m.view.focus = FocusSidebar
	}
}
