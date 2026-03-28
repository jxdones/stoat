package sidebar

import (
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/jxdones/stoat/internal/ui/common"
	"github.com/jxdones/stoat/internal/ui/keys"
	"github.com/jxdones/stoat/internal/ui/theme"
	"github.com/jxdones/stoat/internal/ui/viewstate"
)

const (
	sidebarSectionHeaderRows = 2 // [ Databases ] + [ Tables ]
	sidebarSectionGapRows    = 1 // visual padding between the two sections
	sidebarOverflowMarker    = "…"
	sidebarTruncateSuffix    = "…"
	sidebarMinWidth          = 12
	sidebarMinHeight         = 8
	sidebarDBRowsDivisor     = 2 // databases viewport is capped to half list rows
	maxCountDigits           = 3 // maximum number of digits in the count buffer
)

// Event represents an action or state change in the sidebar.
//
// The sidebar uses an event-return design instead of returning tea.Cmd: Update
// returns (Model, Event). The root model should handle events as follows:
//
//   - EventOpenRequested: User pressed Enter on a database or table. Call
//     OpenSelectedDatabase() when the target is a DB (and optionally load
//     tables, then open). For tables, the root might focus the table or run a
//     query. The sidebar does not mutate on Enter; the root decides when to open.
//   - EventSelectionChanged: Selection moved (e.g. j/k). Use if you need to
//     sync other UI or load preview data.
//   - EventSectionChanged: User switched between Databases and Tables section
//     (e.g. via a key the root maps to SwitchSection). Rare; only react if needed.
//
// Returning events keeps the sidebar decoupled from I/O and commands: it never
// sends tea.Cmd, so the root owns all side effects (loading tables, navigation).
type Event int

const (
	EventNone Event = iota
	EventSelectionChanged
	EventSectionChanged
	EventOpenRequested
)

// Model represents a sidebar with databases and tables.
type Model struct {
	databases []string
	tables    map[string][]string

	loadingDBs    bool
	loadingTables bool

	selectedDB    int
	selectedTable int
	dbOffset      int
	tableOffset   int
	openedDB      int
	section       int // 0 for databases, 1 for tables

	width         int
	height        int
	focused       bool
	databaseLabel string
	countBuffer   string
}

// New creates a new sidebar model with the given databases and tables.
func New(databases []string, tables map[string][]string) Model {
	m := Model{
		databases:     databases,
		tables:        tables,
		openedDB:      -1,
		width:         20,
		height:        10,
		databaseLabel: "Databases",
	}
	return m
}

// SetSize sets the size of the sidebar and clamps it to the minimum width and height.
func (m *Model) SetSize(width, height int) {
	m.width = common.ClampMin(width, sidebarMinWidth)
	m.height = common.ClampMin(height, sidebarMinHeight)
	m.clamp()
}

// SetFocused sets the focused state of the sidebar.
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
}

// SetDatabases sets the databases.
func (m *Model) SetDatabases(databases []string) {
	dbs := make([]string, 0, len(databases))
	for _, db := range databases {
		if strings.TrimSpace(db) != "" {
			dbs = append(dbs, db)
		}
	}
	m.databases = dbs
	if m.tables == nil {
		m.tables = make(map[string][]string)
	}
	m.clamp()
}

// SetTables sets the tables for a given database.
func (m *Model) SetTables(db string, tables []string) {
	if m.tables == nil {
		m.tables = make(map[string][]string)
	}
	tbls := make([]string, 0, len(tables))
	for _, table := range tables {
		if strings.TrimSpace(table) != "" {
			tbls = append(tbls, table)
		}
	}
	m.tables[db] = tbls
	m.clamp()
}

// SetLoadingDatabases sets the loading state of the databases.
func (m *Model) SetLoadingDatabases(loading bool) {
	m.loadingDBs = loading
	if loading {
		m.openedDB = -1
		m.selectedTable = 0
		m.section = 0
	}
	m.clamp()
}

// SetLoadingTables sets the loading state of the tables.
func (m *Model) SetLoadingTables(loading bool) {
	m.loadingTables = loading
	m.clamp()
}

// InTablesSection returns true if the sidebar is in the tables section.
func (m Model) InTablesSection() bool {
	return m.section == 1
}

