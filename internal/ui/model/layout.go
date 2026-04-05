package model

import (
	"strings"

	"github.com/jxdones/stoat/internal/ui/common"
	"github.com/jxdones/stoat/internal/ui/viewstate"
)

const (
	cellDetailMinWidth     = 40
	cellDetailMinHeight    = 10
	cellDetailMaxWidth     = 100
	cellDetailMaxHeight    = 30
	cellDetailScreenMargin = 4 // minimum gap between modal edge and screen edge
)

const (
	narrowLayoutThreshold = 52
	minLeftPaneNarrow     = 16
	minMainPaneNarrow     = 38
	leftPaneWideFraction  = 5
	minMainPaneForMaxLeft = 28
	minLeftPane           = 18
	minMainPane           = 20
	minHeightRow          = 1
	paneGap               = 0

	mainHeaderRows       = 5
	mainTabsRows         = 3
	mainDetailRowsNormal = 1
	mainDetailRowsEdit   = 3
	mainQueryRows        = 3
	maxQueryRows         = 10
	queryGrowthThreshold = 30 // terminal height at which the query box starts growing
	minTableRows         = 1
	minSectionRows       = 1
)

// layoutColumns represents the two primary panes in the UI layout.
type layoutColumns struct {
	leftPane int
	mainPane int
}

// layoutRows represents the three main content rows in the UI layout.
type layoutRows struct {
	mainContent int
	statusRow   int
	optionsRow  int
}

// layout represents the two primary panes in the UI layout.
type layout struct {
	columns layoutColumns
	rows    layoutRows
	main    mainSections
}

// mainSections represents section heights in the main content pane.
type mainSections struct {
	header int
	tabs   int
	table  int
	detail int
	query  int
}

// computeColumns computes the optimal layout columns for a given width.
func computeColumns(width int) layoutColumns {
	if width <= 0 {
		return layoutColumns{}
	}

	var leftPane int
	if width < narrowLayoutThreshold {
		leftPane = max(minLeftPaneNarrow, width-minMainPaneNarrow)
	} else {
		leftPane = width / leftPaneWideFraction
	}

	maxLeft := width - minMainPaneForMaxLeft
	leftPane = clampRange(leftPane, minLeftPane, maxLeft)

	mainPane := width - leftPane - paneGap
	if mainPane < minMainPane {
		mainPane = minMainPane
		leftPane = max(0, width-mainPane-paneGap)
	}

	return layoutColumns{
		leftPane: leftPane,
		mainPane: mainPane,
	}
}

// computeRows computes the optimal layout rows for a given height.
func computeRows(height, optionsHeight int) layoutRows {
	if height <= 0 {
		return layoutRows{}
	}

	statusRow := minHeightRow
	optionsRow := optionsHeight
	mainContent := height - statusRow - optionsRow

	if mainContent < 1 {
		statusRow = 0
		mainContent = max(0, height-minHeightRow)
	}

	return layoutRows{
		mainContent: mainContent,
		statusRow:   statusRow,
		optionsRow:  optionsRow,
	}
}

// computeMainSections derives heights for main-pane sections.
func computeMainSections(height, detailRows, queryRows int) mainSections {
	sections := mainSections{
		header: mainHeaderRows,
		tabs:   mainTabsRows,
		detail: detailRows,
		query:  queryRows,
	}

	tableHeight := height - (sections.header + sections.tabs + sections.detail + sections.query)
	if tableHeight < minTableRows {
		rowsNeededForTable := minTableRows - tableHeight

		rowsToTake := min(rowsNeededForTable, sections.query-minSectionRows)
		sections.query -= rowsToTake
		rowsNeededForTable -= rowsToTake

		rowsToTake = min(rowsNeededForTable, sections.header-minSectionRows)
		sections.header -= rowsToTake
		rowsNeededForTable -= rowsToTake

		tableHeight = height - (sections.header + sections.tabs + sections.detail + sections.query)
		if rowsNeededForTable > 0 {
			tableHeight -= rowsNeededForTable
		}
	}
	if tableHeight < 1 {
		tableHeight = 1
	}
	sections.table = tableHeight

	return sections
}

// computeLayout computes the full frame layout for the current terminal size.
func computeLayout(width, height, optionsHeight, detailRows, queryRows int) layout {
	cols := computeColumns(width)
	rows := computeRows(height, optionsHeight)
	main := computeMainSections(rows.mainContent, detailRows, queryRows)
	return layout{
		columns: cols,
		rows:    rows,
		main:    main,
	}
}

// clampRange returns value if value is between min and max, otherwise min or max.
func clampRange(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// applyViewState updates component sizes and focus based on the current terminal dimensions.
func (m *Model) applyViewState() {
	var optionsHeight int
	if m.helpExpanded {
		optionsHeight = expandedOptionsHeight(m.view.width, m.fullHelpBindings())
	} else {
		optionsHeight = 2
	}

	frame := computeLayout(m.view.width, m.view.height, optionsHeight, m.detailRows(), m.effectiveQueryRows())

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
	if m.mode == modeInsert {
		m.editbox.Focus()
	} else {
		m.editbox.Blur()
	}

	m.table.SetSize(
		common.BoxInnerWidth(frame.columns.mainPane),
		common.PaneInnerHeight(frame.main.table),
	)

	m.fkViewport.SetWidth(common.BoxInnerWidth(frame.columns.mainPane))
	m.fkViewport.SetHeight(common.PaneInnerHeight(frame.main.table))

	cdWidth := min(clampRange(m.view.width*2/3, cellDetailMinWidth, cellDetailMaxWidth), m.view.width-cellDetailScreenMargin)
	cdHeight := min(clampRange(m.view.height/2, cellDetailMinHeight, cellDetailMaxHeight), m.view.height-cellDetailScreenMargin)
	cdHeight = m.cellDetail.PreferredHeight(cdHeight)
	m.cellDetail.SetSize(cdWidth, cdHeight)

	label, color := m.modeStyle()
	m.statusbar.SetMode(label, color)
}

// effectiveQueryRows returns the query box height in rows for the current model state.
// On short terminals (< queryGrowthThreshold), it returns the fixed minimum.
// On taller terminals, it grows with the content up to maxQueryRows.
func (m Model) effectiveQueryRows() int {
	if m.view.height < queryGrowthThreshold {
		return mainQueryRows
	}
	lines := strings.Count(m.querybox.Value(), "\n") + 1
	return clampRange(lines+2, mainQueryRows, maxQueryRows)
}
