package editbox

import (
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/jxdones/stoat/internal/ui/testutil"
	"github.com/jxdones/stoat/internal/ui/theme"
)

func TestNew(t *testing.T) {
	m := New()
	if got := m.Value(); got != "" {
		t.Errorf("New() Value() = %q, want empty", got)
	}
	if view := m.View(); view.Content == "" {
		t.Error("New() View() is empty")
	}
}

func TestFocusBlur(t *testing.T) {
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
			if view := m.View(); view.Content == "" {
				t.Errorf("View() %s is empty", tt.name)
			}
		})
	}
}

func TestValueSetValue(t *testing.T) {
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
			name:      "set_simple_text",
			setValues: []string{"hello"},
			wantValue: "hello",
		},
		{
			name:      "set_empty_clears",
			setValues: []string{"hello", ""},
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

func TestSetValueMovesCursorToEnd(t *testing.T) {
	tests := []struct {
		name     string
		initial  string
		typed    string
		wantText string
	}{
		{
			name:     "typed_text_appends_after_set_value",
			initial:  "ab",
			typed:    "c",
			wantText: "abc",
		},
		{
			name:     "multi_rune_append",
			initial:  "sto",
			typed:    "at",
			wantText: "stoat",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			m.SetValue(tt.initial)
			m.Focus()
			for _, r := range tt.typed {
				m, _ = m.Update(testutil.KeyRune(r))
			}
			if got := m.Value(); got != tt.wantText {
				t.Errorf("after typing Value() = %q, want %q", got, tt.wantText)
			}
		})
	}
}

func TestHugeStringCanNavigateToBeginningAndEnd(t *testing.T) {
	huge := strings.Repeat("x", 5000)

	tests := []struct {
		name       string
		navigateTo rune
		marker     rune
		wantValue  string
	}{
		{
			name:       "home_moves_cursor_to_beginning",
			navigateTo: tea.KeyHome,
			marker:     'A',
			wantValue:  "A" + huge,
		},
		{
			name:       "end_moves_cursor_to_end",
			navigateTo: tea.KeyEnd,
			marker:     'Z',
			wantValue:  huge + "Z",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			m.SetValue(huge)
			m.Focus()

			m, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: tt.navigateTo}))
			m, _ = m.Update(testutil.KeyRune(tt.marker))

			if got := m.Value(); got != tt.wantValue {
				t.Errorf("Value() = %q, want %q", got, tt.wantValue)
			}
		})
	}
}

func TestSetWidth(t *testing.T) {
	tests := []struct {
		name  string
		width int
	}{
		{
			name:  "zero_width",
			width: 0,
		},
		{
			name:  "small_width",
			width: 8,
		},
		{
			name:  "larger_width",
			width: 40,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			m.SetValue("hello")
			m.SetWidth(tt.width)
			if got := m.Value(); got != "hello" {
				t.Errorf("Value() after SetWidth(%d) = %q, want %q", tt.width, got, "hello")
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
			},
			runUpdate: func(m Model) (Model, tea.Cmd) {
				var cmd tea.Cmd
				for _, r := range "ok" {
					m, cmd = m.Update(testutil.KeyRune(r))
				}
				return m, cmd
			},
			wantValue: "ok",
		},
		{
			name: "window_size_preserves_value",
			setup: func(m *Model) {
				m.SetValue("edit me")
			},
			runUpdate: func(m Model) (Model, tea.Cmd) {
				return m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
			},
			wantValue: "edit me",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			if tt.setup != nil {
				tt.setup(&m)
			}
			m, _ = tt.runUpdate(m)
			if got := m.Value(); got != tt.wantValue {
				t.Errorf("after Update Value() = %q, want %q", got, tt.wantValue)
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
			name:         "empty_not_blank",
			setValue:     "",
			wantContains: "",
		},
		{
			name:         "value_is_rendered",
			setValue:     "row value",
			wantContains: "row value",
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
			if tt.wantContains != "" && regexp.MustCompile(regexp.QuoteMeta(tt.wantContains)).FindString(plain) == "" {
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
			name:     "confirm_binding",
			wantKey:  "enter",
			wantHelp: "confirm",
		},
		{
			name:     "cancel_binding",
			wantKey:  "esc",
			wantHelp: "cancel",
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
