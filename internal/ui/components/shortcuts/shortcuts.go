package shortcuts

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/jxdones/stoat/internal/ui/theme"
)

const (
	shortcutsHorizontalPaddingColumns = 2
	shortcutsMinContentWidth          = 8
)

// RenderShortcuts renders the shortcuts for the given width and bindings.
func RenderShortcuts(width int, bindings []key.Binding) string {
	contentWidth := max(shortcutsMinContentWidth, width-shortcutsHorizontalPaddingColumns)

	helpModel := help.New()
	helpModel.Width = contentWidth
	helpModel.Styles.ShortKey = lipgloss.NewStyle().Foreground(theme.Current.TextAccent)
	helpModel.Styles.ShortDesc = lipgloss.NewStyle().Foreground(theme.Current.TextMuted)
	helpModel.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(theme.Current.DividerBorder)
	helpModel.Styles.Ellipsis = lipgloss.NewStyle().Foreground(theme.Current.DividerBorder)

	return ansi.Truncate(helpModel.ShortHelpView(bindings), contentWidth, "…")
}
