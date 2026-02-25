package common

import (
	"testing"

	"github.com/jxdones/stoat/internal/ui/theme"
)

func TestClampMin(t *testing.T) {
	tests := []struct {
		name  string
		value int
		min   int
		want  int
	}{
		{
			name:  "width_large_terminal_above_min",
			value: 80,
			min:   24,
			want:  80,
		},
		{
			name:  "width_exactly_min",
			value: 24,
			min:   24,
			want:  24,
		},
		{
			name:  "width_small_terminal_clamped_to_min",
			value: 10,
			min:   24,
			want:  24,
		},
		{
			name:  "width_zero_clamped_to_min",
			value: 0,
			min:   24,
			want:  24,
		},
		{
			name:  "height_large_above_min",
			value: 25,
			min:   1,
			want:  25,
		},
		{
			name:  "height_exactly_min",
			value: 1,
			min:   1,
			want:  1,
		},
		{
			name:  "height_zero_clamped_to_min",
			value: 0,
			min:   1,
			want:  1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClampMin(tt.value, tt.min)
			if got != tt.want {
				t.Errorf("ClampMin(%d, %d) = %d, want %d", tt.value, tt.min, got, tt.want)
			}
		})
	}
}

func TestFocusBorder(t *testing.T) {
	prev := theme.Current
	t.Cleanup(func() { theme.Current = prev })

	theme.Current = theme.DraculaTheme()
	focusedColor := FocusBorder(true)
	unfocusedColor := FocusBorder(false)
	if focusedColor != theme.Current.BorderFocused {
		t.Errorf("FocusBorder(true) = %v, want BorderFocused %v", focusedColor, theme.Current.BorderFocused)
	}
	if unfocusedColor != theme.Current.Border {
		t.Errorf("FocusBorder(false) = %v, want Border %v", unfocusedColor, theme.Current.Border)
	}
}

func TestBoxInnerWidth(t *testing.T) {
	tests := []struct {
		name  string
		outer int
		want  int
	}{
		{
			name:  "outer_zero_clamped_to_min",
			outer: 0,
			want:  1,
		},
		{
			name:  "outer_4_clamped_to_min",
			outer: 4,
			want:  1,
		},
		{
			name:  "outer_5_exactly_min",
			outer: 5,
			want:  1,
		},
		{
			name:  "outer_6_inner_2",
			outer: 6,
			want:  2,
		},
		{
			name:  "outer_10_inner_6",
			outer: 10,
			want:  6,
		},
		{
			name:  "outer_80_inner_76",
			outer: 80,
			want:  76,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BoxInnerWidth(tt.outer)
			if got != tt.want {
				t.Errorf("BoxInnerWidth(%d) = %d, want %d", tt.outer, got, tt.want)
			}
		})
	}
}
