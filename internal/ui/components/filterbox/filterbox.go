package filterbox

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/jxdones/stoat/internal/ui/theme"
)

// Model represents a filter box with an text input.
type Model struct {
	input textinput.Model
	width int
}

// New creates a new filter box model with the default configuration.
func New() Model {
	ti := textinput.New()
	ti.Placeholder = "e.g. NLD or Dutch"
	ti.CharLimit = 512
	ti.SetWidth(50)
	styles := ti.Styles()
	styles.Focused.Prompt = styles.Focused.Prompt.Foreground(theme.Current.TextAccent)
	styles.Blurred.Prompt = styles.Blurred.Prompt.Foreground(theme.Current.TextMuted)
	ti.SetStyles(styles)

	return Model{
		input: ti,
		width: 50,
	}
}

// Focus sets the focus state of the filter box to true.
func (m *Model) Focus() {
	m.input.Focus()
}

// Blur sets the focus state of the filter box to false.
func (m *Model) Blur() {
	m.input.Blur()
}

// Value returns the current value of the filter box.
func (m *Model) Value() string {
	return m.input.Value()
}

// SetValue sets the value of the filter box.
func (m *Model) SetValue(value string) {
	m.input.SetValue(value)
}

// Update handles key messages and updates the model state.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// View renders the filter box with the current state.
func (m Model) View() tea.View {
	return tea.NewView(m.input.View())
}

// HelpBindings returns the key bindings for the filter box.
func HelpBindings() []key.Binding {
	return []key.Binding{
		key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "apply filter"),
		),
	}
}
