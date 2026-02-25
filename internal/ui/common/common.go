package common

const (
	minDimension              = 1
	boxHorizontalFrameColumns = 4

	paneVerticalBorderRows = 2
)

// ClampMin enforces a lower bound: it returns value if value >= min, otherwise min.
// For layout dimensions, pass a positive min (e.g. 1) so the result is never
// negative or zero, avoiding invalid sizes in lipgloss width/height APIs.
func ClampMin(value, min int) int {
	if value < min {
		return min
	}
	return value
}

// BoxInnerWidth is used to clamp the width of a box's inner content.
// The "-4" accounts for left+right border (2) plus horizontal padding (2).
func BoxInnerWidth(outerWidth int) int {
	return ClampMin(outerWidth-boxHorizontalFrameColumns, minDimension)
}

// PaneInnerHeight is used to clamp the height of a pane's inner content.
// The "-2" removes top and bottom borders.
func PaneInnerHeight(outerHeight int) int {
	return ClampMin(outerHeight-paneVerticalBorderRows, minDimension)
}
