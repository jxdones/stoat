package celldetail

import (
	"strings"
	"testing"
)

func TestPreferredHeight(t *testing.T) {
	tests := []struct {
		name    string
		colType string
		content string
		max     int
		want    int
	}{
		{
			name:    "single line text fits below max",
			colType: "text",
			content: "Clean Coder",
			max:     20,
			want:    modalVerticalChrome + 1,
		},
		{
			name:    "multi-line text",
			colType: "text",
			content: "line1\nline2\nline3",
			max:     20,
			want:    modalVerticalChrome + 3,
		},
		{
			name:    "capped at maximum",
			colType: "text",
			content: "line1\nline2\nline3\nline4\nline5",
			max:     8,
			want:    8,
		},
		{
			name:    "valid JSON uses formatted line count",
			colType: "jsonb",
			content: `{"publisher":"O'Reilly","tags":["distributed systems","patterns"]}`,
			max:     30,
			want:    modalVerticalChrome + 7, // { "publisher" "tags": [ el el ] }
		},
		{
			name:    "invalid JSON falls back to raw line count",
			colType: "jsonb",
			content: "not valid json",
			max:     20,
			want:    modalVerticalChrome + 1,
		},
		{
			name:    "json column type also matches",
			colType: "json",
			content: `{"key":"value"}`,
			max:     30,
			want:    modalVerticalChrome + 3, // { "key" }
		},
		{
			name:    "max equals preferred returns max",
			colType: "text",
			content: "hello",
			max:     modalVerticalChrome + 1,
			want:    modalVerticalChrome + 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			m = m.SetContent("col", tt.colType, tt.content)
			got := m.PreferredHeight(tt.max)
			if got != tt.want {
				t.Errorf("PreferredHeight(%d) = %d, want %d", tt.max, got, tt.want)
			}
		})
	}
}

// TestModalHeightConsistency is a regression test for the lipgloss Width bug:
// long ANSI-colored JSON content was word-wrapping inside the modal border,
// producing a different number of lines than short content at the same SetSize height.
func TestModalHeightConsistency(t *testing.T) {
	tests := []struct {
		name    string
		colType string
		content string
	}{
		{
			name:    "short text",
			colType: "text",
			content: "Clean Coder",
		},
		{
			name:    "long compact JSON",
			colType: "jsonb",
			content: `{"id":1,"name":"Alice Smith","email":"alice@example.com","bio":"A software engineer with over 10 years of experience in distributed systems.","tags":["go","rust","postgres","redis"],"score":9.5}`,
		},
		{
			name:    "small JSON object",
			colType: "jsonb",
			content: `{"key":"value"}`,
		},
	}

	const width, height = 53, 12
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			m.SetSize(width, height)
			m = m.SetContent("col", tt.colType, tt.content)

			got := len(strings.Split(m.View().Content, "\n"))
			if got != height {
				t.Errorf("View() produced %d lines, want %d", got, height)
			}
		})
	}
}
