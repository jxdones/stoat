package syntaxtextarea

import (
	"regexp"
	"strings"
	"testing"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/jxdones/stoat/internal/ui/testutil"
	"github.com/jxdones/stoat/internal/ui/theme"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func joinSegmentText(segments []segment) string {
	var sb strings.Builder
	for _, s := range segments {
		sb.WriteString(s.text)
	}
	return sb.String()
}

func newTextareaForTest() textarea.Model {
	ta := textarea.New()
	ta.Prompt = "sql> "
	ta.Placeholder = "Enter your query here..."
	ta.ShowLineNumbers = true
	ta.SetWidth(40)
	ta.SetHeight(3)
	ta.Focus()
	return ta
}

func TestTokenize(t *testing.T) {
	prev := theme.Current
	t.Cleanup(func() { theme.Current = prev })
	theme.Current = theme.DefaultTheme()

	tests := []struct {
		name      string
		input     string
		wantParts bool
	}{
		{
			name:      "empty",
			input:     "",
			wantParts: false,
		},
		{
			name:      "simple_select",
			input:     "SELECT 1",
			wantParts: true,
		},
		{
			name:      "multiline_with_comment_and_string",
			input:     "SELECT 'x' -- comment\nFROM users\nWHERE id = 42;",
			wantParts: true,
		},
		{
			name:      "non_sql_text_preserved",
			input:     "hello @@@ not-sql ???",
			wantParts: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tokenize(tt.input)
			if gotText := joinSegmentText(got); gotText != tt.input {
				t.Fatalf("tokenize() text = %q, want %q", gotText, tt.input)
			}
			if tt.wantParts && len(got) == 0 {
				t.Fatal("tokenize() returned no segments for non-empty input")
			}
		})
	}
}

func TestBuildLines(t *testing.T) {
	prev := theme.Current
	t.Cleanup(func() { theme.Current = prev })
	theme.Current = theme.DefaultTheme()

	base := lipgloss.NewStyle().Foreground(theme.Current.TextPrimary)
	alt := lipgloss.NewStyle().Foreground(theme.Current.SyntaxKeyword)

	tests := []struct {
		name               string
		segments           []segment
		cursorRow          int
		cursorCharCol      int
		wantPlainLines     []string
		wantLineCount      int
		wantANSIOnLine     int
		wantANSIOnLineOnly bool
	}{
		{
			name: "cursor_inside_single_segment",
			segments: []segment{
				{text: "SELECT", style: base},
			},
			cursorRow:      0,
			cursorCharCol:  2,
			wantPlainLines: []string{"SELECT"},
			wantLineCount:  1,
			wantANSIOnLine: 0,
		},
		{
			name: "cursor_at_end_of_line_appends_block_space",
			segments: []segment{
				{text: "ab", style: base},
			},
			cursorRow:      0,
			cursorCharCol:  2,
			wantPlainLines: []string{"ab "},
			wantLineCount:  1,
			wantANSIOnLine: 0,
		},
		{
			name: "multiline_cursor_on_second_line",
			segments: []segment{
				{text: "a\nbc", style: base},
			},
			cursorRow:      1,
			cursorCharCol:  1,
			wantPlainLines: []string{"a", "bc"},
			wantLineCount:  2,
			wantANSIOnLine: 1,
		},
		{
			name: "cursor_on_trailing_empty_line",
			segments: []segment{
				{text: "a\n", style: base},
			},
			cursorRow:      1,
			cursorCharCol:  0,
			wantPlainLines: []string{"a", " "},
			wantLineCount:  2,
			wantANSIOnLine: 1,
		},
		{
			name: "cursor_at_segment_boundary",
			segments: []segment{
				{text: "ab", style: base},
				{text: "cd", style: alt},
			},
			cursorRow:      0,
			cursorCharCol:  2,
			wantPlainLines: []string{"abcd"},
			wantLineCount:  1,
			wantANSIOnLine: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildLines(tt.segments, tt.cursorRow, tt.cursorCharCol)
			if len(got) != tt.wantLineCount {
				t.Fatalf("buildLines() line count = %d, want %d", len(got), tt.wantLineCount)
			}

			for i, line := range got {
				plain := testutil.StripANSI(line)
				if i >= len(tt.wantPlainLines) {
					t.Fatalf("unexpected extra line %d: %q", i, plain)
				}
				if plain != tt.wantPlainLines[i] {
					t.Fatalf("line %d plain = %q, want %q", i, plain, tt.wantPlainLines[i])
				}
			}

			if tt.wantANSIOnLine >= 0 && tt.wantANSIOnLine < len(got) {
				if !ansiRE.MatchString(got[tt.wantANSIOnLine]) {
					t.Fatalf("expected ANSI styling on line %d, got %q", tt.wantANSIOnLine, got[tt.wantANSIOnLine])
				}
			}
		})
	}
}

func TestView(t *testing.T) {
	prev := theme.Current
	t.Cleanup(func() { theme.Current = prev })
	theme.Current = theme.DefaultTheme()

	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "empty_value_falls_back_to_native_textarea_view",
			run: func(t *testing.T) {
				ta := newTextareaForTest()
				expected := ta.View()

				m := New()
				m.SetInput(ta)

				if got := m.View(); got != expected {
					t.Fatalf("View() fallback mismatch\n got: %q\nwant: %q", got, expected)
				}
			},
		},
		{
			name: "non_empty_view_shows_prompt_line_numbers_and_content",
			run: func(t *testing.T) {
				ta := newTextareaForTest()
				ta.SetValue("SELECT 1")

				m := New()
				m.SetInput(ta)

				got := testutil.StripANSI(m.View())
				for _, want := range []string{"sql>", "1", "SELECT 1"} {
					if !strings.Contains(got, want) {
						t.Fatalf("View() plain should contain %q, got %q", want, got)
					}
				}
			},
		},
		{
			name: "view_respects_viewport_scroll_offset",
			run: func(t *testing.T) {
				ta := newTextareaForTest()
				ta.SetHeight(2)
				ta.SetValue("l1\nl2\nl3\nl4")

				m := New()
				m.SetInput(ta)

				// Move the cursor down to push viewport to the bottom.
				for range 3 {
					var cmd tea.Cmd
					m, cmd = m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
					_ = cmd
				}

				got := testutil.StripANSI(m.View())
				if strings.Contains(got, "l1") {
					t.Fatalf("expected scrolled view to exclude top line; got %q", got)
				}
				if !strings.Contains(got, "l3") || !strings.Contains(got, "l4") {
					t.Fatalf("expected scrolled view to contain lower lines; got %q", got)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}
