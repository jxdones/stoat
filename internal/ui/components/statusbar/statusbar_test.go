package statusbar

import (
	"strings"
	"testing"
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
			wantContains: "very long", // truncated with "…"
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
			name:         "min_content_width_used_when_width_small",
			width:        5,
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
