package querybox

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jxdones/stoat/internal/ui/common"
	"github.com/jxdones/stoat/internal/ui/theme"
)

// Model represents a query box with an input textarea.
type Model struct {
	input  textarea.Model
	width  int
	height int
	focus  bool
}

// New creates a new query box model with the default configuration.
func New() Model {
	ta := textarea.New()
	ta.Prompt = "sql> "
	ta.Placeholder = "Enter your query here..."
	ta.ShowLineNumbers = true
	ta.CharLimit = 10000
	ta.SetHeight(1)

	ta.FocusedStyle.Prompt = ta.FocusedStyle.Prompt.Foreground(theme.Current.TextAccent)
	ta.FocusedStyle.Text = ta.FocusedStyle.Text.Foreground(theme.Current.TextPrimary)
	ta.FocusedStyle.CursorLine = ta.FocusedStyle.CursorLine.Foreground(theme.Current.TextPrimary)
	ta.FocusedStyle.CursorLineNumber = ta.FocusedStyle.CursorLineNumber.Foreground(theme.Current.TextMuted)
	ta.FocusedStyle.LineNumber = ta.FocusedStyle.LineNumber.Foreground(theme.Current.TextMuted)
	ta.FocusedStyle.Placeholder = ta.FocusedStyle.Placeholder.Foreground(theme.Current.TextMuted)

	ta.BlurredStyle.Prompt = ta.BlurredStyle.Prompt.Foreground(theme.Current.TextMuted)
	ta.BlurredStyle.Text = ta.BlurredStyle.Text.Foreground(theme.Current.TextPrimary)
	ta.BlurredStyle.CursorLine = ta.BlurredStyle.CursorLine.Foreground(theme.Current.TextPrimary)
	ta.BlurredStyle.CursorLineNumber = ta.BlurredStyle.CursorLineNumber.Foreground(theme.Current.TextMuted)
	ta.BlurredStyle.LineNumber = ta.BlurredStyle.LineNumber.Foreground(theme.Current.TextMuted)
	ta.BlurredStyle.Placeholder = ta.BlurredStyle.Placeholder.Foreground(theme.Current.TextMuted)
	ta.Focus()

	return Model{
		input:  ta,
		width:  40,
		height: 3,
		focus:  false,
	}
}

// SetSize clamps dimensions and updates the input textarea size.
func (m *Model) SetSize(width, height int) {
	width = common.ClampMin(width, 24)
	height = common.ClampMin(height, 3)
	m.width = width
	m.height = height
	m.input.SetWidth(common.BoxInnerWidth(width))
	m.input.SetHeight(common.PaneInnerHeight(height))
}

// SetFocused sets the focus state of the query box.
func (m *Model) SetFocused(focused bool) {
	m.focus = focused
}

// Focus sets the focus state of the query box to true.
func (m *Model) Focus() {
	m.focus = true
	m.input.Focus()
}

// Blur sets the focus state of the query box to false.
func (m *Model) Blur() {
	m.focus = false
	m.input.Blur()
}

// Value returns the current value of the query box.
func (m Model) Value() string {
	return m.input.Value()
}

// SetValue sets the value of the query box.
func (m *Model) SetValue(value string) {
	m.input.SetValue(value)
}

// Update handles key messages and updates the model state.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// View renders the query box with the current state.
func (m Model) View() string {
	return common.BorderedPane(m.width, m.height, m.focus, common.FocusBorder(m.focus)).
		Render(m.input.View())
}

// HelpBindings returns the key bindings for the query box.
func HelpBindings() []key.Binding {
	return []key.Binding{
		key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "run query"),
		),
	}
}
