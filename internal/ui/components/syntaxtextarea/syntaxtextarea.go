package syntaxtextarea

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"

	"github.com/jxdones/stoat/internal/ui/theme"
)

// Model wraps textarea.Model with syntax highlighting.
type Model struct {
	input textarea.Model
}

// New creates a new Model with default textarea settings.
func New() Model {
	return Model{input: textarea.New()}
}

// SetInput replaces the underlying textarea. Use this to apply textarea
// configuration (prompt, placeholder, line numbers, etc.) before use.
func (m *Model) SetInput(ta textarea.Model) {
	m.input = ta
}

// Input returns the underlying textarea model for configuration.
func (m Model) Input() textarea.Model {
	return m.input
}

// Value returns the current text value.
func (m Model) Value() string { return m.input.Value() }

// SetValue sets the text value.
func (m *Model) SetValue(s string) { m.input.SetValue(s) }

// Focus focuses the textarea.
func (m *Model) Focus() tea.Cmd { return m.input.Focus() }

// Blur blurs the textarea.
func (m *Model) Blur() { m.input.Blur() }

// Focused reports whether the textarea is focused.
func (m Model) Focused() bool { return m.input.Focused() }

// LineInfo returns cursor line information from the underlying textarea.
func (m Model) LineInfo() textarea.LineInfo { return m.input.LineInfo() }

// SetWidth sets the textarea width.
func (m *Model) SetWidth(w int) { m.input.SetWidth(w) }

// SetHeight sets the textarea height.
func (m *Model) SetHeight(h int) { m.input.SetHeight(h) }

// SetStyles sets the textarea styles (used for cursor line background, etc.).
func (m *Model) SetStyles(s textarea.Styles) { m.input.SetStyles(s) }

// Styles returns the current textarea styles.
func (m Model) Styles() textarea.Styles { return m.input.Styles() }

// Update delegates all input handling to the underlying textarea.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// segment is a styled piece of text.
type segment struct {
	text  string
	style lipgloss.Style
}

