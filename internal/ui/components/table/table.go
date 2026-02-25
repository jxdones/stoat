package table

import (
	"maps"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jxdones/stoat/internal/ui/keys"
	"github.com/jxdones/stoat/internal/ui/theme"
)

// Table layout and viewport constants.
const (
	tableDefaultWidth  = 80
	tableDefaultHeight = 20
	tableMinWidth      = 20
	tableMinHeight     = 2

	columnMinWidthFloor = 4

	headerRowCount     = 1
	rowNumberMinDigits = 2
	minBodyRowsVisible = 1
	columnGapWidth     = 1
	minHorizontalSpace = 1
	offsetSearchBuffer = 2

	uiIndexBase    = 1
	maxCountDigits = 3
)

// Model represents a table with columns and rows.
type Model struct {
	width  int
	height int

	columns []Column
	rows    []Row

	rowIndex  int
	rowOffset int
	colIndex  int
	colOffset int // colOffset is the first column index of the current horizontal viewport

	countBuffer string
}

// Column stores metadata about a table column.
type Column struct {
	Key      string
	Title    string
	Type     string
	MinWidth int
	Order    int
}

// Row stores one record keyed by column key.
type Row map[string]string

// columnWindow represents a contiguous range of columns within the table.
type columnWindow struct {
	indices []int
	start   int
	end     int
}

// New creates a new table model with the given columns and rows.
func New(columns []Column, rows []Row) Model {
	m := Model{
		columns: normalizeColumns(columns),
		rows:    rows,
	}
	m.SetSize(tableDefaultWidth, tableDefaultHeight)
	return m
}

// SetSize clamps dimensions and recomputes visible row/column windows.
func (m *Model) SetSize(width, height int) {
	if width < tableMinWidth {
		width = tableMinWidth
	}
	if height < tableMinHeight {
		height = tableMinHeight
	}
	m.width = width
	m.height = height
	m.clampSelection()
	m.ensureVisibleColumn()
	m.ensureVisibleRow()
}

// SetColumns replaces the columns and keeps selection/viewport valid.
func (m *Model) SetColumns(columns []Column) {
	m.columns = normalizeColumns(columns)
	m.clampSelection()
	m.ensureVisibleColumn()
	m.ensureVisibleRow()
}

// SetRows replaces the dataset and keeps select/viewport valid.
func (m *Model) SetRows(rows []Row) {
	m.rows = rows
	m.clampSelection()
	m.ensureVisibleColumn()
	m.ensureVisibleRow()
}

// Columns returns a copy of the columns in the model.
func (m Model) Columns() []Column {
	out := make([]Column, len(m.columns))
	copy(out, m.columns)
	return out
}

// RowCount returns the number of rows in the model.
func (m Model) RowCount() int {
	return len(m.rows)
}

// ColumnCount returns the number of columns in the model.
func (m Model) ColumnCount() int {
	return len(m.columns)
}

// VisibleColumnCount returns the number of columns that are visible in the viewport.
func (m Model) VisibleColumnCount() int {
	return len(m.visibleColumns().indices)
}

// CursorPosition returns the current cursor position in the model.
func (m Model) CursorPosition() (line, column int) {
	if len(m.rows) == 0 {
		return 0, 0
	}
	return m.rowIndex + uiIndexBase, m.colIndex + uiIndexBase
}

// ActiveCell returns the current active cell in the model.
func (m Model) ActiveCell() (Column, string, bool) {
	if len(m.rows) == 0 || len(m.columns) == 0 {
		return Column{}, "", false
	}
	if m.rowIndex < 0 || m.rowIndex >= len(m.rows) || m.colIndex < 0 || m.colIndex >= len(m.columns) {
		return Column{}, "", false
	}
	col := m.columns[m.colIndex]
	return col, m.rows[m.rowIndex][col.Key], true
}

// ActiveRow returns the current active row in the model.
func (m Model) ActiveRow() (Row, bool) {
	if len(m.rows) == 0 {
		return nil, false
	}
	if m.rowIndex < 0 || m.rowIndex >= len(m.rows) {
		return nil, false
	}
	return maps.Clone(m.rows[m.rowIndex]), true
}

