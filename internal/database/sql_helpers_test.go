package database

import (
	"testing"
)

func TestAsString_nil_returns_NullSentinel(t *testing.T) {
	if got := AsString(nil); got != NullSentinel {
		t.Errorf("AsString(nil) = %q, want NullSentinel %q", got, NullSentinel)
	}
}

func TestColumnMinWidth(t *testing.T) {
	tests := []struct {
		name      string
		headerLen int
		want      int
	}{
		{
			name:      "header_based_width",
			headerLen: len("started_at"),
			want:      12, // max(8, min(24, 10+2))
		},
		{
			name:      "short_header_uses_notes",
			headerLen: len("notes"),
			want:      8, // max(8, min(24, 5+2))
		},
		{
			name:      "short_header_clamped_to_min",
			headerLen: 2,
			want:      MinColumnWidth,
		},
		{
			name:      "long_header_clamped_to_max",
			headerLen: 30,
			want:      MaxColumnWidth,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ColumnMinWidth(tt.headerLen)
			if got != tt.want {
				t.Errorf("ColumnMinWidth(%d) = %d, want %d", tt.headerLen, got, tt.want)
			}
		})
	}
}

func TestFirstSQLKeyword(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  string
	}{
		{
			name:  "simple_select",
			query: "select * from users",
			want:  "SELECT",
		},
		{
			name: "leading_whitespace_and_line_comments",
			query: `

-- explain query
   -- another comment
   insert into users(id) values (1)
`,
			want: "INSERT",
		},
		{
			name:  "single_line_block_comment_before_query",
			query: "/* comment */ delete from sessions where id = 1",
			want:  "DELETE",
		},
		{
			name: "multiline_block_comment_before_query",
			query: `
/* comment start
still in comment
comment end */
update users set active = true
`,
			want: "UPDATE",
		},
		{
			name: "multiline_block_comment_closes_mid_line",
			query: `
/* metadata
version: 1 */ select 1
`,
			want: "SELECT",
		},
		{
			name: "empty_or_comment_only_returns_empty",
			query: `
-- only comments
/* and block comments */
`,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FirstSQLKeyword(tt.query)
			if got != tt.want {
				t.Errorf("FirstSQLKeyword() = %q, want %q", got, tt.want)
			}
		})
	}
}
