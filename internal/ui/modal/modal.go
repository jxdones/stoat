package modal

import (
	"charm.land/lipgloss/v2"

	"github.com/jxdones/stoat/internal/ui/theme"
)

const (
	horizontalPad    = 1
	borderAndPadding = 4
)

// Render produces a styled modal box with a title, content, and shortcuts footer.
// width is the outer width of the modal including border and padding.
func Render(title, content, shortcuts string, width int) string {
	innerWidth := max(width-borderAndPadding, 1)

	titleLine := lipgloss.NewStyle().
		Foreground(theme.Current.TextHeader).
		Bold(true).
		Render(title)

	footerLine := lipgloss.NewStyle().
		Foreground(theme.Current.OverlayFooter).
		Render(shortcuts)

	body := lipgloss.JoinVertical(lipgloss.Top, titleLine, "", content, "", footerLine)

	return lipgloss.NewStyle().
		Width(innerWidth).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Current.OverlayBorder).
		Padding(0, horizontalPad).
		Render(body)
}

// CenteredLayer returns a lipgloss Layer for the given content, centered within screenW x screenH.
func CenteredLayer(content string, screenW, screenH int) *lipgloss.Layer {
	w := lipgloss.Width(content)
	h := lipgloss.Height(content)
	x := (screenW - w) / 2
	y := (screenH - h) / 2
	return lipgloss.NewLayer(content).X(x).Y(y)
}