// Update handles key messages and updates the model state.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	if isDigitKey(keyMsg) {
		if len(m.countBuffer) < maxCountDigits {
			m.countBuffer += keyMsg.String()
		}
		return m, nil
	}

	switch {
	case key.Matches(keyMsg, keys.Default.MoveUp):
		n := parseBufferCount(m.countBuffer)
		m.countBuffer = ""
		m.rowIndex -= n
	case key.Matches(keyMsg, keys.Default.MoveDown):
		n := parseBufferCount(m.countBuffer)
		m.countBuffer = ""
		m.rowIndex += n
	case key.Matches(keyMsg, keys.Default.MoveLeft):
		n := parseBufferCount(m.countBuffer)
		m.countBuffer = ""
		m.colIndex -= n
	case key.Matches(keyMsg, keys.Default.MoveRight):
		n := parseBufferCount(m.countBuffer)
		m.countBuffer = ""
		m.colIndex += n
	case key.Matches(keyMsg, keys.Default.GotoTop):
		m.countBuffer = ""
		m.rowIndex = 0
	case key.Matches(keyMsg, keys.Default.GotoBottom):
		m.countBuffer = ""
		m.rowIndex = len(m.rows) - 1
	default:
		m.countBuffer = ""
		return m, nil
	}

	m.clampSelection()
	m.ensureVisibleColumn()
	m.ensureVisibleRow()
	return m, nil
}

// HelpBindings returns the key bindings for the table.
func HelpBindings() []key.Binding {
	return []key.Binding{
		key.NewBinding(
			key.WithKeys("up", "down", "left", "right", "h", "j", "k", "l"),
			key.WithHelp("h/j/k/l", "move"),
		),
		key.NewBinding(
			key.WithKeys("home", "g", "end", "G"),
			key.WithHelp("g/G", "top/bottom"),
		),
		key.NewBinding(
			key.WithKeys("ctrl+n", "ctrl+b"),
			key.WithHelp("ctrl+n/b", "next/prev page"),
		),
		key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "edit cell"),
		),
	}
}

// View renders a fixed-size table viewport with visible columns and rows.
func (m Model) View() string {
	window := m.visibleColumns()
	lines := make([]string, 0, m.height)
	lines = append(lines, m.renderHeader(window.indices))
	lines = append(lines, m.renderBody(window.indices)...)
	return lipgloss.NewStyle().Width(m.width).Height(m.height).Render(strings.Join(lines, "\n"))
}

// renderHeader renders the visible column titles with fixed width cells.
func (m Model) renderHeader(columns []int) string {
	parts := make([]string, 0, len(columns)+1)
	headStyle := lipgloss.NewStyle().Foreground(theme.Current.TableHeader).Bold(true)
	parts = append(parts, padOrTrim("#", m.rowNumberWidth(), headStyle))
	for _, index := range columns {
		col := m.columns[index]
		parts = append(parts, padOrTrim(col.Title, col.MinWidth, headStyle))
	}
	return strings.Join(parts, " ")
}

// renderBody renders the visible rows in the table viewport.
func (m Model) renderBody(visibleColumns []int) []string {
	rowsAvailable := m.bodyRowsVisible()
	lines := make([]string, 0, rowsAvailable)

	for i := 0; i < rowsAvailable; i++ {
		rowIndex := m.rowOffset + i
		if rowIndex >= len(m.rows) {
			lines = append(lines, strings.Repeat(" ", max(1, m.width)))
			continue
		}
		lines = append(lines, m.renderRow(rowIndex, visibleColumns))
	}
	return lines
}

// renderRow renders one row with both row and active-cell highlighting.
func (m Model) renderRow(rowIndex int, columns []int) string {
	parts := make([]string, 0, len(columns)+1)
	row := m.rows[rowIndex]

	rowStyle := lipgloss.NewStyle().Foreground(theme.Current.TextMuted)
	if rowIndex == m.rowIndex {
		rowStyle = lipgloss.NewStyle().Foreground(theme.Current.Border)
	}

	rowNumber := rowIndex + uiIndexBase
	parts = append(parts, padOrTrim(strconv.Itoa(rowNumber), m.rowNumberWidth(), rowStyle))
	for _, columnIndex := range columns {
		column := m.columns[columnIndex]
		cell := normalizeCellText(row[column.Key])
		style := lipgloss.NewStyle().Foreground(theme.Current.TextPrimary)
		if rowIndex == m.rowIndex {
			style = style.Background(theme.Current.Border)
		}

		if rowIndex == m.rowIndex && columnIndex == m.colIndex {
			style = lipgloss.NewStyle().Foreground(theme.Current.TabsActiveText).Background(theme.Current.TabsActiveBg).Bold(true)
		}

		parts = append(parts, padOrTrim(cell, column.MinWidth, style))
	}
	return strings.Join(parts, " ")
}

