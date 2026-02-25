package tabs

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"

	"github.com/jxdones/stoat/internal/ui/common"
	"github.com/jxdones/stoat/internal/ui/theme"
	"github.com/jxdones/stoat/internal/ui/viewstate"
)

// Model represents a tabs component with a list of tabs.
type Model struct {
	tabs   []string
	active int
	width  int
	focus  bool
}

// New creates a new tabs model with the given tabs. The first tab is active.
func New(tabs []string) Model {
	return Model{
		tabs:   tabs,
		active: 0,
		width:  24,
	}
}

// SetActive sets the active tab by index (0-based). No-op if index is out of range.
func (m *Model) SetActive(index int) {
	if index < 0 || index >= len(m.tabs) {
		return
	}
	m.active = index
}

// ActiveTab returns the label of the active tab, or "" if none.
func (m Model) ActiveTab() string {
	if m.active < 0 || m.active >= len(m.tabs) {
		return ""
	}
	return m.tabs[m.active]
}

// SetSize sets the width of the tabs component (clamped to at least 24).
func (m *Model) SetSize(width int) {
	m.width = common.ClampMin(width, 24)
}

// SetFocused sets the focused state (border color in View reflects focus).
func (m *Model) SetFocused(focused bool) {
	m.focus = focused
}

// ApplyViewState applies the view state (width and focus) to the tabs component.
func (m *Model) ApplyViewState(viewState viewstate.ViewState) {
	m.SetSize(viewState.Width)
	m.SetFocused(viewState.Focused)
}

// View renders the tabs component as a single-line bordered box. The active tab
// is highlighted; content is clipped with "…" when it exceeds the box width.
func (m Model) View() string {
	width := common.ClampMin(m.width, 24)
	contentWidth := common.BoxContentWidth(width)

	prefixText := "Sections: "
	prefix := lipgloss.NewStyle().Foreground(theme.Current.TabsPrefix).Render(prefixText)
	used := len([]rune(prefixText))
	line := prefix

	for i, tab := range m.tabs {
		partText := fmt.Sprintf("%d:%s", i+1, tab)
		segmentText := partText
		if i > 0 {
			segmentText = " | " + partText
		}

		style := lipgloss.NewStyle().Foreground(theme.Current.TextPrimary).Bold(true)
		if i == m.active {
			style = lipgloss.NewStyle().Foreground(theme.Current.TableHeader).Bold(true)
		}

		remaining := contentWidth - used
		if remaining == 0 {
			break
		}
		if len([]rune(segmentText)) > remaining {
			clipped := "…"
			if remaining == 1 {
				line += style.Render(clipped)
			} else {
				clipped = string([]rune(segmentText)[:remaining-1]) + "…"
				line += style.Render(clipped)
			}
			used = contentWidth
			break
		}

		line += style.Render(segmentText)
		used += len([]rune(segmentText))
	}

	if used < contentWidth {
		line += strings.Repeat(" ", contentWidth-used)
	}

	return common.BorderedBox(width, common.FocusBorder(m.focus)).
		Render(line)
}

// HelpBindings returns the key bindings for switching sections.
func HelpBindings() []key.Binding {
	return []key.Binding{
		key.NewBinding(
			key.WithKeys("ctrl+1", "ctrl+2", "ctrl+3", "ctrl+4", "ctrl+5"),
			key.WithHelp("ctrl+1-5", "switch section"),
		),
	}
}
