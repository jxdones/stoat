package model

import (
	"io"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/jxdones/stoat/internal/config"
	"github.com/jxdones/stoat/internal/database"
	"github.com/jxdones/stoat/internal/ui/common"
	"github.com/jxdones/stoat/internal/ui/components/connectionpicker"
	"github.com/jxdones/stoat/internal/ui/components/editbox"
	"github.com/jxdones/stoat/internal/ui/components/filterbox"
	"github.com/jxdones/stoat/internal/ui/components/querybox"
	"github.com/jxdones/stoat/internal/ui/components/sidebar"
	"github.com/jxdones/stoat/internal/ui/components/statusbar"
	"github.com/jxdones/stoat/internal/ui/components/table"
	"github.com/jxdones/stoat/internal/ui/components/tabs"
	"github.com/jxdones/stoat/internal/ui/datasource"
	"github.com/jxdones/stoat/internal/ui/theme"
	"github.com/jxdones/stoat/internal/ui/viewstate"
)

// screenState tracks the size and focus of the main screen.
type screenState struct {
	width   int
	height  int
	focus   FocusedPanel
	compact bool
}

type activeModal int

const (
	modalNone activeModal = iota
	modalConnectionPicker
)

type tableSchema struct {
	columns     []database.Column
	constraints []database.Constraint
	foreignKeys []database.ForeignKey
	indexes     []database.Index
}

// Model is the root Bubble Tea model; it composes the sidebar, table, status bar, and other components.
type Model struct {
	// states
	view                 screenState
	activeModal          activeModal
	viewingQueryResult   bool
	helpExpanded         bool
	inlineEditMode       bool
	pendingTableReload   bool
	pendingDeleteConfirm bool

	// data
	tableSchema tableSchema

	// pendingConfig holds a database config that has not yet been connected to.
	// When set, Init fires an async ConnectCmd instead of loading databases directly.
	pendingConfig *database.Config

	source           datasource.DataSource
	sidebar          sidebar.Model
	statusbar        statusbar.Model
	tabs             tabs.Model
	querybox         querybox.Model
	filterbox        filterbox.Model
	table            table.Model
	editbox          editbox.Model
	schemaTable      table.Model // table for displaying the schema of the table
	connectionPicker connectionpicker.Model
	paging           pagingState
	savedQueries     []SavedQuery
	unfilteredRows   []table.Row // rows before any filter is applied; always holds the current page from the DB

	// tablePKColumns are primary key column names for the table last loaded; used to build
	// a safe WHERE clause when generating UPDATE from a cell. tablePKTarget identifies
	// which table they belong to.
	tablePKColumns []string
	tablePKTarget  database.DatabaseTarget

	queryResultPreview string // truncated one-line preview of the last run query for the header
	lastKey            string // last key pressed, used to detect key sequences

	forceReadOnly bool // set by --read-only CLI flag; forces read-only regardless of connection config
	readOnly      bool // true when the active connection is read-only (either flag or config)

	debugOutput io.Writer // for timing debug output
}

// detailRows returns the number of rows the detail section should occupy.
func (m Model) detailRows() int {
	if m.inlineEditMode || m.pendingDeleteConfirm {
		return mainDetailRowsEdit
	}
	return mainDetailRowsNormal
}

// New creates a new root model with default component state.
func New() Model {
	m := Model{
		editbox:   editbox.New(),
		sidebar:   sidebar.New(nil, nil),
		statusbar: statusbar.New(),
		filterbox: filterbox.New(),
		tabs: tabs.NewWithShortLabels(
			[]string{"Records", "Columns", "Constraints", "Foreign Keys", "Indexes"},
			[]string{"Recs", "Cols", "Cons", "FKs", "Idx"},
		),
		table:            table.New(nil, nil),
		querybox:         querybox.New(),
		connectionPicker: connectionpicker.New(),
		view: screenState{
			width:  80,
			height: 24,
			focus:  FocusSidebar,
		},
		paging: pagingState{
			afterStack: []string{""},
			pendingNav: pageNavNone,
		},
		savedQueries: []SavedQuery{},
	}
	m.applyViewState()
	return m
}

