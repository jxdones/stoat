package model

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/jxdones/stoat/internal/ui/common"
	"github.com/jxdones/stoat/internal/ui/components/filterbox"
	"github.com/jxdones/stoat/internal/ui/components/querybox"
	"github.com/jxdones/stoat/internal/ui/components/shortcuts"
	"github.com/jxdones/stoat/internal/ui/components/sidebar"
	"github.com/jxdones/stoat/internal/ui/components/table"
	"github.com/jxdones/stoat/internal/ui/theme"
)

const (
	tablePaneMinOuterHeight = 3
	maxTableShrinkPasses    = 8
	minRenderWidth          = 1

	minTerminalWidth   = 80
	minTerminalHeight  = 24
	minPaneInnerHeight = 1

	noDataSourcePlaceholder = "No data source connected.\n\nPress Esc then q to exit, or Ctrl+C"
	selectTablePlaceholder  = "Select a table from the sidebar and press Enter to view data."
)

// View renders the UI layout.
func (m Model) View() tea.View {
	m.applyViewState()
	if m.view.compact {
		full := normalizeCanvas(m.compactView(), m.view.width, m.view.height)
		v := tea.NewView(full)
		v.AltScreen = true
		return v
	}

	frame := computeLayout(m.view.width, m.view.height)

	base := normalizeCanvas(m.renderBase(frame), m.view.width, frame.rows.mainContent)
	lines := []string{base}

	if frame.rows.statusRow > 0 {
		lines = append(lines, m.renderStatus())
	}
	lines = append(lines, m.renderOptions())

	full := normalizeCanvas(strings.Join(lines, "\n"), m.view.width, m.view.height)
	v := tea.NewView(full)
	v.AltScreen = true
	return v
}

// renderBase renders the base area of the UI layout.
func (m Model) renderBase(frame layout) string {
	if frame.rows.mainContent <= 0 {
		return ""
	}

	leftPane := normalizeCanvas(m.sidebar.View().Content, frame.columns.leftPane, frame.rows.mainContent)
	header := m.renderHeader(frame.columns.mainPane)
	tabs := normalizeCanvas(m.tabs.View().Content, frame.columns.mainPane, frame.main.tabs)
	detail := m.renderDetail(frame.columns.mainPane)
	query := normalizeCanvas(m.querybox.View().Content, frame.columns.mainPane, frame.main.query)

	fixed := lipgloss.JoinVertical(lipgloss.Top, header, tabs, detail, query)
	fixedHeight := lipgloss.Height(fixed)
	tableOuterHeight := max(tablePaneMinOuterHeight, frame.rows.mainContent-fixedHeight)

	var mainRaw string
	for range maxTableShrinkPasses {
		m.table.SetSize(
			common.BoxInnerWidth(frame.columns.mainPane),
			common.PaneInnerHeight(tableOuterHeight),
		)
		table := m.renderTable(frame.columns.mainPane, tableOuterHeight)
		mainRaw = lipgloss.JoinVertical(lipgloss.Top, header, tabs, table, detail, query)

		overflow := lipgloss.Height(mainRaw) - frame.rows.mainContent
		if overflow <= 0 {
			break
		}

		tableOuterHeight -= overflow
		if tableOuterHeight < tablePaneMinOuterHeight {
			tableOuterHeight = tablePaneMinOuterHeight
			m.table.SetSize(
				common.BoxInnerWidth(frame.columns.mainPane),
				common.PaneInnerHeight(tableOuterHeight),
			)
			table := m.renderTable(frame.columns.mainPane, tableOuterHeight)
			mainRaw = lipgloss.JoinVertical(lipgloss.Top, header, tabs, table, detail, query)
			break
		}
	}

	rightPane := normalizeCanvas(mainRaw, frame.columns.mainPane, frame.rows.mainContent)

	gap := lipgloss.NewStyle().
		Width(paneGap).
		Height(frame.rows.mainContent).
		Render("")

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, gap, rightPane)
	return normalizeCanvas(body, frame.columns.leftPane+paneGap+frame.columns.mainPane, frame.rows.mainContent)
}

// renderHeader renders the header area of the UI layout.
func (m Model) renderHeader(width int) string {
	db := m.sidebar.EffectiveDB()
	table := m.sidebar.SelectedTable()

	target := "No connection"
	if db != "" {
		target = db
	}

	if db != "" && table != "" && table != "(none)" {
		target = db + "." + table
	}

	title := lipgloss.NewStyle().Foreground(theme.Current.TextAccent).Bold(true).Render(target)
	filterLabel := lipgloss.NewStyle().Foreground(theme.Current.TextPrimary).Render("Filter:")
	filterLine := lipgloss.JoinHorizontal(lipgloss.Left, filterLabel, " ", m.filterbox.View().Content)
	line1 := lipgloss.JoinVertical(lipgloss.Left, title, filterLine)

	rowsShown := m.table.RowCount()
	rowsWord := "rows"
	if rowsShown == 1 {
		rowsWord = "row"
	}
	pageNum := len(m.paging.afterStack)
	pageNotLoaded := pageNum == 1 && strings.TrimSpace(m.paging.afterStack[0]) == "" &&
		m.table.ColumnCount() == 0 &&
		m.table.RowCount() == 0

	if pageNotLoaded {
		pageNum = 0
	}

	showing := fmt.Sprintf("page %d | %d %s", pageNum, rowsShown, rowsWord)

	line2 := lipgloss.NewStyle().Foreground(theme.Current.TextHeader).Render(
		"columns: " + strconv.Itoa(m.table.ColumnCount()) + ", visible: " + strconv.Itoa(m.table.VisibleColumnCount()) + " | " + showing,
	)

	return common.BorderedBox(width, common.FocusBorder(m.view.focus == FocusFilterbox)).
		Render(lipgloss.JoinVertical(lipgloss.Top, line1, line2))
}

