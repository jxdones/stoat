package statusbar

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name         string
		wantContains string
	}{
		{
			name:         "default_message_is_ready",
			wantContains: " Ready",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			view := m.View(80)
			if view.Content == "" {
				t.Error("View(80).Content is empty")
			}
			if !strings.Contains(view.Content, tt.wantContains) {
				t.Errorf("View(80).Content should contain %q; got %q", tt.wantContains, view.Content)
			}
		})
	}
}

func TestSetStatus(t *testing.T) {
	tests := []struct {
		name         string
		text         string
		kind         Kind
		wantContains string
	}{
		{
			name:         "info_message",
			text:         "Connected",
			kind:         Info,
			wantContains: "Connected",
		},
		{
			name:         "success_message",
			text:         "Saved",
			kind:         Success,
			wantContains: "Saved",
		},
		{
			name:         "warning_message",
			text:         "Slow query",
			kind:         Warning,
			wantContains: "Slow query",
		},
		{
			name:         "error_message",
			text:         "Connection failed",
			kind:         Error,
			wantContains: "Connection failed",
		},
		{
			name:         "empty_text",
			text:         "",
			kind:         Info,
			wantContains: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			m.SetStatus(tt.text, tt.kind)
			view := m.View(80)
			if view.Content == "" && tt.wantContains != "" {
				t.Error("View(80).Content is empty")
			}
			if tt.wantContains != "" && !strings.Contains(view.Content, tt.wantContains) {
				t.Errorf("View(80).Content should contain %q; got %q", tt.wantContains, view.Content)
			}
		})
	}
}

func TestSetStatusWithTTLReturnsExpirationCmd(t *testing.T) {
	m := New()
	cmd := m.SetStatusWithTTL("Done", Success, time.Millisecond)
	if cmd == nil {
		t.Fatal("expected non-nil expiration cmd")
	}
	msg := cmd()
	expired, ok := msg.(ExpiredMsg)
	if !ok {
		t.Fatalf("expected ExpiredMsg, got %T", msg)
	}
	if expired.Seq != m.seq {
		t.Fatalf("expected expiration seq %d, got %d", m.seq, expired.Seq)
	}
}

func TestHandleExpired(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*Model) int // returns seq to pass to HandleExpired
		wantText string
		wantKind Kind
	}{
		{
			name: "ignores_stale_sequence",
			setup: func(m *Model) int {
				_ = m.SetStatusWithTTL("first", Warning, 0)
				firstSeq := m.seq
				_ = m.SetStatusWithTTL("second", Success, 0)
				return firstSeq
			},
			wantText: "second",
			wantKind: Success,
		},
		{
			name: "resets_to_default_when_seq_matches",
			setup: func(m *Model) int {
				_ = m.SetStatusWithTTL("boom", Error, 0)
				return m.seq
			},
			wantText: defaultText,
			wantKind: Info,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			seq := tt.setup(&m)
			m.HandleExpired(ExpiredMsg{Seq: seq})
			if m.text != tt.wantText {
				t.Errorf("text = %q, want %q", m.text, tt.wantText)
			}
			if m.kind != tt.wantKind {
				t.Errorf("kind = %v, want %v", m.kind, tt.wantKind)
			}
		})
	}
}

func TestView(t *testing.T) {
	tests := []struct {
		name         string
		width        int
		setup        func(*Model)
		wantContains string
		nonEmpty     bool
	}{
		{
			name:         "default_at_80_width",
			width:        80,
			wantContains: " Ready",
			nonEmpty:     true,
		},
		{
			name:         "small_width_truncates_content",
			width:        12,
			setup:        func(m *Model) { m.SetStatus("very long status message here", Info) },
			wantContains: "very long",
			nonEmpty:     true,
		},
		{
			name:         "large_width_shows_full_message",
			width:        200,
			setup:        func(m *Model) { m.SetStatus("Done", Success) },
			wantContains: "Done",
			nonEmpty:     true,
		},
		{
			name:         "left_message_always_visible",
			width:        80,
			wantContains: " Ready",
			nonEmpty:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			if tt.setup != nil {
				tt.setup(&m)
			}
			view := m.View(tt.width)
			if tt.nonEmpty && view.Content == "" {
				t.Error("View().Content is empty")
			}
			if tt.wantContains != "" && !strings.Contains(view.Content, tt.wantContains) {
				t.Errorf("View(%d).Content should contain %q; got %q", tt.width, tt.wantContains, view.Content)
			}
		})
	}
}

func TestViewRightZone(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(*Model)
		wantContains []string
		wantAbsent   []string
	}{
		{
			name:         "connection_name_shown",
			setup:        func(m *Model) { m.SetConnectionName("supabase-prod") },
			wantContains: []string{"supabase-prod"},
		},
		{
			name:         "read_only_badge_shown",
			setup:        func(m *Model) { m.SetReadOnly(true) },
			wantContains: []string{"[RO]"},
		},
		{
			name: "connection_name_and_read_only_both_shown",
			setup: func(m *Model) {
				m.SetConnectionName("supabase-prod")
				m.SetReadOnly(true)
			},
			wantContains: []string{"supabase-prod", "[RO]"},
		},
		{
			name:         "long_name_truncated_with_ellipsis",
			setup:        func(m *Model) { m.SetConnectionName("this-is-a-very-long-connection-name-exceeding-limit") },
			wantContains: []string{"…"},
		},
		{
			name:       "no_name_no_ro_right_zone_absent",
			setup:      func(m *Model) {},
			wantAbsent: []string{"[RO]"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			tt.setup(&m)
			view := m.View(80)
			for _, want := range tt.wantContains {
				if !strings.Contains(view.Content, want) {
					t.Errorf("View(80).Content should contain %q; got %q", want, view.Content)
				}
			}
			for _, absent := range tt.wantAbsent {
				if strings.Contains(view.Content, absent) {
					t.Errorf("View(80).Content should not contain %q; got %q", absent, view.Content)
				}
			}
		})
	}
}

func TestViewWidth(t *testing.T) {
	for _, width := range []int{80, 100, 120, 200} {
		t.Run(fmt.Sprintf("width_%d_fills_exactly", width), func(t *testing.T) {
			m := New()
			m.SetConnectionName("supabase-prod")
			m.SetReadOnly(true)
			view := m.View(width)
			got := ansi.StringWidth(view.Content)
			if got != width {
				t.Errorf("View(%d) rendered width = %d, want %d", width, got, width)
			}
		})
	}
}
