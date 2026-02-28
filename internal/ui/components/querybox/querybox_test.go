package querybox

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
	if plain != "" && regexp.MustCompile(`sql>\s*`).FindString(plain) == "" {
		if regexp.MustCompile("Enter your query").FindString(plain) == "" {
			t.Logf("View (plain): %q", plain)
		}
	}
}

func TestSetSize_clamps_dimensions(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
	}{
		{
			name:   "small_clamped_to_min",
			width:  10,
			height: 1,
		},
		{
			name:   "large_no_panic",
			width:  200,
			height: 50,
		},
		{
			name:   "exact_min",
			width:  24,
			height: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			m.SetSize(tt.width, tt.height)
			view := m.View()
			if view.Content == "" {
				t.Errorf("View() after SetSize(%d,%d) is empty", tt.width, tt.height)
			}
		})
	}
}

func TestFocus_Blur_SetFocused(t *testing.T) {
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
		{
			name: "after_setFocused_true",
			run:  func(m *Model) { m.SetFocused(true) },
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
		setValues []string // applied in order; empty slice = no SetValue calls
		wantValue string
	}{
		{
			name:      "initial_empty",
			setValues: nil,
			wantValue: "",
		},
		{
			name:      "set_SELECT_1",
			setValues: []string{"SELECT 1"},
			wantValue: "SELECT 1",
		},
		{
			name:      "set_empty_clears",
			setValues: []string{"SELECT 1", ""},
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
			name: "typing_updates_value",
			setup: func(m *Model) {
				m.Focus()
				m.SetSize(40, 5)
			},
			runUpdate: func(m Model) (Model, tea.Cmd) {
				var cmd tea.Cmd
				for _, r := range "hi" {
					m, cmd = m.Update(testutil.KeyRune(r))
				}
				return m, cmd
			},
			wantValue: "hi",
		},
		{
			name: "window_size_preserves_value",
			setup: func(m *Model) {
				m.SetValue("SELECT * FROM t")
			},
			runUpdate: func(m Model) (Model, tea.Cmd) {
				return m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
			},
			wantValue: "SELECT * FROM t",
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
		wantContains string // substring or pattern to find in plain view (after stripANSI)
	}{
		{
			name:         "empty_shows_prompt_or_placeholder",
			setValue:     "",
			wantContains: "sql>",
		},
		{
			name:         "with_value_shows_content",
			setValue:     "SELECT 1",
			wantContains: "SELECT 1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			if tt.setValue != "" {
				m.SetValue(tt.setValue)
			}
			m.SetSize(40, 3)
			view := m.View()
			if view.Content == "" {
				t.Fatal("View() is empty")
			}
			plain := testutil.StripANSI(view.Content)
			if plain == "" {
				t.Fatal("View() has no plain text after stripping ANSI")
			}
			if regexp.MustCompile(regexp.QuoteMeta(tt.wantContains)).FindString(plain) == "" {
				// Fallback: empty case may show placeholder instead of prompt
				if tt.setValue == "" && regexp.MustCompile("Enter your query").FindString(plain) == "" {
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
			name:     "run_query_binding",
			wantKey:  "ctrl+s",
			wantHelp: "run query",
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