// ApplyViewState applies the view state to the sidebar.
func (m *Model) ApplyViewState(viewState viewstate.ViewState) {
	m.SetSize(viewState.Width, viewState.Height)
	m.SetFocused(viewState.Focused)
}

// Update handles key messages and updates the model state.
func (m Model) Update(msg tea.Msg) (Model, Event) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, EventNone
	}

	beforeSection := m.section
	beforeDB := m.SelectedDB()
	beforeTable := m.SelectedTable()

	if isDigitKey(k) {
		if len(m.countBuffer) < maxCountDigits {
			m.countBuffer += k.String()
		}
		return m, EventNone
	}

	switch {
	case key.Matches(k, keys.Default.MoveUp):
		n := parseBufferCount(m.countBuffer)
		m.countBuffer = ""
		m.Move(-n)
	case key.Matches(k, keys.Default.MoveDown):
		n := parseBufferCount(m.countBuffer)
		m.countBuffer = ""
		m.Move(n)
	case key.Matches(k, keys.Default.GotoTop):
		m.MoveToTop()
	case key.Matches(k, keys.Default.GotoBottom):
		m.MoveToBottom()
	case key.Matches(k, keys.Default.MoveLeft):
		m.SwitchSection(-1)
	case key.Matches(k, keys.Default.MoveRight):
		m.SwitchSection(1)
	case key.Matches(k, keys.Default.Enter):
		return m, EventOpenRequested
	default:
		return m, EventNone
	}

	if m.section != beforeSection {
		return m, EventSectionChanged
	}
	if m.SelectedDB() != beforeDB || m.SelectedTable() != beforeTable {
		return m, EventSelectionChanged
	}

	return m, EventNone
}

// View renders the sidebar with the current state.
func (m Model) View() tea.View {
	width := common.ClampMin(m.width, sidebarMinWidth)
	height := common.ClampMin(m.height, sidebarMinHeight)
	contentWidth := common.BoxContentWidth(width)
	innerHeight := common.PaneInnerHeight(height)
	dbRows, tableRows := m.visibleRows()
	if innerHeight <= 0 {
		content := common.BorderedPane(width, height, m.focused, common.FocusBorder(m.focused)).Render("")
		return tea.NewView(content)
	}

	lines := make([]string, 0, innerHeight)
	lines = append(lines, sectionTile(fit("[ "+m.databaseLabel+" ]", contentWidth), m.focused && m.section == 0))
	lines = append(lines, viewportLines(m.databaseLines(contentWidth), m.dbOffset, dbRows, contentWidth)...)
	lines = append(lines, "")

	lines = append(lines, sectionTile(fit("[ Tables ]", contentWidth), m.focused && m.section == 1))
	lines = append(lines, viewportLines(m.tableLines(contentWidth), m.tableOffset, tableRows, contentWidth)...)
	for len(lines) < innerHeight {
		lines = append(lines, "")
	}
	if len(lines) > innerHeight {
		lines = lines[:innerHeight]
	}
	content := common.BorderedPane(width, height, m.focused, common.FocusBorder(m.focused)).Render(strings.Join(lines, "\n"))
	return tea.NewView(content)
}

// HelpBindings returns the key bindings for the sidebar.
func HelpBindings() []key.Binding {
	return []key.Binding{
		key.NewBinding(
			key.WithKeys("up", "down", "j", "k"),
			key.WithHelp("j/k", "navigate"),
		),
		key.NewBinding(
			key.WithKeys("g", "home", "G", "end"),
			key.WithHelp("g/G", "jump top/bottom"),
		),
		key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "open/focus table"),
		),
	}
}

// viewportLines returns a slice of lines for the current viewport, with "…" markers
// pinned to the absolute first/last row when content overflows in that direction.
// keepOffsetVisible ensures the selected item is never on the edge row when a marker
// is present, so the selected item is always adjacent to — never behind — the marker.
func viewportLines(lines []string, start, rows, contentWidth int) []string {
	window := offsetWindowLines(lines, start, rows)
	if rows <= 0 || len(window) == 0 || len(lines) <= rows {
		return window
	}

	hasAbove := start > 0
	hasBelow := start+rows < len(lines)
	marker := centeredOverflowMarker(contentWidth)

	if hasAbove {
		window[0] = marker
	}
	if hasBelow {
		window[rows-1] = marker
	}
	return window
}

