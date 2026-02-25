package theme

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
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
			if th.Border != tt.expect.Border {
				t.Errorf("SetNamedTheme(%q) theme.Border = %q, want %q", tt.input, th.Border, tt.expect.Border)
			}
			if Current.Border != tt.expect.Border {
				t.Errorf("after SetNamedTheme(%q) Current.Border = %q, want %q", tt.input, Current.Border, tt.expect.Border)
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
	if th.Border != before.Border {
		t.Errorf("returned theme.Border = %q, want Current before call %q", th.Border, before.Border)
	}
	if Current.Border != before.Border {
		t.Error("Current was changed after unknown name")
	}
}

func Test_builtin_themes_return_distinct_non_empty_colors(t *testing.T) {
	defaultT := DefaultTheme()
	draculaT := DraculaTheme()
	solarizedT := SolarizedTheme()

	// Each theme should have key colors set (lipgloss.Color is a string)
	if string(defaultT.Border) == "" || string(defaultT.TextPrimary) == "" {
		t.Error("DefaultTheme() has empty Border or TextPrimary")
	}
	if string(draculaT.Border) == "" || string(draculaT.TextPrimary) == "" {
		t.Error("DraculaTheme() has empty Border or TextPrimary")
	}
	if string(solarizedT.Border) == "" || string(solarizedT.TextPrimary) == "" {
		t.Error("SolarizedTheme() has empty Border or TextPrimary")
	}

	// Built-ins should be distinct so theme switch is visible
	if defaultT.Border == draculaT.Border && defaultT.TextPrimary == draculaT.TextPrimary {
		t.Error("Default and Dracula themes are identical")
	}
	if defaultT.Border == solarizedT.Border && defaultT.TextPrimary == solarizedT.TextPrimary {
		t.Error("Default and Solarized themes are identical")
	}
	if draculaT.Border == solarizedT.Border && draculaT.TextPrimary == solarizedT.TextPrimary {
		t.Error("Dracula and Solarized themes are identical")
	}
}

func Test_DefaultTheme_has_expected_styling_fields_set(t *testing.T) {
	th := DefaultTheme()
	// Spot-check a few fields so we don't rely on every literal; ensures struct is populated
	fields := []lipgloss.Color{th.Border, th.TextPrimary, th.TextError, th.TableHeader, th.TabsActiveBg}
	for i, c := range fields {
		if string(c) == "" {
			t.Errorf("DefaultTheme() has empty field at index %d", i)
		}
	}
}

func Test_DraculaTheme_has_expected_styling_fields_set(t *testing.T) {
	th := DraculaTheme()
	fields := []lipgloss.Color{th.Border, th.TextPrimary, th.TextError, th.TableHeader}
	for i, c := range fields {
		if string(c) == "" {
			t.Errorf("DraculaTheme() has empty field at index %d", i)
		}
	}
}

func Test_SolarizedTheme_has_expected_styling_fields_set(t *testing.T) {
	th := SolarizedTheme()
	fields := []lipgloss.Color{th.Border, th.TextPrimary, th.TextError, th.TableHeader}
	for i, c := range fields {
		if string(c) == "" {
			t.Errorf("SolarizedTheme() has empty field at index %d", i)
		}
	}
}
