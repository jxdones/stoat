package querybox

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"

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

	ta.SetStyles(themedStyles(ta))
	ta.Focus()

	return Model{
		input:  ta,
		width:  40,
		height: 3,
		focus:  false,
	}
}

// ApplyTheme re-applies the current theme colors to the textarea styles.
// Call this whenever the active theme changes.
func (m *Model) ApplyTheme() {
	m.input.SetStyles(themedStyles(m.input))
}

func themedStyles(ta textarea.Model) textarea.Styles {
	styles := ta.Styles()
	styles.Focused.Prompt = styles.Focused.Prompt.Foreground(theme.Current.TextAccent)
	styles.Focused.Text = styles.Focused.Text.Foreground(theme.Current.TextPrimary)
	styles.Focused.CursorLine = styles.Focused.CursorLine.Foreground(theme.Current.TextPrimary)
	styles.Focused.CursorLineNumber = styles.Focused.CursorLineNumber.Foreground(theme.Current.TextMuted)
	styles.Focused.LineNumber = styles.Focused.LineNumber.Foreground(theme.Current.TextMuted)
	styles.Focused.Placeholder = styles.Focused.Placeholder.Foreground(theme.Current.TextMuted)
	styles.Blurred.Prompt = styles.Blurred.Prompt.Foreground(theme.Current.TextMuted)
	styles.Blurred.Text = styles.Blurred.Text.Foreground(theme.Current.TextPrimary)
	styles.Blurred.CursorLine = styles.Blurred.CursorLine.Foreground(theme.Current.TextPrimary)
	styles.Blurred.CursorLineNumber = styles.Blurred.CursorLineNumber.Foreground(theme.Current.TextMuted)
	styles.Blurred.LineNumber = styles.Blurred.LineNumber.Foreground(theme.Current.TextMuted)
	styles.Blurred.Placeholder = styles.Blurred.Placeholder.Foreground(theme.Current.TextMuted)
	return styles
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

// LineInfo returns the line information of the query box.
func (m Model) LineInfo() textarea.LineInfo {
	return m.input.LineInfo()
}

// AdvanceCursor advances the cursor by n characters.
func (m *Model) AdvanceCursor(n int) {
	for range n {
		m.input, _ = m.input.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyRight}))
	}
}

// Update handles key messages and updates the model state.
// We only intercept ctrl+l (clear); all other keys and messages go to the textarea.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	if k, ok := msg.(tea.KeyPressMsg); ok && k.String() == "ctrl+l" {
		m.input.SetValue("")
		return m, nil
	}
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// View renders the query box with the current state.
func (m Model) View() tea.View {
	content := common.BorderedPane(m.width, m.height, m.focus, common.FocusBorder(m.focus)).
		Render(m.input.View())
	return tea.NewView(content)
}

// HelpBindings returns the key bindings for the query box.
func HelpBindings() []key.Binding {
	return []key.Binding{
		key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "run query"),
		),
		key.NewBinding(
			key.WithKeys("ctrl+n"),
			key.WithHelp("ctrl+n", "expand saved query"),
		),
		key.NewBinding(
			key.WithKeys("ctrl+e"),
			key.WithHelp("ctrl+e", "open in editor"),
		),
		key.NewBinding(
			key.WithKeys("ctrl+l"),
			key.WithHelp("ctrl+l", "clear query"),
		),
	}
}