// centeredOverflowMarker returns a centered "…" marker for overflow.
func centeredOverflowMarker(contentWidth int) string {
	return lipgloss.NewStyle().
		Foreground(theme.Current.TextMuted).
		Width(contentWidth).
		Align(lipgloss.Center).
		Render(sidebarOverflowMarker)
}

// offsetWindowLines returns a copy of the lines visible in the current viewport.
// A copy is returned so that callers can overwrite entries (e.g. overflow markers)
// without mutating the original slice.
func offsetWindowLines(lines []string, start, maxRows int) []string {
	if maxRows <= 0 || len(lines) == 0 {
		return nil
	}
	if len(lines) <= maxRows {
		out := make([]string, len(lines))
		copy(out, lines)
		return out
	}
	if start < 0 {
		start = 0
	}
	lastStart := len(lines) - maxRows
	if start > lastStart {
		start = lastStart
	}
	out := make([]string, maxRows)
	copy(out, lines[start:start+maxRows])
	return out
}

// databaseLines returns the lines for the databases section.
func (m Model) databaseLines(contentWidth int) []string {
	if m.loadingDBs {
		return []string{
			lipgloss.NewStyle().Foreground(theme.Current.TextWarning).
				Render(fit("(loading databases...)", contentWidth)),
		}
	}
	if len(m.databases) == 0 {
		return []string{
			lipgloss.NewStyle().Foreground(theme.Current.TextMuted).
				Render(fit("(none)", contentWidth)),
		}
	}
	out := make([]string, 0, len(m.databases))
	for i, db := range m.databases {
		style := lipgloss.NewStyle().Foreground(theme.Current.TextPrimary)
		if i == m.selectedDB {
			style = lipgloss.NewStyle().Foreground(theme.Current.SidebarSelectedFg).Background(theme.Current.SidebarSelectedBg).Bold(true)
		}
		label := "  " + db
		out = append(out, style.Render(fit(label, contentWidth)))
	}
	return out
}

// tableLines returns the lines for the tables section.
func (m Model) tableLines(contentWidth int) []string {
	tables := m.currentTables()

	out := make([]string, 0, len(tables))
	for i, table := range tables {
		style := lipgloss.NewStyle().Foreground(theme.Current.TabsPrefix)
		if table == "(none)" {
			style = lipgloss.NewStyle().Foreground(theme.Current.TextMuted)
		}
		if i == m.selectedTable && m.openedDB >= 0 {
			style = lipgloss.NewStyle().Foreground(theme.Current.SidebarSelectedFg).Background(theme.Current.SidebarSelectedBg).Bold(true)
		}

		label := "  " + table
		if strings.HasPrefix(table, "(") && strings.HasSuffix(table, ")") {
			label = table
		}
		out = append(out, style.Render(fit(label, contentWidth)))

	}
	return out
}

// clamp ensures the selection and offset are within valid bounds.
func (m *Model) clamp() {
	if len(m.databases) == 0 {
		m.selectedDB = -1
		m.openedDB = -1
		m.selectedTable = 0
		m.dbOffset = 0
		m.tableOffset = 0
		m.section = 0
		return
	}
	if m.selectedDB < 0 {
		m.selectedDB = 0
	}
	if m.selectedDB >= len(m.databases) {
		m.selectedDB = len(m.databases) - 1
	}
	if m.openedDB >= len(m.databases) {
		m.openedDB = -1
	}
	if m.section == 1 && m.openedDB < 0 {
		m.section = 0
	}

	tables := m.currentTables()
	hasTables := len(tables) > 0
	if !hasTables {
		m.selectedTable = 0
		m.tableOffset = 0
	} else {
		if m.selectedTable < 0 {
			m.selectedTable = 0
		}
		if m.selectedTable >= len(tables) {
			m.selectedTable = len(tables) - 1
		}
	}

	dbRows, tableRows := m.visibleRows()
	m.dbOffset = keepOffsetVisible(m.selectedDB, dbRows, m.dbOffset, m.databaseLineCount())
	if hasTables {
		m.tableOffset = keepOffsetVisible(m.selectedTable, tableRows, m.tableOffset, m.tableLineCount())
	}
}

