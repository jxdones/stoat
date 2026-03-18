// Package syntaxtextarea provides a textarea component with SQL syntax highlighting.
// It wraps bubbles/textarea for all input handling and replaces View() with a
// custom renderer that injects chroma-colored tokens and a block cursor.
package syntaxtextarea

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/rivo/uniseg"

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

// segment is a styled piece of text from the chroma tokenizer.
type segment struct {
	text  string
	style lipgloss.Style
}

// runeStyle pairs a single rune with its syntax style.
type runeStyle struct {
	r     rune
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

// rsDisplayWidth returns the total visual display width of a runeStyle slice.
func rsDisplayWidth(rs []runeStyle) int {
	var w int
	for _, r := range rs {
		w += uniseg.StringWidth(string(r.r))
	}
	return w
}

// wrapRuneStyles soft-wraps rs into visual sub-rows at textWidth using the same
// word-wrap algorithm as the underlying textarea (see textarea.wrap).
func wrapRuneStyles(rs []runeStyle, textWidth int) [][]runeStyle {
	if textWidth <= 0 || len(rs) == 0 {
		return [][]runeStyle{rs}
	}

	var (
		rows   = [][]runeStyle{{}}
		word   []runeStyle
		row    int
		spaces int
	)

	// spaceStyle returns the style to use for flushed space characters.
	spaceStyle := func() lipgloss.Style {
		if len(word) > 0 {
			return word[len(word)-1].style
		}
		if len(rows[row]) > 0 {
			return rows[row][len(rows[row])-1].style
		}
		return lipgloss.NewStyle()
	}

	for _, rr := range rs {
		if unicode.IsSpace(rr.r) {
			spaces++
		} else {
			word = append(word, rr)
		}

		if spaces > 0 {
			ss := spaceStyle()
			if rsDisplayWidth(rows[row])+rsDisplayWidth(word)+spaces > textWidth {
				row++
				rows = append(rows, []runeStyle{})
				rows[row] = append(rows[row], word...)
				for range spaces {
					rows[row] = append(rows[row], runeStyle{' ', ss})
				}
			} else {
				rows[row] = append(rows[row], word...)
				for range spaces {
					rows[row] = append(rows[row], runeStyle{' ', ss})
				}
			}
			spaces = 0
			word = nil
		} else if len(word) > 0 {
			lastW := uniseg.StringWidth(string(word[len(word)-1].r))
			if rsDisplayWidth(word)+lastW > textWidth {
				if len(rows[row]) > 0 {
					row++
					rows = append(rows, []runeStyle{})
				}
				rows[row] = append(rows[row], word...)
				word = nil
			}
		}
	}

	// Final flush with one trailing space — mirrors textarea.wrap().
	ss := spaceStyle()
	if rsDisplayWidth(rows[row])+rsDisplayWidth(word)+spaces >= textWidth {
		row++
		rows = append(rows, []runeStyle{})
		rows[row] = append(rows[row], word...)
		spaces++
		for range spaces {
			rows[row] = append(rows[row], runeStyle{' ', ss})
		}
	} else {
		rows[row] = append(rows[row], word...)
		spaces++
		for range spaces {
			rows[row] = append(rows[row], runeStyle{' ', ss})
		}
	}

	return rows
}

// visualLine is one soft-wrapped display row.
type visualLine struct {
	content  string
	hardLine int // 0-indexed hard line
	subRow   int // 0 = first sub-row of the hard line, >0 = soft-wrap continuation
}

// buildLines converts token segments into soft-wrapped visual lines, injecting a
// block cursor at the position identified by (cursorHardRow, cursorSubRow, cursorColOffset).
//
//   - cursorHardRow   = m.input.Line()                  hard (\n-delimited) line index
//   - cursorSubRow    = m.input.LineInfo().RowOffset     soft-wrapped sub-row within that line
//   - cursorColOffset = m.input.LineInfo().ColumnOffset  rune index within that sub-row
func buildLines(segments []segment, textWidth, cursorHardRow, cursorSubRow, cursorColOffset int) []visualLine {
	cursorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#000000")).
		Background(theme.Current.TextPrimary)

	// Phase 1: split segments into per-hard-line runeStyle slices.
	hardLines := [][]runeStyle{{}}
	lineIdx := 0
	for _, seg := range segments {
		for _, r := range seg.text {
			if r == '\n' {
				lineIdx++
				hardLines = append(hardLines, []runeStyle{})
			} else {
				hardLines[lineIdx] = append(hardLines[lineIdx], runeStyle{r, seg.style})
			}
		}
	}

	// Phase 2: soft-wrap each hard line and render, injecting the cursor.
	var result []visualLine
	for hardIdx, hl := range hardLines {
		subRows := wrapRuneStyles(hl, textWidth)
		isCursorHardLine := hardIdx == cursorHardRow

		for subIdx, subRow := range subRows {
			isCursorRow := isCursorHardLine && subIdx == cursorSubRow

			var sb strings.Builder
			if isCursorRow {
				colPos := 0
				cursorPlaced := false
				for _, rr := range subRow {
					if !cursorPlaced && colPos == cursorColOffset {
						sb.WriteString(cursorStyle.Render(string(rr.r)))
						cursorPlaced = true
					} else {
						sb.WriteString(rr.style.Render(string(rr.r)))
					}
					colPos++
				}
				if !cursorPlaced {
					sb.WriteString(cursorStyle.Render(" "))
				}
			} else {
				for _, rr := range subRow {
					sb.WriteString(rr.style.Render(string(rr.r)))
				}
			}

			result = append(result, visualLine{
				content:  sb.String(),
				hardLine: hardIdx,
				subRow:   subIdx,
			})
		}
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

	// m.input.Width() is the text-only width: the underlying textarea already
	// subtracts prompt width, line number width, and border from the set width
	// in SetWidth(). Do NOT subtract prefixes again here.
	textWidth := max(1, m.input.Width())

	prompt := m.input.Prompt
	showLineNumbers := m.input.ShowLineNumbers

	// Line number column width matches what SetWidth reserved:
	// numDigits(MaxHeight) + gap(2) = e.g. 2 + 2 = 4 for MaxHeight=99.
	lineNumDigits := len(strconv.Itoa(m.input.MaxHeight))

	li := m.input.LineInfo()
	segments := tokenize(value)
	visualLines := buildLines(segments, textWidth, m.input.Line(), li.RowOffset, li.ColumnOffset)

	height := m.input.Height()
	scrollOffset := m.input.ScrollYOffset()
	start := min(scrollOffset, len(visualLines))
	end := min(start+height, len(visualLines))
	visible := visualLines[start:end]

	promptStyle := lipgloss.NewStyle().Foreground(theme.Current.TextAccent)
	lineNumStyle := lipgloss.NewStyle().Foreground(theme.Current.TextMuted)

	var sb strings.Builder
	for i, vl := range visible {
		sb.WriteString(promptStyle.Render(prompt))

		if showLineNumbers {
			var lineNumStr string
			if vl.subRow == 0 {
				lineNumStr = fmt.Sprintf(" %*d ", lineNumDigits, vl.hardLine+1)
			} else {
				lineNumStr = fmt.Sprintf(" %*s ", lineNumDigits, "")
			}
			sb.WriteString(lineNumStyle.Render(lineNumStr))
		}

		sb.WriteString(lipgloss.NewStyle().Width(textWidth).Render(vl.content))

		if i < len(visible)-1 {
			sb.WriteRune('\n')
		}
	}

	// Pad remaining height with blank lines.
	for i := len(visible); i < height; i++ {
		if i > 0 {
			sb.WriteRune('\n')
		}
		sb.WriteString(promptStyle.Render(prompt))
		if showLineNumbers {
			sb.WriteString(lineNumStyle.Render(fmt.Sprintf(" %*s ", lineNumDigits, "")))
		}
	}

	return sb.String()
}
