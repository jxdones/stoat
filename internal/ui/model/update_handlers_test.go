package model

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// ellipsis is the same character used in queryPreviewForHeader for truncation.
const ellipsis = "…"

func TestQueryPreviewForHeader(t *testing.T) {
	const maxLen = 52

	tests := []struct {
		name  string
		query string
		want  string
	}{
		{
			name:  "empty_returns_empty",
			query: "",
			want:  "",
		},
		{
			name:  "whitespace_only_returns_empty",
			query: "   \n\t  ",
			want:  "",
		},
		{
			name:  "short_query_unchanged",
			query: "SELECT 1",
			want:  "SELECT 1",
		},
		{
			name:  "single_word_unchanged",
			query: "SELECT",
			want:  "SELECT",
		},
		{
			name:  "newlines_collapsed_to_single_space",
			query: "SELECT *\nFROM users\nWHERE id = 1",
			want:  "SELECT * FROM users WHERE id = 1",
		},
		{
			name:  "multiple_spaces_collapsed",
			query: "SELECT   *   FROM   users",
			want:  "SELECT * FROM users",
		},
		{
			name:  "leading_and_trailing_space_trimmed",
			query: "  SELECT * FROM users  ",
			want:  "SELECT * FROM users",
		},
		{
			name:  "exactly_52_chars_not_truncated",
			query: strings.Repeat("x", maxLen),
			want:  strings.Repeat("x", maxLen),
		},
		{
			name:  "53_chars_truncated_with_ellipsis",
			query: strings.Repeat("a", 53),
			want:  strings.Repeat("a", maxLen-1) + ellipsis,
		},
		{
			name:  "long_query_truncated",
			query: "SELECT id, name, email FROM users WHERE active = 1 ORDER BY created_at DESC LIMIT 100",
			want:  "SELECT id, name, email FROM users WHERE active = 1 " + ellipsis,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := queryPreviewForHeader(tt.query)
			if got != tt.want {
				t.Errorf("queryPreviewForHeader() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHelpExpanded(t *testing.T) {
	tests := []struct {
		name              string
		msg               tea.KeyPressMsg
		initiallyExpanded bool
		wantExpanded      bool
	}{
		{
			name:              "toggle_help_expanded_on",
			initiallyExpanded: false,
			msg:               keyMsg("?"),
			wantExpanded:      true,
		},
		{
			name:              "toggle_help_expanded_off",
			initiallyExpanded: true,
			msg:               keyMsg("esc"),
			wantExpanded:      false,
		},
		{
			name:              "toggle_help_expanded_on_when_already_expanded",
			initiallyExpanded: true,
			msg:               keyMsg("?"),
			wantExpanded:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			m.helpExpanded = tt.initiallyExpanded
			result, _ := m.handleKeyPress(tt.msg)
			got := result.(Model)
			if got.helpExpanded != tt.wantExpanded {
				t.Errorf("helpExpanded = %v, want %v", m.helpExpanded, tt.wantExpanded)
			}
		})
	}
}