// tokenStyle maps a chroma token type to a lipgloss style using the active theme.
func tokenStyle(t chroma.TokenType) lipgloss.Style {
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

// tokenize runs the chroma SQL lexer over sql and returns styled segments.
func tokenize(sql string) []segment {
	lexer := lexers.Get("sql")
	if lexer == nil {
		lexer = lexers.Fallback
	}
	iter, err := lexer.Tokenise(nil, sql)
	if err != nil {
		return []segment{{text: sql, style: lipgloss.NewStyle().Foreground(theme.Current.TextPrimary)}}
	}
	var segments []segment
	for tok := iter(); tok != chroma.EOF; tok = iter() {
		segments = append(segments, segment{
			text:  tok.Value,
			style: tokenStyle(tok.Type),
		})
	}
	return segments
}

// buildLines converts token segments into rendered line strings, inserting a
// block cursor at (cursorRow, cursorCharCol).
func buildLines(segments []segment, cursorRow, cursorCharCol int) []string {
	cursorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#000000")).
		Background(theme.Current.TextPrimary)

	// lineSegs accumulates styled segments per hard line.
	type lineSeg struct {
		text  string
		style lipgloss.Style
	}
	lines := [][]lineSeg{{}}
	lineIdx := 0
	charPos := 0 // character offset within the current hard line

	for _, seg := range segments {
		remaining := []rune(seg.text)
		for len(remaining) > 0 {
			// Find next newline within remaining runes.
			nl := -1
			for i, r := range remaining {
				if r == '\n' {
					nl = i
					break
				}
			}

			var part []rune
			var hasNewline bool
			if nl == -1 {
				part = remaining
				remaining = nil
			} else {
				part = remaining[:nl]
				remaining = remaining[nl+1:]
				hasNewline = true
			}

			if lineIdx == cursorRow && len(part) > 0 {
				partEnd := charPos + len(part)
				if cursorCharCol >= charPos && cursorCharCol < partEnd {
					// Cursor falls within this segment.
					rel := cursorCharCol - charPos
					if rel > 0 {
						lines[lineIdx] = append(lines[lineIdx], lineSeg{string(part[:rel]), seg.style})
					}
					lines[lineIdx] = append(lines[lineIdx], lineSeg{string(part[rel : rel+1]), cursorStyle})
					if rel+1 < len(part) {
						lines[lineIdx] = append(lines[lineIdx], lineSeg{string(part[rel+1:]), seg.style})
					}
				} else {
					if len(part) > 0 {
						lines[lineIdx] = append(lines[lineIdx], lineSeg{string(part), seg.style})
					}
				}
				charPos = partEnd
			} else {
				if len(part) > 0 {
					lines[lineIdx] = append(lines[lineIdx], lineSeg{string(part), seg.style})
				}
			}

			if hasNewline {
				lineIdx++
				charPos = 0
				lines = append(lines, []lineSeg{})
			}
		}
	}

	// Cursor at end of line (past all tokens on that line).
	if lineIdx == cursorRow && charPos == cursorCharCol {
		lines[lineIdx] = append(lines[lineIdx], lineSeg{" ", cursorStyle})
	}

	// Render each line.
	result := make([]string, len(lines))
	for i, line := range lines {
		var sb strings.Builder
		for _, ls := range line {
			sb.WriteString(ls.style.Render(ls.text))
		}
		result[i] = sb.String()
	}
	return result
}

// View renders the textarea with syntax highlighting.
// Falls back to the native textarea view when the value is empty (shows placeholder).
func (m Model) View() string {
	value := m.input.Value()
	if value == "" {
		return m.input.View()
	}

	prompt := m.input.Prompt
	promptWidth := lipgloss.Width(prompt)
	showLineNumbers := m.input.ShowLineNumbers

	totalLines := strings.Count(value, "\n") + 1
	lineNumDigits := len(strconv.Itoa(totalLines))

	segments := tokenize(value)
	cursorRow := m.input.Line()
	cursorCharCol := m.input.LineInfo().CharOffset

	coloredLines := buildLines(segments, cursorRow, cursorCharCol)

	width := m.input.Width()
	height := m.input.Height()

	promptStyle := lipgloss.NewStyle().Foreground(theme.Current.TextAccent)
	lineNumStyle := lipgloss.NewStyle().Foreground(theme.Current.TextMuted)

	// Respect the textarea's viewport — only show `height` lines starting at scrollOffset.
	scrollOffset := m.input.ScrollYOffset()
	start := min(scrollOffset, len(coloredLines))
	end := min(start+height, len(coloredLines))
	visibleLines := coloredLines[start:end]

	var sb strings.Builder
	for i, line := range visibleLines {
		actualLineIdx := start + i

		// Prompt
		sb.WriteString(promptStyle.Render(prompt))

		// Line number
		lineNumStr := ""
		if showLineNumbers {
			lineNumStr = fmt.Sprintf("  %*d ", lineNumDigits, actualLineIdx+1)
			sb.WriteString(lineNumStyle.Render(lineNumStr))
		}

		// Text content — width minus prompt and line number
		prefixWidth := promptWidth + lipgloss.Width(lineNumStr)
		textWidth := max(1, width-prefixWidth)
		sb.WriteString(lipgloss.NewStyle().Width(textWidth).Render(line))

		if i < len(visibleLines)-1 {
			sb.WriteRune('\n')
		}
	}

	// Pad remaining height with blank lines.
	for i := len(visibleLines); i < height; i++ {
		if i > 0 {
			sb.WriteRune('\n')
		}
		sb.WriteString(promptStyle.Render(prompt))
		if showLineNumbers {
			sb.WriteString(lipgloss.NewStyle().Width(lineNumDigits + 2).Render(""))
		}
	}

	return sb.String()
}
