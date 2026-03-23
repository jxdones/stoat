package celldetail

import (
	"encoding/json"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/jxdones/stoat/internal/ui/modal"
	"github.com/jxdones/stoat/internal/ui/theme"
)

// modalVerticalChrome is the number of rows consumed by the modal frame:
// top border + title + blank + blank + footer + bottom border.
const modalVerticalChrome = 6

// Event represents an event that can occur in the cell detail modal.
type Event int

const (
	EventNone Event = iota
	EventClosed
)

// Model represents the cell detail modal.
type Model struct {
	viewport   viewport.Model
	colKey     string
	colType    string
	rawContent string
}

// New creates a new cell detail modal.
func New() Model {
	vp := viewport.New()
	vp.FillHeight = true
	return Model{
		viewport: vp,
	}
}

// SetSize sets the size of the cell detail modal.
func (m *Model) SetSize(width, height int) {
	m.viewport.SetWidth(width - modal.BorderAndPadding)
	m.viewport.SetHeight(max(1, height-modalVerticalChrome))
}

// SetContent sets the content of the cell detail modal.
func (m Model) SetContent(colKey, colType, content string) Model {
	m.colKey = colKey
	m.colType = colType
	m.rawContent = content
	if isJSONColumnType(colType) {
		m.viewport.LeftGutterFunc = func(viewport.GutterContext) string { return "  " }
	} else {
		m.viewport.LeftGutterFunc = viewport.NoGutter
	}
	m.viewport.SetContent(displayContent(m.rawContent, m.colType))
	m.viewport.GotoTop()
	return m
}

// PreferredHeight returns the ideal total modal height for the current content,
// capped at maximum. Shrinks to fit short content rather than always using the max.
func (m Model) PreferredHeight(maximum int) int {
	lines := strings.Count(m.rawContent, "\n") + 1
	if isJSONColumnType(m.colType) {
		var v any
		if err := json.Unmarshal([]byte(m.rawContent), &v); err == nil {
			if formatted, err := json.MarshalIndent(v, "", " "); err == nil {
				lines = strings.Count(string(formatted), "\n") + 1
			}
		}
	}
	return min(maximum, modalVerticalChrome+lines)
}

// Update handles key and other messages, forwarding them to the underlying viewport.
func (m Model) Update(msg tea.Msg) (Model, Event) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, EventNone
	}

	switch keyMsg.String() {
	case "esc":
		return m, EventClosed
	default:
		viewport, _ := m.viewport.Update(msg)
		m.viewport = viewport
		return m, EventNone
	}
}

// View renders the cell detail modal.
func (m Model) View() tea.View {
	content := m.viewport.View()
	return tea.NewView(modal.Render(
		fmt.Sprintf("Viewing: %s", m.colKey),
		content,
		"esc close",
		m.viewport.Width()+modal.BorderAndPadding,
	))
}

// displayContent formats the content for display based on the column type.
func displayContent(content, colType string) string {
	if isJSONColumnType(colType) {
		var v any
		if err := json.Unmarshal([]byte(content), &v); err != nil {
			return highlightJSON(content)
		}
		formatted, err := json.MarshalIndent(v, "", " ")
		if err != nil {
			return highlightJSON(content)
		}
		return highlightJSON(string(formatted))
	}
	return content
}

// isJSONColumnType reports whether the column type is JSON.
func isJSONColumnType(colType string) bool {
	colType = strings.ToLower(strings.TrimSpace(colType))
	return colType == "json" || colType == "jsonb"
}

// highlightJSON highlights the JSON content.
func highlightJSON(content string) string {
	lexer := lexers.Get("json")
	if lexer == nil {
		lexer = lexers.Fallback
	}

	iter, err := lexer.Tokenise(nil, content)
	if err != nil {
		return content
	}

	var sb strings.Builder
	for tok := iter(); tok != chroma.EOF; tok = iter() {
		sb.WriteString(jsonTokenStyle(tok.Type).Render(tok.Value))
	}
	return sb.String()
}

// jsonTokenStyle returns the style for a JSON token.
func jsonTokenStyle(t chroma.TokenType) lipgloss.Style {
	switch {
	case t.InCategory(chroma.Keyword):
		return lipgloss.NewStyle().Foreground(theme.Current.SyntaxKeyword)
	case t.InCategory(chroma.LiteralString):
		return lipgloss.NewStyle().Foreground(theme.Current.SyntaxString)
	case t.InCategory(chroma.LiteralNumber):
		return lipgloss.NewStyle().Foreground(theme.Current.SyntaxNumber)
	case t.InCategory(chroma.Comment):
		return lipgloss.NewStyle().Foreground(theme.Current.SyntaxComment)
	case t.InCategory(chroma.Operator), t.InCategory(chroma.Punctuation):
		return lipgloss.NewStyle().Foreground(theme.Current.SyntaxOperator)
	default:
		return lipgloss.NewStyle().Foreground(theme.Current.TextPrimary)
	}
}