// normalizeColumns enforces minimum widths and deterministic render order.
func normalizeColumns(columns []Column) []Column {
	out := make([]Column, len(columns))
	copy(out, columns)

	for i := range out {
		if out[i].MinWidth < columnMinWidthFloor {
			out[i].MinWidth = columnMinWidthFloor
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		return compareColumn(out[i], out[j]) < 0
	})
	return out
}

// compareColumn compares two columns for sorting.
func compareColumn(a, b Column) int {
	if a.Order != b.Order {
		if a.Order < b.Order {
			return -1
		}
		return 1
	}
	if a.Title < b.Title {
		return -1
	}
	if a.Title > b.Title {
		return 1
	}
	return 0
}

// clampSelection ensures the selection/cursor is within the bounds of the model
func (m *Model) clampSelection() {
	if len(m.rows) == 0 {
		m.rowIndex = 0
	} else {
		if m.rowIndex < 0 {
			m.rowIndex = 0
		} else if m.rowIndex >= len(m.rows) {
			m.rowIndex = len(m.rows) - 1
		}
	}

	if len(m.columns) == 0 {
		m.colIndex = 0
	} else {
		if m.colIndex < 0 {
			m.colIndex = 0
		} else if m.colIndex >= len(m.columns) {
			m.colIndex = len(m.columns) - 1
		}
	}
}

// visibleColumns returns the indices of the columns that fit in the current horizontal viewport.
func (m Model) visibleColumns() columnWindow {
	indices := make([]int, 0, len(m.columns))
	space := max(m.width-(m.rowNumberWidth()+columnGapWidth), minHorizontalSpace)

	start := max(0, min(m.colOffset, len(m.columns)))
	end := start
	usedSpace := 0

	for end < len(m.columns) {
		index := end
		columnSlotWidth := m.columns[index].MinWidth + columnGapWidth
		if usedSpace+columnSlotWidth > space && end > start {
			break
		}

		if columnSlotWidth > space && end == start {
			indices = append(indices, index)
			end++
			break
		}

		if usedSpace+columnSlotWidth > space {
			break
		}

		usedSpace += columnSlotWidth
		indices = append(indices, index)
		end++
	}

	return columnWindow{
		indices: indices,
		start:   start,
		end:     end,
	}
}

// ensureVisibleColumn adjusts colOffset so the selected column (colIndex) is inside the visible window.
func (m *Model) ensureVisibleColumn() {
	if len(m.columns) == 0 {
		m.colOffset = 0
		return
	}

	for i := 0; i < len(m.columns)+offsetSearchBuffer; i++ {
		window := m.visibleColumns()
		if m.colIndex >= window.start && m.colIndex < window.end {
			return
		}
		if m.colIndex < window.start {
			m.colOffset--
			if m.colOffset < 0 {
				m.colOffset = 0
				return
			}
			continue
		}
		m.colOffset++
	}
}

// ensureVisibleRow ensures the current row index is visible in the viewport.
func (m *Model) ensureVisibleRow() {
	rowsAvail := m.bodyRowsVisible()
	if m.rowIndex < m.rowOffset {
		m.rowOffset = m.rowIndex
	}
	if m.rowIndex >= m.rowOffset+rowsAvail {
		m.rowOffset = m.rowIndex - rowsAvail + 1
	}
	if m.rowOffset < 0 {
		m.rowOffset = 0
	}
}

// bodyRowsVisible returns the number of rows that are visible in the body of the table.
func (m Model) bodyRowsVisible() int {
	rowsAvailable := m.height - headerRowCount
	return max(rowsAvailable, minBodyRowsVisible)
}

// rowNumberWidth returns the width of the row number gutter.
func (m Model) rowNumberWidth() int {
	digits := len(strconv.Itoa(max(1, len(m.rows))))
	return max(digits, rowNumberMinDigits)
}

// padOrTrim enforces fixed-width cells using rune-aware trimming and padding.
func padOrTrim(s string, width int, style lipgloss.Style) string {
	if width < 1 {
		width = 1
	}
	r := []rune(s)
	if len(r) > width {
		r = r[:max(0, width-1)]
		s = string(r) + "~"
	}
	if len([]rune(s)) < width {
		s += strings.Repeat(" ", width-len([]rune(s)))
	}
	return style.Render(s)
}

// normalizeCellText replaces newline, tab, and carriage return with a space.
func normalizeCellText(s string) string {
	return strings.NewReplacer("\r", " ", "\n", " ", "\t", " ").Replace(s)
}

// isDigitKey reports whether the key is a single digit 0-9 (for count prefix).
func isDigitKey(keyMsg tea.KeyMsg) bool {
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
