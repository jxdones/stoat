package common

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/jxdones/stoat/internal/ui/theme"
)

const (
	noVerticalPaddingRows = 0
	horizontalPaddingCols = 1
)

// FocusBorder returns the border color for the given focused state.
func FocusBorder(focused bool) lipgloss.Color {
	if focused {
		return theme.Current.BorderFocused
	}
	return theme.Current.Border
}

// BorderedBox returns a style for a bordered box with the given width and border color.
func BorderedBox(width int, borderColor lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().
		Width(ClampMin(width-boxHorizontalFrameColumns, minDimension)).
		Border(lipgloss.NormalBorder()).
		BorderForeground(borderColor).
		Padding(noVerticalPaddingRows, horizontalPaddingCols)
}

// BorderedPane returns a style for a bordered pane with the given width, height, and border color.
func BorderedPane(width, height int, focused bool, borderColor lipgloss.Color) lipgloss.Style {
	return BorderedBox(width, borderColor).
		Height(ClampMin(height-paneVerticalBorderRows, minDimension))
}