// databaseLineCount returns the number of lines needed to display the databases.
func (m Model) databaseLineCount() int {
	if m.loadingDBs || len(m.databases) == 0 {
		return 1
	}
	return len(m.databases)
}

// tableLineCount returns the number of lines needed to display the tables.
func (m Model) tableLineCount() int {
	tables := m.currentTables()
	if len(tables) == 0 {
		return 1
	}
	return len(tables)
}

// currentTables returns the tables for the currently opened database.
func (m Model) currentTables() []string {
	if m.loadingTables {
		return []string{"Loading tables..."}
	}
	if len(m.databases) == 0 {
		return []string{"(none)"}
	}
	if m.openedDB < 0 || m.openedDB >= len(m.databases) {
		return []string{"(press Enter on a database)"}
	}
	db := m.databases[m.openedDB]
	tables := m.tables[db]
	if len(tables) == 0 {
		return []string{"(none)"}
	}
	return tables
}

// visibleRows returns how many rows the databases and tables sections can show.
// It splits the available list area between DBs (capped at half) and tables.
func (m Model) visibleRows() (dbRows, tableRows int) {
	innerHeight := common.PaneInnerHeight(common.ClampMin(m.height, sidebarMinHeight))
	listRows := innerHeight - sidebarSectionHeaderRows - sidebarSectionGapRows
	if listRows < 0 {
		return 0, 0
	}
	dbMaxRows := listRows / sidebarDBRowsDivisor
	dbLineCount := m.databaseLineCount()
	dbRows = min(dbMaxRows, dbLineCount)
	tableRows = max(0, listRows-dbRows)
	return dbRows, tableRows
}

// SelectedDB returns the currently selected database.
func (m Model) SelectedDB() string {
	if m.selectedDB < 0 || m.selectedDB >= len(m.databases) {
		return ""
	}
	return m.databases[m.selectedDB]
}

// SelectedTable returns the currently selected table.
func (m Model) SelectedTable() string {
	if m.openedDB < 0 {
		return ""
	}
	tables := m.currentTables()
	if m.selectedTable < 0 || m.selectedTable >= len(tables) {
		return ""
	}
	return tables[m.selectedTable]
}

// ActiveDB returns the currently active/opened database.
func (m Model) ActiveDB() string {
	if m.openedDB < 0 || m.openedDB >= len(m.databases) {
		return ""
	}
	return m.databases[m.openedDB]
}

// EffectiveDB returns the database to use for loading (ActiveDB if set, else SelectedDB).
func (m Model) EffectiveDB() string {
	if db := m.ActiveDB(); db != "" {
		return db
	}
	return m.SelectedDB()
}

// ActiveSection returns the currently active section.
func (m Model) ActiveSection() string {
	if m.section == 0 {
		return "databases"
	}
	return "tables"
}

// keepOffsetVisible returns an offset so the selected index stays visible in a
// viewport of maxRows. Used to scroll the list when selection moves off-screen.
func keepOffsetVisible(selected, maxRows, offset, total int) int {
	if total <= 0 || maxRows <= 0 {
		return 0
	}
	selected = max(0, min(selected, total-1))
	offset = max(0, offset)
	offset = min(selected, offset)
	if selected >= offset+maxRows {
		offset = selected - maxRows + 1
	}
	maxOffset := max(0, total-maxRows)
	offset = min(offset, maxOffset)

	// When overflow markers are present, ensure the selected item never lands on
	// the edge row that the marker occupies. Only applies when the viewport is
	// large enough to have at least one non-marker row visible on each side.
	if maxRows > 2 {
		if offset+maxRows < total && selected == offset+maxRows-1 {
			// Items below: selected at last row — scroll down one so marker fits there.
			offset = min(offset+1, maxOffset)
		}
		if offset > 0 && selected == offset {
			// Items above: selected at first row — scroll up one so marker fits there.
			offset = max(0, offset-1)
		}
	}
	return offset
}

