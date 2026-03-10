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
	"github.com/jxdones/stoat/internal/ui/datasource"
	"github.com/jxdones/stoat/internal/ui/viewstate"
)

// screenState tracks the size and focus of the main screen.
type screenState struct {
	width   int
	height  int
	focus   FocusedPanel
	compact bool
}

// Model is the root Bubble Tea model; it composes the sidebar, table, status bar, and other components.
type Model struct {
	view screenState

	source    datasource.DataSource
	sidebar   sidebar.Model
	statusbar statusbar.Model
	tabs      tabs.Model
	querybox  querybox.Model
	filterbox filterbox.Model
	table     table.Model
	paging    pagingState
}

// New creates a new root model with default component state.
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
		paging: pagingState{
			afterStack: []string{""},
			pendingNav: pageNavNone,
		},
	}
	m.applyViewState()
	return m
}

// Init loads databases when a data source is set.
func (m Model) Init() tea.Cmd {
	if m.HasConnection() {
		return LoadDatabasesCmd(m.source)
	}
	return nil
}

// SetDataSource sets the data source.
// Pass nil when disconnected. The model uses it to load tables, rows, etc.
func (m *Model) SetDataSource(source datasource.DataSource) {
	m.source = source
}

// HasConnection checks if the model has an active data source.
func (m Model) HasConnection() bool {
	return m.source != nil
}

// applyViewState updates the view state based on the current terminal size.
func (m *Model) applyViewState() {
	frame := computeLayout(m.view.width, m.view.height)

	m.view.compact = m.view.width < 80 || m.view.height < 24
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

// isFocused checks if the given panel is focused.
func (m Model) isFocused(p FocusedPanel) bool {
	return m.view.focus == p
}

// Update handles window resize, async load results,
// and key messages for the focused component.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case DatabasesLoadedMsg:
		return m.handleDatabasesLoaded(msg)
	case TablesLoadedMsg:
		return m.handleTablesLoaded(msg)
	case RowsLoadedMsg:
		return m.handleRowsLoaded(msg)
	case QueryExecutedMsg:
		return m.handleQueryExecuted(msg)
	case QueryRunRequestedMsg:
		return m, RunQueryCmd(m.source, msg.Query)
	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)
	case statusbar.ExpiredMsg:
		m.statusbar.HandleExpired(msg)
		return m, nil
	case tea.KeyPressMsg:
		return m.handleKeyPress(msg)
	case tea.PasteMsg:
		return m.handlePasteMsg(msg)
	}
	return m, nil
}
