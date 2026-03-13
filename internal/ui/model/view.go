package model

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/jxdones/stoat/internal/database"
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
	helpBorderHeight   = 1
	helpTitleHeight    = 1

	noDataSourcePlaceholder      = "No data source connected.\n\nPress Esc then q to exit, or Ctrl+C"
	selectTablePlaceholder       = "Select a table from the sidebar and press Enter to view data."
	schemaNoTablePlaceholder     = "No table selected."
	schemaQueryResultPlaceholder = "Schema not available for query results."
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

	var optionsHeight int
	if m.helpExpanded {
		optionsHeight = expandedOptionsHeight(m.view.width, m.fullHelpBindings())
	} else {
		optionsHeight = 2
	}

	frame := computeLayout(m.view.width, m.view.height, optionsHeight)

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

	activeTable := m.table
	foreignKeysActive := false
	schemaPlaceholder := ""
	isSchemaTab := m.tabs.ActiveTab() != "Records"
	if isSchemaTab {
		if m.viewingQueryResult {
			schemaPlaceholder = schemaQueryResultPlaceholder
		} else if m.tablePKTarget == (database.DatabaseTarget{}) {
			schemaPlaceholder = schemaNoTablePlaceholder
		}
	}
	switch m.tabs.ActiveTab() {
	case "Columns", "Constraints", "Indexes":
		activeTable = m.schemaTable
	case "Foreign Keys":
		foreignKeysActive = true
	}

	var mainRaw string
	for range maxTableShrinkPasses {
		m.table.SetSize(
			common.BoxInnerWidth(frame.columns.mainPane),
			common.PaneInnerHeight(tableOuterHeight),
		)

		activeTable.SetSize(
			common.BoxInnerWidth(frame.columns.mainPane),
			common.PaneInnerHeight(tableOuterHeight),
		)

		var table string
		if schemaPlaceholder != "" {
			table = m.renderSchemaPlaceholder(frame.columns.mainPane, tableOuterHeight, schemaPlaceholder)
		} else if foreignKeysActive {
			table = m.renderForeignKeys(frame.columns.mainPane, tableOuterHeight)
		} else {
			table = m.renderTable(frame.columns.mainPane, tableOuterHeight, activeTable)
		}
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

			activeTable.SetSize(
				common.BoxInnerWidth(frame.columns.mainPane),
				common.PaneInnerHeight(tableOuterHeight),
			)

			if schemaPlaceholder != "" {
				table = m.renderSchemaPlaceholder(frame.columns.mainPane, tableOuterHeight, schemaPlaceholder)
			} else if foreignKeysActive {
				table = m.renderForeignKeys(frame.columns.mainPane, tableOuterHeight)
			} else {
				table = m.renderTable(frame.columns.mainPane, tableOuterHeight, activeTable)
			}
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
	if m.viewingQueryResult {
		if m.queryResultPreview != "" {
			target = m.queryResultPreview
		} else {
			target = "Query result"
		}
		if db != "" {
			target = db + " — " + target
		}
	} else if db != "" {
		target = db
		if table != "" && table != "(none)" {
			target = db + "." + table
		}
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
func (m Model) renderTable(width, height int, table table.Model) string {
	content := table.View().Content
	if table.ColumnCount() == 0 && table.RowCount() == 0 {
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

	content := ""
	if m.helpExpanded {
		switch m.view.focus {
		case FocusSidebar:
			content += lipgloss.NewStyle().Foreground(theme.Current.TextAccent).Render("Sidebar") + "\n"
		case FocusFilterbox:
			content += lipgloss.NewStyle().Foreground(theme.Current.TextAccent).Render("Filterbox") + "\n"
		case FocusTable:
			content += lipgloss.NewStyle().Foreground(theme.Current.TextAccent).Render("Table") + "\n"
		case FocusQuerybox:
			content += lipgloss.NewStyle().Foreground(theme.Current.TextAccent).Render("Querybox") + "\n"
		case FocusNone:
			content += lipgloss.NewStyle().Foreground(theme.Current.TextAccent).Render("Global") + "\n"
		}

		h := help.New()
		h.SetWidth(outerWidth)
		content += h.FullHelpView(m.fullHelpBindings())
	} else {
		content += shortcuts.RenderShortcuts(outerWidth, m.statusBindings())
	}
	helpLine := lipgloss.NewStyle().
		Width(outerWidth).
		BorderTop(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(theme.Current.DividerBorder).
		Padding(0, 1).
		Render(content)
	return helpLine
}

func expandedOptionsHeight(width int, bindings [][]key.Binding) int {
	helpModel := help.New()
	helpModel.SetWidth(width)
	fullHelp := helpModel.FullHelpView(bindings)
	return lipgloss.Height(fullHelp) + helpBorderHeight + helpTitleHeight
}

// renderSchemaPlaceholder renders a placeholder message inside the schema tab pane.
func (m Model) renderSchemaPlaceholder(width, height int, msg string) string {
	content := lipgloss.NewStyle().Foreground(theme.Current.TextMuted).Render(msg)
	return common.BorderedPane(width, height, m.isFocused(FocusTable), common.FocusBorder(m.isFocused(FocusTable))).
		Render(content)
}

// renderForeignKeys renders the foreign keys area of the UI layout.
func (m Model) renderForeignKeys(width, height int) string {
	content := []string{}
	for _, fk := range m.tableSchema.foreignKeys {
		indent := strings.Repeat(" ", len(fk.Column)+5)
		line := fmt.Sprintf("%s → %s.%s\n", fk.Column, fk.RefTable, fk.RefColumn)
		if fk.OnDeleteAction != "" {
			line += fmt.Sprintf("%son DELETE: %s\n", indent, fk.OnDeleteAction)
		}
		if fk.OnUpdateAction != "" {
			line += fmt.Sprintf("%son UPDATE: %s\n", indent, fk.OnUpdateAction)
		}
		line += "\n"
		content = append(content, line)
	}
	innerHeight := common.PaneInnerHeight(height)
	lines := strings.Split(strings.Join(content, "\n"), "\n")
	if len(lines) > innerHeight {
		lines = lines[:innerHeight]
	}
	clipped := strings.Join(lines, "\n")
	return common.BorderedPane(width, height, m.isFocused(FocusTable), common.FocusBorder(m.isFocused(FocusTable))).
		Render(lipgloss.NewStyle().Foreground(theme.Current.TextMuted).Render(clipped))
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
	paneBindings, globalBindings := m.helpBindings()
	return append(paneBindings, globalBindings...)
}

// fullHelpBindings returns the key bindings for the help area.
func (m Model) fullHelpBindings() [][]key.Binding {
	paneBindings, globalBindings := m.helpBindings()
	return [][]key.Binding{paneBindings, globalBindings}
}

func (m Model) helpBindings() (pane []key.Binding, global []key.Binding) {
	globalBindings := []key.Binding{
		key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
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
		globalBindings = append(globalBindings, key.NewBinding(
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

	return paneBindings, globalBindings
}
