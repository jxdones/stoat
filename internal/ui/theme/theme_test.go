package theme

import (
	"image/color"
	"testing"
)

// restoreCurrent restores Current to the value it had before the test (for tests that mutate it).
func restoreCurrent(prev Theme) {
	Current = prev
}

func Test_SetNamedTheme(t *testing.T) {
	prev := Current
	t.Cleanup(func() { restoreCurrent(prev) })

	tests := []struct {
		name   string
		input  string
		expect Theme
	}{
		{
			name:   "empty_is_default",
			input:  "",
			expect: DefaultTheme(),
		},
		{
			name:   "default",
			input:  "default",
			expect: DefaultTheme(),
		},
		{
			name:   "dracula",
			input:  "dracula",
			expect: DraculaTheme(),
		},
		{
			name:   "solarized",
			input:  "solarized",
			expect: SolarizedTheme(),
		},
		{
			name:   "case_insensitive",
			input:  "DRAcula",
			expect: DraculaTheme(),
		},
		{
			name:   "trimmed",
			input:  "  solarized  ",
			expect: SolarizedTheme(),
		},
		{
			name:   "case_insensitive",
			input:  "DRAcula",
			expect: DraculaTheme(),
		},
		{
			name:   "trimmed",
			input:  "  solarized  ",
			expect: SolarizedTheme(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th, ok := SetNamedTheme(tt.input)
			if !ok {
				t.Errorf("SetNamedTheme(%q) ok = false, want true", tt.input)
			}
			if !colorsEqual(th.Border, tt.expect.Border) {
				t.Errorf("SetNamedTheme(%q) theme.Border mismatch", tt.input)
			}
			if !colorsEqual(Current.Border, tt.expect.Border) {
				t.Errorf("after SetNamedTheme(%q) Current.Border mismatch", tt.input)
			}
		})
	}
}

func Test_SetNamedTheme_unknown_name_returns_current_and_false(t *testing.T) {
	prev := Current
	t.Cleanup(func() { restoreCurrent(prev) })

	// Set to a known state so we can assert on the returned theme
	SetNamedTheme("dracula")
	before := Current

	th, ok := SetNamedTheme("unknown")
	if ok {
		t.Error("SetNamedTheme(\"unknown\") ok = true, want false")
	}
	if !colorsEqual(th.Border, before.Border) {
		t.Errorf("returned theme.Border should match Current before call")
	}
	if !colorsEqual(Current.Border, before.Border) {
		t.Error("Current was changed after unknown name")
	}
}

// colorsEqual reports whether two color.Colors are equal (same RGBA).
func colorsEqual(a, b color.Color) bool {
	if a == b {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	r1, g1, b1, a1 := a.RGBA()
	r2, g2, b2, a2 := b.RGBA()
	return r1 == r2 && g1 == g2 && b1 == b2 && a1 == a2
}

func Test_builtin_themes_return_distinct_non_empty_colors(t *testing.T) {
	defaultT := DefaultTheme()
	draculaT := DraculaTheme()
	solarizedT := SolarizedTheme()

	// Each theme should have key colors set
	if defaultT.Border == nil || defaultT.TextPrimary == nil {
		t.Error("DefaultTheme() has nil Border or TextPrimary")
	}
	if draculaT.Border == nil || draculaT.TextPrimary == nil {
		t.Error("DraculaTheme() has nil Border or TextPrimary")
	}
	if solarizedT.Border == nil || solarizedT.TextPrimary == nil {
		t.Error("SolarizedTheme() has nil Border or TextPrimary")
	}

	// Built-ins should be distinct so theme switch is visible
	if colorsEqual(defaultT.Border, draculaT.Border) && colorsEqual(defaultT.TextPrimary, draculaT.TextPrimary) {
		t.Error("Default and Dracula themes are identical")
	}
	if colorsEqual(defaultT.Border, solarizedT.Border) && colorsEqual(defaultT.TextPrimary, solarizedT.TextPrimary) {
		t.Error("Default and Solarized themes are identical")
	}
	if colorsEqual(draculaT.Border, solarizedT.Border) && colorsEqual(draculaT.TextPrimary, solarizedT.TextPrimary) {
		t.Error("Dracula and Solarized themes are identical")
	}
}

func Test_DefaultTheme_has_expected_styling_fields_set(t *testing.T) {
	th := DefaultTheme()
	// Spot-check a few fields so we don't rely on every literal; ensures struct is populated
	fields := []color.Color{th.Border, th.TextPrimary, th.TextError, th.TableHeader, th.TabsActiveBg}
	for i, c := range fields {
		if c == nil {
			t.Errorf("DefaultTheme() has nil field at index %d", i)
		}
	}
}

func Test_DraculaTheme_has_expected_styling_fields_set(t *testing.T) {
	th := DraculaTheme()
	fields := []color.Color{th.Border, th.TextPrimary, th.TextError, th.TableHeader}
	for i, c := range fields {
		if c == nil {
			t.Errorf("DraculaTheme() has nil field at index %d", i)
		}
	}
}

func Test_SolarizedTheme_has_expected_styling_fields_set(t *testing.T) {
	th := SolarizedTheme()
	fields := []color.Color{th.Border, th.TextPrimary, th.TextError, th.TableHeader}
	for i, c := range fields {
		if c == nil {
			t.Errorf("SolarizedTheme() has nil field at index %d", i)
		}
	}
}