// renderTable renders the table area with an outer pane border.
// When there is no table data, it shows a placeholder: "No data source connected"
// when disconnected, or "Select a table..." when connected but no table opened yet.
func (m Model) renderTable(width, height int) string {
	content := m.table.View().Content
	if m.table.ColumnCount() == 0 && m.table.RowCount() == 0 {
		msg := noDataSourcePlaceholder
		if m.HasConnection() {
			msg = selectTablePlaceholder
		}
		content = lipgloss.NewStyle().Foreground(theme.Current.TextMuted).Render(msg)
	}
	return common.BorderedPane(width, height, m.isFocused(FocusTable), common.FocusBorder(m.isFocused(FocusTable))).
		Render(content)
}

// renderDetail renders the detail area of the UI layout.
func (m Model) renderDetail(width int) string {
	line, col := m.table.CursorPosition()
	column, value, ok := m.table.ActiveCell()

	if !ok {
		txt := lipgloss.NewStyle().Foreground(theme.Current.TextMuted).Render("Ln 0, Col 0 | field: - | type: - | value: -")
		return common.DividerTopRow(width, theme.Current.DividerBorder).Render(txt)
	}

	fieldType := column.Type
	if strings.TrimSpace(fieldType) == "" {
		fieldType = "text"
	}

	head := lipgloss.NewStyle().Foreground(theme.Current.TextPrimary).Render(fmt.Sprintf("Ln %d, Col %d", line, col))
	field := lipgloss.NewStyle().Foreground(theme.Current.TextAccent).Render(column.Title)
	typ := lipgloss.NewStyle().Foreground(theme.Current.TextWarning).Render(fieldType)

	plain := fmt.Sprintf("%s | field: %s | type: %s | value: %s", ansi.Strip(head), ansi.Strip(field), ansi.Strip(typ), value)
	trimmed := ansi.Truncate(plain, max(0, width-2), "…")
	return common.DividerTopRow(width, theme.Current.DividerBorder).
		Render(lipgloss.NewStyle().Foreground(theme.Current.TextMuted).Render(trimmed))
}

// renderStatus renders the status area of the UI layout.
func (m Model) renderStatus() string {
	return m.statusbar.View(m.view.width).Content
}

// renderOptions renders the options area of the UI layout.
func (m Model) renderOptions() string {
	outerWidth := max(minRenderWidth, m.view.width)
	content := shortcuts.RenderShortcuts(outerWidth, m.statusBindings())
	helpLine := lipgloss.NewStyle().
		Width(outerWidth).
		BorderTop(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(theme.Current.DividerBorder).
		Padding(0, 1).
		Render(content)
	return helpLine
}

// normalizeCanvas clips/pads content to an exact width x height rectangle.
// This prevents section-overflow artifacts when terminal space is tight.
func normalizeCanvas(content string, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	lines := strings.Split(content, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}

	for len(lines) < height {
		lines = append(lines, "")
	}

	clippedWidth := max(minRenderWidth, width)

	for i := range lines {
		line := ansi.Truncate(lines[i], clippedWidth, "")
		lineWidth := ansi.StringWidth(line)
		if lineWidth < clippedWidth {
			line += strings.Repeat(" ", clippedWidth-lineWidth)
		}
		lines[i] = line
	}

	return strings.Join(lines, "\n")
}

// compactView renders the compact view of the UI layout.
func (m Model) compactView() string {
	// No status bar in compact view; only message area + options row.
	contentH := max(minPaneInnerHeight, m.view.height-1)
	title := lipgloss.NewStyle().Foreground(theme.Current.TextWarning).Bold(true).Render("Terminal too small")
	body := []string{
		title,
		"",
		fmt.Sprintf("Current: %dx%d", m.view.width, m.view.height),
		fmt.Sprintf("Minimum: %dx%d", minTerminalWidth, minTerminalHeight),
		"",
		"Resize the terminal to continue using the full UI.",
		"Keys: q quit, Tab cycle focus, Esc clear focus.",
	}
	msg := normalizeCanvas(strings.Join(body, "\n"), m.view.width, contentH)
	return normalizeCanvas(lipgloss.JoinVertical(lipgloss.Left, msg, m.renderOptions()), m.view.width, m.view.height)
}

// statusBindings returns the key bindings for the status area.
func (m Model) statusBindings() []key.Binding {
	globalBindings := []key.Binding{
		key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "focus panes"),
		),
		key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "focus previous pane"),
		),
		key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "focus filterbox"),
		),
		key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "clear focus"),
		),
	}
	if m.view.focus == FocusNone {
		return append(globalBindings, key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		))
	}

	var paneBindings []key.Binding
	switch m.view.focus {
	case FocusSidebar:
		paneBindings = sidebar.HelpBindings()
	case FocusFilterbox:
		paneBindings = filterbox.HelpBindings()
	case FocusTable:
		paneBindings = table.HelpBindings()
	case FocusQuerybox:
		paneBindings = querybox.HelpBindings()
	}
	return append(paneBindings, globalBindings...)
}
