package editbox

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/jxdones/stoat/internal/ui/theme"
)

// Model wraps a text input for inline cell editing.
type Model struct {
	input textinput.Model
}

// New creates a new editbox model with the default configuration.
func New() Model {
	ti := textinput.New()
	ti.CharLimit = 0
	styles := ti.Styles()
	styles.Focused.Prompt = styles.Focused.Prompt.Foreground(theme.Current.TextAccent)
	ti.SetStyles(styles)
	return Model{input: ti}
}

// Focus sets focus on the input.
func (m *Model) Focus() {
	m.input.Focus()
}

// Blur removes focus from the input.
func (m *Model) Blur() {
	m.input.Blur()
}

// Value returns the current input value.
func (m *Model) Value() string {
	return m.input.Value()
}

// SetValue sets the input value and moves the cursor to the end.
func (m *Model) SetValue(v string) {
	m.input.SetValue(v)
	m.input.CursorEnd()
}

// SetWidth sets the visible width of the input.
func (m *Model) SetWidth(w int) {
	m.input.SetWidth(w)
}

// Update handles key and other messages, forwarding them to the underlying input.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// View renders the input.
func (m Model) View() tea.View {
	return tea.NewView(m.input.View())
}

// HelpBindings returns the key bindings shown in the help bar while editing.
func HelpBindings() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
		key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
	}
}
