package filterbox

import (
	"regexp"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/jxdones/stoat/internal/ui/testutil"
	"github.com/jxdones/stoat/internal/ui/theme"
)

func TestNew(t *testing.T) {
	m := New()
	if m.Value() != "" {
		t.Errorf("New() Value() = %q, want empty", m.Value())
	}
	view := m.View()
	if view.Content == "" {
		t.Error("New() View() is empty")
	}
	plain := testutil.StripANSI(view.Content)
	if regexp.MustCompile("Filter:").FindString(plain) == "" &&
		regexp.MustCompile("filter table rows").FindString(plain) == "" {
		t.Logf("View (plain): %q", plain)
	}
}

func TestFocus_Blur(t *testing.T) {
	tests := []struct {
		name string
		run  func(*Model)
	}{
		{
			name: "after_focus",
			run:  func(m *Model) { m.Focus() },
		},
		{
			name: "after_blur",
			run:  func(m *Model) { m.Blur() },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			tt.run(&m)
			view := m.View()
			if view.Content == "" {
				t.Errorf("View() %s is empty", tt.name)
			}
		})
	}
}

func TestValue_SetValue(t *testing.T) {
	tests := []struct {
		name      string
		setValues []string
		wantValue string
	}{
		{
			name:      "initial_empty",
			setValues: nil,
			wantValue: "",
		},
		{
			name:      "set_filter_text",
			setValues: []string{"NLD"},
			wantValue: "NLD",
		},
		{
			name:      "set_empty_clears",
			setValues: []string{"Dutch", ""},
			wantValue: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			for _, v := range tt.setValues {
				m.SetValue(v)
			}
			if got := m.Value(); got != tt.wantValue {
				t.Errorf("Value() = %q, want %q", got, tt.wantValue)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	prev := theme.Current
	t.Cleanup(func() { theme.Current = prev })
	theme.Current = theme.DefaultTheme()

	tests := []struct {
		name      string
		setup     func(*Model)
		runUpdate func(Model) (Model, tea.Cmd)
		wantValue string
	}{
		{
			name:  "typing_updates_value",
			setup: func(m *Model) { m.Focus() },
			runUpdate: func(m Model) (Model, tea.Cmd) {
				var cmd tea.Cmd
				for _, r := range "NLD" {
					m, cmd = m.Update(testutil.KeyRune(r))
				}
				return m, cmd
			},
			wantValue: "NLD",
		},
		{
			name: "window_size_preserves_value",
			setup: func(m *Model) {
				m.SetValue("Dutch")
			},
			runUpdate: func(m Model) (Model, tea.Cmd) {
				return m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
			},
			wantValue: "Dutch",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			if tt.setup != nil {
				tt.setup(&m)
			}
			m, _ = tt.runUpdate(m)
			if m.Value() != tt.wantValue {
				t.Errorf("after Update Value() = %q, want %q", m.Value(), tt.wantValue)
			}
		})
	}
}

func TestView(t *testing.T) {
	tests := []struct {
		name         string
		setValue     string
		wantContains string
	}{
		{
			name:         "empty_shows_prompt_or_placeholder",
			setValue:     "",
			wantContains: "Filter:",
		},
		{
			name:         "with_value_shows_content",
			setValue:     "NLD",
			wantContains: "NLD",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			if tt.setValue != "" {
				m.SetValue(tt.setValue)
			}
			view := m.View()
			if view.Content == "" {
				t.Fatal("View() is empty")
			}
			plain := testutil.StripANSI(view.Content)
			if plain == "" {
				t.Fatal("View() has no plain text after stripping ANSI")
			}
			if regexp.MustCompile(regexp.QuoteMeta(tt.wantContains)).FindString(plain) == "" {
				if tt.setValue == "" && regexp.MustCompile("filter table rows").FindString(plain) == "" {
					t.Errorf("View() plain should contain %q or placeholder; got: %q", tt.wantContains, plain)
				} else if tt.setValue != "" {
					t.Errorf("View() plain should contain %q; got: %q", tt.wantContains, plain)
				}
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
			name:     "apply_filter_binding",
			wantKey:  "enter",
			wantHelp: "apply filter",
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
