package common

import (
	"image/color"

	"charm.land/lipgloss/v2"

	"github.com/jxdones/stoat/internal/ui/theme"
)

const (
	noVerticalPaddingRows = 0
	horizontalPaddingCols = 1
)

// FocusBorder returns the border color for the given focused state.
func FocusBorder(focused bool) color.Color {
	if focused {
		return theme.Current.BorderFocused
	}
	return theme.Current.Border
}

// BorderedBox returns a style for a bordered box with the given width and border color.
func BorderedBox(width int, borderColor color.Color) lipgloss.Style {
	return lipgloss.NewStyle().
		Width(ClampMin(width, minDimension)).
		Border(lipgloss.NormalBorder()).
		BorderForeground(borderColor).
		Padding(noVerticalPaddingRows, horizontalPaddingCols)
}

// BorderedPane returns a style for a bordered pane with the given width, height, and border color.
func BorderedPane(width, height int, focused bool, borderColor color.Color) lipgloss.Style {
	return BorderedBox(width, borderColor).
		Height(ClampMin(height, minDimension))
}

// DividerTopRow returns a style for a divider top row with the given width and border color.
func DividerTopRow(width int, borderColor color.Color) lipgloss.Style {
	return lipgloss.NewStyle().
		Width(width).
		BorderTop(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(borderColor).
		Padding(noVerticalPaddingRows, horizontalPaddingCols)
}
