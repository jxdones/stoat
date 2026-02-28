package tabs

import (
	"regexp"
	"testing"

	"github.com/jxdones/stoat/internal/ui/testutil"
	"github.com/jxdones/stoat/internal/ui/theme"
	"github.com/jxdones/stoat/internal/ui/viewstate"
)

func TestNew(t *testing.T) {
	m := New([]string{"Results", "Schema"})
	if m.ActiveTab() != "Results" {
		t.Errorf("New() first tab active: ActiveTab() = %q, want %q", m.ActiveTab(), "Results")
	}
	view := m.View()
	if view.Content == "" {
		t.Error("New() View() is empty")
	}
	plain := testutil.StripANSI(view.Content)
	if regexp.MustCompile("Sections:").FindString(plain) == "" {
		t.Errorf("View() should contain \"Sections:\"; plain: %q", plain)
	}
}

func TestSetActive(t *testing.T) {
	tests := []struct {
		name       string
		tabs       []string
		setIndex   int
		wantActive string
	}{
		{
			name:       "first_tab",
			tabs:       []string{"A", "B"},
			setIndex:   0,
			wantActive: "A",
		},
		{name: "second_tab",
			tabs:       []string{"A", "B"},
			setIndex:   1,
			wantActive: "B",
		},
		{
			name:       "middle_tab",
			tabs:       []string{"A", "B", "C"},
			setIndex:   1,
			wantActive: "B",
		},
		{
			name:       "out_of_range_negative_no_change",
			tabs:       []string{"A", "B"},
			setIndex:   -1,
			wantActive: "A",
		},
		{
			name:       "out_of_range_too_high_no_change",
			tabs:       []string{"A", "B"},
			setIndex:   2,
			wantActive: "A",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(tt.tabs)
			m.SetActive(tt.setIndex)
			if got := m.ActiveTab(); got != tt.wantActive {
				t.Errorf("after SetActive(%d) ActiveTab() = %q, want %q", tt.setIndex, got, tt.wantActive)
			}
		})
	}
}

func TestActiveTab(t *testing.T) {
	tests := []struct {
		name      string
		tabs      []string
		activeIdx int
		wantLabel string
	}{
		{
			name:      "empty_tabs",
			tabs:      nil,
			activeIdx: 0,
			wantLabel: "",
		},
		{
			name:      "first",
			tabs:      []string{"Results", "Schema"},
			activeIdx: 0,
			wantLabel: "Results",
		},
		{
			name:      "second",
			tabs:      []string{"Results", "Schema"},
			activeIdx: 1,
			wantLabel: "Schema",
		},
		{
			name:      "single_tab",
			tabs:      []string{"Only"},
			activeIdx: 0,
			wantLabel: "Only",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(tt.tabs)
			if tt.tabs != nil {
				m.SetActive(tt.activeIdx)
			}
			if got := m.ActiveTab(); got != tt.wantLabel {
				t.Errorf("ActiveTab() = %q, want %q", got, tt.wantLabel)
			}
		})
	}
}

func TestSetSize(t *testing.T) {
	tests := []struct {
		name  string
		width int
	}{
		{
			name:  "small_clamped",
			width: 10,
		},
		{
			name:  "min_width",
			width: 24,
		},
		{
			name:  "large",
			width: 80,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New([]string{"A", "B"})
			m.SetSize(tt.width)
			view := m.View()
			if view.Content == "" {
				t.Errorf("View() after SetSize(%d) is empty", tt.width)
			}
		})
	}
}

func TestSetFocused(t *testing.T) {
	tests := []struct {
		name    string
		focused bool
	}{
		{
			name:    "focused",
			focused: true,
		},
		{
			name:    "blurred",
			focused: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New([]string{"A"})
			m.SetFocused(tt.focused)
			view := m.View()
			if view.Content == "" {
				t.Errorf("View() after SetFocused(%v) is empty", tt.focused)
			}
		})
	}
}

func TestApplyViewState(t *testing.T) {
	prev := theme.Current
	t.Cleanup(func() { theme.Current = prev })
	theme.Current = theme.DefaultTheme()

	m := New([]string{"A", "B"})
	m.ApplyViewState(viewstate.ViewState{Width: 40, Focused: true})
	view := m.View()
	if view.Content == "" {
		t.Error("View() after ApplyViewState is empty")
	}
	plain := testutil.StripANSI(view.Content)
	if regexp.MustCompile("Sections:").FindString(plain) == "" {
		t.Errorf("View() should contain \"Sections:\"; plain: %q", plain)
	}
}

func TestView(t *testing.T) {
	prev := theme.Current
	t.Cleanup(func() { theme.Current = prev })
	theme.Current = theme.DefaultTheme()

	tests := []struct {
		name         string
		tabs         []string
		activeIdx    int
		setWidth     int
		wantContains string // substring in plain view
	}{
		{
			name:         "single_tab",
			tabs:         []string{"Results"},
			activeIdx:    0,
			setWidth:     40,
			wantContains: "1:Results",
		},
		{
			name:         "two_tabs_first_active",
			tabs:         []string{"Results", "Schema"},
			activeIdx:    0,
			setWidth:     50,
			wantContains: "Sections:",
		},
		{
			name:         "two_tabs_second_active",
			tabs:         []string{"Results", "Schema"},
			activeIdx:    1,
			setWidth:     50,
			wantContains: "2:Schema",
		},
		{
			name:         "empty_tabs",
			tabs:         []string{},
			activeIdx:    0,
			setWidth:     40,
			wantContains: "Sections:",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(tt.tabs)
			if len(tt.tabs) > 0 {
				m.SetActive(tt.activeIdx)
			}
			m.SetSize(tt.setWidth)
			view := m.View()
			if view.Content == "" {
				t.Fatal("View() is empty")
			}
			plain := testutil.StripANSI(view.Content)
			if regexp.MustCompile(regexp.QuoteMeta(tt.wantContains)).FindString(plain) == "" {
				t.Errorf("View() plain should contain %q; got: %q", tt.wantContains, plain)
			}
		})
	}
}

func TestHelpBindings(t *testing.T) {
	tests := []struct {
		name     string
		wantKey  string
		wantHelp string
	}{
		{
			name:     "switch_section",
			wantKey:  "ctrl+1-5",
			wantHelp: "switch section",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bindings := HelpBindings()
			if len(bindings) == 0 {
				t.Fatal("HelpBindings() returned empty slice")
			}
			var found bool
			for _, b := range bindings {
				h := b.Help()
				if h.Key == tt.wantKey {
					found = true
					if tt.wantHelp != "" && h.Desc != tt.wantHelp {
						t.Errorf("binding %q Help().Desc = %q, want %q", tt.wantKey, h.Desc, tt.wantHelp)
					}
					break
				}
			}
			if !found {
				t.Errorf("HelpBindings() should include key %q", tt.wantKey)
			}
		})
	}
}
