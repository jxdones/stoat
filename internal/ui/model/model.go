package model

import (
	"io"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/jxdones/stoat/internal/config"
	"github.com/jxdones/stoat/internal/database"
	"github.com/jxdones/stoat/internal/ui/components/celldetail"
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
)

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
	cellDetail       celldetail.Model
	paging           pagingState
	savedQueries     []SavedQuery
	unfilteredRows   []table.Row    // rows before any filter is applied; always holds the current page from the DB
	fkViewport       viewport.Model // viewport for displaying data from the Foreign Keys tab

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
		cellDetail:       celldetail.New(),
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
		m.sidebar.SetDatabaseLabel(m.source.DatabaseLabel())
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

// isFocused checks if the given panel is focused.
func (m Model) isFocused(p FocusedPanel) bool {
	return m.view.focus == p
}

// SetConfig applies the configuration to the model. It sets the theme and
// populates the connection picker. Saved queries are loaded later in
// onConnected, once a connection is established.
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

// OpenConnectionPicker opens the connection picker modal.
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