// Init starts the connection when a pending config is present, or loads
// databases immediately when a data source is already set (e.g. in tests).
func (m Model) Init() tea.Cmd {
	if m.pendingConfig != nil {
		cfg := *m.pendingConfig
		return func() tea.Msg { return ConnectingMsg{cfg: cfg} }
	}
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

// SetPendingConfig stores a database config to be connected asynchronously
// when the program starts. Init will fire a ConnectCmd for it.
func (m *Model) SetPendingConfig(cfg database.Config) {
	m.pendingConfig = &cfg
}

// HasConnection checks if the model has an active data source.
func (m Model) HasConnection() bool {
	return m.source != nil
}

// Close releases the active data source connection, if any.
func (m Model) Close() {
	if m.source != nil {
		_ = m.source.Close()
	}
}

// applyViewState updates the view state based on the current terminal size.
func (m *Model) applyViewState() {
	var optionsHeight int
	if m.helpExpanded {
		optionsHeight = expandedOptionsHeight(m.view.width, m.fullHelpBindings())
	} else {
		optionsHeight = 2
	}
	frame := computeLayout(m.view.width, m.view.height, optionsHeight, m.detailRows())

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

	m.editbox.SetWidth(common.BoxInnerWidth(frame.columns.mainPane))
	if m.inlineEditMode {
		m.editbox.Focus()
	} else {
		m.editbox.Blur()
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

// SetConfig applies the configuration to the model. It sets the theme and
// populates the connection picker. Saved queries are loaded later in
// handleConnected, once a connection is established.
func (m *Model) SetConfig(config config.Config) {
	if _, ok := theme.SetNamedTheme(config.Theme); ok {
		m.querybox.ApplyTheme()
		m.editbox.ApplyTheme()
		m.applyViewState()
	}
	m.connectionPicker.SetConnections(config.Connections)
}

// SetDebugOutput sets the output writer for timing debug output.
func (m *Model) SetDebugOutput(out io.Writer) {
	m.debugOutput = out
}

func (m *Model) OpenConnectionPicker() {
	m.activeModal = modalConnectionPicker
}

// SetReadOnly sets the read-only flag, shown as [RO] on the right of the status bar.
func (m *Model) SetReadOnly(readOnly bool) {
	m.forceReadOnly = readOnly
	m.statusbar.SetReadOnly(readOnly)
}

// isWriteQuery checks if the query is a write query.
func (m Model) isWriteQuery(query string) bool {
	q := strings.ToUpper(strings.TrimSpace(query))
	for _, kw := range []string{"INSERT", "UPDATE", "DELETE", "DROP", "CREATE", "ALTER", "TRUNCATE"} {
		if strings.HasPrefix(q, kw) {
			return true
		}
	}
	return false
}

// toModelSavedQueries converts a list of config.SavedQuery to a list of Model.SavedQuery.
func toModelSavedQueries(savedQueries []config.SavedQuery) []SavedQuery {
	modelSavedQueries := make([]SavedQuery, len(savedQueries))
	for i, savedQuery := range savedQueries {
		modelSavedQueries[i] = SavedQuery{
			Name:  savedQuery.Name,
			Query: savedQuery.Query,
		}
	}
	return modelSavedQueries
}

// Update handles window resize, async load results,
// and key messages for the focused component.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ConnectingMsg:
		return m.handleConnecting(msg)
	case ConnectedMsg:
		return m.handleConnected(msg)
	case ConnectionFailedMsg:
		return m.handleConnectionFailed(msg)
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
	case EditorQueryMsg:
		return m.handleEditorQueryDone(msg)
	case TableConstraintsLoadedMsg:
		return m.handleTableConstraintsLoaded(msg)
	case IndexesLoadedMsg:
		return m.handleIndexesLoaded(msg)
	case ForeignKeysLoadedMsg:
		return m.handleForeignKeysLoaded(msg)
	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)
	case statusbar.ExpiredMsg:
		m.statusbar.HandleExpired(msg)
		return m, nil
	case tea.KeyPressMsg:
		return m.handleKeyPress(msg)
	case tea.PasteMsg:
		return m.handlePasteMsg(msg)
	case CopyDoneMsg:
		if msg.Err != nil {
			cmd := m.statusbar.SetStatusWithTTL(" Copy failed: "+msg.Err.Error(), statusbar.Error, 2*time.Second)
			return m, cmd
		}
		cmd := m.statusbar.SetStatusWithTTL(" Copied to clipboard", statusbar.Success, 2*time.Second)
		return m, cmd
	default:
		cmd := m.statusbar.Update(msg)
		return m, cmd
	}
}
