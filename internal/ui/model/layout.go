package model

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

	mainHeaderRows = 5
	mainTabsRows   = 3
	mainDetailRows = 1
	mainQueryRows  = 3
	minTableRows   = 1
	minSectionRows = 1
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
func computeMainSections(height int) mainSections {
	sections := mainSections{
		header: mainHeaderRows,
		tabs:   mainTabsRows,
		detail: mainDetailRows,
		query:  mainQueryRows,
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
func computeLayout(width, height, optionsHeight int) layout {
	cols := computeColumns(width)
	rows := computeRows(height, optionsHeight)
	main := computeMainSections(rows.mainContent)
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
