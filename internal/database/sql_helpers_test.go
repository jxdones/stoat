package database

import (
	"testing"
)

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
