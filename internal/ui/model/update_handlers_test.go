package model

import (
	"strings"
	"testing"
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
			name:  "empty returns empty",
			query: "",
			want:  "",
		},
		{
			name:  "whitespace only returns empty",
			query: "   \n\t  ",
			want:  "",
		},
		{
			name:  "short query unchanged",
			query: "SELECT 1",
			want:  "SELECT 1",
		},
		{
			name:  "single word unchanged",
			query: "SELECT",
			want:  "SELECT",
		},
		{
			name:  "newlines collapsed to single space",
			query: "SELECT *\nFROM users\nWHERE id = 1",
			want:  "SELECT * FROM users WHERE id = 1",
		},
		{
			name:  "multiple spaces collapsed",
			query: "SELECT   *   FROM   users",
			want:  "SELECT * FROM users",
		},
		{
			name:  "leading and trailing space trimmed",
			query: "  SELECT * FROM users  ",
			want:  "SELECT * FROM users",
		},
		{
			name:  "exactly 52 chars not truncated",
			query: strings.Repeat("x", maxLen),
			want:  strings.Repeat("x", maxLen),
		},
		{
			name:  "53 chars truncated with ellipsis",
			query: strings.Repeat("a", 53),
			want:  strings.Repeat("a", maxLen-1) + ellipsis,
		},
		{
			name:  "long query truncated",
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