// sectionTile renders a section tile with the given title and active state.
func sectionTile(title string, active bool) string {
	style := lipgloss.NewStyle().Foreground(theme.Current.SidebarTitle).Bold(true)
	if active {
		style = style.Foreground(theme.Current.SidebarTitleHot)
	}
	return style.Render(title)
}

// fit truncates text to width runes and appends "…" when shortened.
func fit(text string, width int) string {
	r := []rune(text)
	if len(r) <= width {
		return text
	}
	suffix := []rune(sidebarTruncateSuffix)
	if width <= len(suffix) {
		return string(r[:width])
	}
	return string(r[:width-len(suffix)]) + sidebarTruncateSuffix
}

// OpenSelectedDatabase opens the selected database and switches to the tables section.
func (m *Model) OpenSelectedDatabase() {
	if m.loadingDBs || m.loadingTables {
		return
	}
	if m.selectedDB < 0 || m.selectedDB >= len(m.databases) {
		return
	}
	m.openedDB = m.selectedDB
	m.selectedTable = 0
	m.tableOffset = 0
	m.section = 1
	m.clamp()
}

// SwitchSection switches the section to the next or previous one.
func (m *Model) SwitchSection(delta int) {
	if delta == 0 {
		return
	}
	m.section += delta
	if m.section < 0 {
		m.section = 0
	}
	if m.section > 1 {
		m.section = 1
	}
	if m.section == 1 && m.openedDB < 0 {
		m.section = 0
	}
}

// Move moves the selection up or down by the given delta.
func (m *Model) Move(delta int) {
	if m.loadingDBs || (m.loadingTables && m.section == 1) {
		return
	}
	if delta == 0 {
		return
	}
	if m.section == 0 {
		m.selectedDB += delta
		if m.selectedDB < 0 {
			m.selectedDB = 0
		}
		if m.selectedDB >= len(m.databases) {
			m.selectedDB = len(m.databases) - 1
		}
		m.selectedTable = 0
		m.tableOffset = 0
		m.clamp()
		return
	}

	tbls := m.currentTables()
	m.selectedTable += delta
	if m.selectedTable < 0 {
		m.selectedTable = 0
	}
	if m.selectedTable >= len(tbls) {
		m.selectedTable = len(tbls) - 1
	}
	m.clamp()
}

// MoveToTop moves the selection to the top of the current section.
func (m *Model) MoveToTop() {
	if m.loadingDBs || (m.loadingTables && m.section == 1) {
		return
	}
	if m.section == 0 {
		if len(m.databases) > 0 {
			m.selectedDB = 0
		}
		m.selectedTable = 0
		m.tableOffset = 0
		m.clamp()
		return
	}
	m.selectedTable = 0
	m.clamp()
}

// MoveToBottom moves the selection to the bottom of the current section.
func (m *Model) MoveToBottom() {
	if m.loadingDBs || (m.loadingTables && m.section == 1) {
		return
	}
	if m.section == 0 {
		if len(m.databases) > 0 {
			m.selectedDB = len(m.databases) - 1
		}
		m.selectedTable = 0
		m.tableOffset = 0
		m.clamp()
		return
	}
	tbls := m.currentTables()
	if len(tbls) > 0 {
		m.selectedTable = len(tbls) - 1
	}
	m.clamp()
}

// SelectDatabase sets the selected database to the one with the given name.
// If the name is not found, the selection is unchanged.
func (m *Model) SelectDatabase(name string) {
	if name == "" {
		return
	}
	for i, db := range m.databases {
		if db == name {
			m.selectedDB = i
			return
		}
	}
}

// SetDatabaseLabel sets the label for the database.
func (m *Model) SetDatabaseLabel(label string) {
	m.databaseLabel = label
}

// isDigitKey reports whether the key is a single digit 0-9 (for count prefix).
func isDigitKey(keyMsg tea.KeyPressMsg) bool {
	s := keyMsg.String()
	if len(s) != 1 {
		return false
	}
	c := s[0]
	return c >= '0' && c <= '9'
}

// parseBufferCount returns the count from the buffer (default 1); used for vim-style N motion.
func parseBufferCount(buffer string) int {
	if buffer == "" {
		return 1
	}
	count, err := strconv.Atoi(buffer)
	if err != nil || count < 1 {
		return 1
	}
	return count
}
