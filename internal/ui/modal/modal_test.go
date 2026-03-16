package modal

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestRender(t *testing.T) {
	tests := []struct {
		name      string
		title     string
		content   string
		shortcuts string
		width     int
		wantIn    []string
	}{
		{
			name:      "contains_title",
			title:     "Connections",
			content:   "item one",
			shortcuts: "j/k navigate",
			width:     50,
			wantIn:    []string{"Connections"},
		},
		{
			name:      "contains_content",
			title:     "Connections",
			content:   "local-sqlite",
			shortcuts: "j/k navigate",
			width:     50,
			wantIn:    []string{"local-sqlite"},
		},
		{
			name:      "contains_shortcuts",
			title:     "Connections",
			content:   "item one",
			shortcuts: "j/k navigate · esc close",
			width:     50,
			wantIn:    []string{"j/k navigate · esc close"},
		},
		{
			name:      "contains_all_three",
			title:     "Settings",
			content:   "Theme: default",
			shortcuts: "enter select",
			width:     60,
			wantIn:    []string{"Settings", "Theme: default", "enter select"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ansi.Strip(Render(tt.title, tt.content, tt.shortcuts, tt.width))
			for _, want := range tt.wantIn {
				if !strings.Contains(got, want) {
					t.Errorf("Render() output missing %q\ngot:\n%s", want, got)
				}
			}
		})
	}
}
