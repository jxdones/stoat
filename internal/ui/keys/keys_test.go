package keys

import (
	"testing"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
)

func keyMsg(s string) tea.KeyPressMsg {
	var code rune
	var text string
	switch s {
	case "up":
		code = tea.KeyUp
	case "down":
		code = tea.KeyDown
	case "left":
		code = tea.KeyLeft
	case "right":
		code = tea.KeyRight
	default:
		if len(s) > 0 {
			code, _ = utf8.DecodeRuneInString(s)
			text = s
		}
	}
	return tea.KeyPressMsg(tea.Key{Code: code, Text: text})
}

func TestIsDigitKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want bool
	}{
		{
			name: "digit_0",
			key:  "0",
			want: true,
		},
		{
			name: "digit_1",
			key:  "1",
			want: true,
		},
		{
			name: "digit_5",
			key:  "5",
			want: true,
		},
		{
			name: "digit_9",
			key:  "9",
			want: true,
		},
		{
			name: "letter_a",
			key:  "a",
			want: false,
		},
		{
			name: "letter_g",
			key:  "g",
			want: false,
		},
		{
			name: "letter_G",
			key:  "G",
			want: false,
		},
		{
			name: "space",
			key:  " ",
			want: false,
		},
		{
			name: "multi_char",
			key:  "ab",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsDigitKey(keyMsg(tt.key))
			if got != tt.want {
				t.Errorf("IsDigitKey(keyMsg(%q)) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestParseBufferCount(t *testing.T) {
	tests := []struct {
		name   string
		buffer string
		want   int
	}{
		{
			name:   "empty_defaults_to_one",
			buffer: "",
			want:   1,
		},
		{
			name:   "one",
			buffer: "1",
			want:   1,
		},
		{
			name:   "nine",
			buffer: "9",
			want:   9,
		},
		{
			name:   "forty_two",
			buffer: "42",
			want:   42,
		},
		{
			name:   "large",
			buffer: "999",
			want:   999,
		},
		{
			name:   "zero_treated_as_invalid_returns_one",
			buffer: "0",
			want:   1,
		},
		{
			name:   "invalid_returns_one",
			buffer: "abc",
			want:   1,
		},
		{
			name:   "negative_returns_one",
			buffer: "-5",
			want:   1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseBufferCount(tt.buffer)
			if got != tt.want {
				t.Errorf("ParseBufferCount(%q) = %d, want %d", tt.buffer, got, tt.want)
			}
		})
	}
}
