package model

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
	"github.com/jxdones/stoat/internal/ui/components/table"
)

func TestView(t *testing.T) {
	tests := []struct {
		name         string
		width        int
		height       int
		wantContains []string
		altScreen    bool
	}{
		{
			name:         "normal_size_returns_full_layout",
			width:        80,
			height:       24,
			wantContains: []string{"No connection", "Filter:", "No data source", " Ready", "j/k", "g/G"},
			altScreen:    true,
		},
		{
			name:         "small_height_shows_compact_resize_message",
			width:        80,
			height:       10,
			wantContains: []string{"Terminal too small", "Minimum: 80x24", "Resize the terminal", "q quit"},
			altScreen:    true,
		},
		{
			name:         "single_row_height_does_not_panic",
			width:        80,
			height:       1,
			wantContains: []string{},
			altScreen:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			m.view.width = tt.width
			m.view.height = tt.height
			v := m.View()
			if v.Content == "" && len(tt.wantContains) > 0 {
				t.Error("View() returned empty Content")
			}
			if v.AltScreen != tt.altScreen {
				t.Errorf("View().AltScreen = %v, want %v", v.AltScreen, tt.altScreen)
			}
			for _, sub := range tt.wantContains {
				if !strings.Contains(v.Content, sub) {
					t.Errorf("View().Content should contain %q", sub)
				}
			}
		})
	}
}

func TestRenderBase(t *testing.T) {
	tests := []struct {
		name         string
		frame        layout
		wantEmpty    bool
		wantContains []string
	}{
		{
			name:      "zero_main_content_returns_empty",
			frame:     computeLayout(80, 1, 2, mainDetailRowsNormal),
			wantEmpty: true,
		},
		{
			name:         "normal_frame_returns_base_with_header_and_placeholder",
			frame:        computeLayout(80, 24, 2, mainDetailRowsNormal),
			wantContains: []string{"No connection", "Filter:", "No data source"},
		},
		{
			name:         "narrow_width_base_still_has_header",
			frame:        computeLayout(50, 20, 2, mainDetailRowsNormal),
			wantContains: []string{"No connection", "Filter:"},
		},
		{
			name:         "small_height_base_has_table_placeholder",
			frame:        computeLayout(80, 15, 2, mainDetailRowsNormal),
			wantContains: []string{"No connection", "No data source"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			got := m.renderBase(tt.frame)
			if tt.wantEmpty {
				if got != "" {
					t.Errorf("renderBase() should return empty; got %d bytes", len(got))
				}
				return
			}
			if got == "" {
				t.Error("renderBase() returned empty string")
			}
			for _, sub := range tt.wantContains {
				if !strings.Contains(got, sub) {
					t.Errorf("renderBase() should contain %q; got %q", sub, got)
				}
			}
		})
	}
}

func TestRenderHeader(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(*Model)
		width        int
		wantContains []string
	}{
		{
			name:         "no_db_shows_no_connection",
			setup:        func(m *Model) {},
			width:        60,
			wantContains: []string{"No connection", "Filter:"},
		},
		{
			name: "with_db_shows_db_name",
			setup: func(m *Model) {
				m.sidebar.SetDatabases([]string{"mydb"})
				m.sidebar.OpenSelectedDatabase()
			},
			width:        60,
			wantContains: []string{"mydb", "Filter:"},
		},
		{
			name: "with_db_and_table_shows_db_dot_table",
			setup: func(m *Model) {
				m.sidebar.SetDatabases([]string{"mydb"})
				m.sidebar.SetTables("mydb", []string{"users"})
				m.sidebar.OpenSelectedDatabase()
			},
			width:        60,
			wantContains: []string{"mydb.users", "Filter:"},
		},
		{
			name:         "always_shows_filter_and_columns_info",
			setup:        func(m *Model) {},
			width:        80,
			wantContains: []string{"Filter:", "columns:", "visible:", "page", "rows"},
		},
		{
			name: "single_row_shows_row_not_rows",
			setup: func(m *Model) {
				m.table = table.New(
					[]table.Column{{Key: "id", Title: "id", Type: "text", MinWidth: 2, Order: 1}},
					[]table.Row{{"id": "1"}},
				)
			},
			width:        60,
			wantContains: []string{"1 row"},
		},
		{
			name:         "zero_rows_shows_0_rows",
			setup:        func(m *Model) {},
			width:        60,
			wantContains: []string{"0 rows"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			if tt.setup != nil {
				tt.setup(&m)
			}
			got := m.renderHeader(tt.width)
			if got == "" {
				t.Error("renderHeader() returned empty string")
			}
			for _, sub := range tt.wantContains {
				if !strings.Contains(got, sub) {
					t.Errorf("renderHeader() should contain %q; got %q", sub, got)
				}
			}
		})
	}
}

func TestRenderStatus(t *testing.T) {
	tests := []struct {
		name         string
		width        int
		wantContains string
	}{
		{
			name:         "returns_statusbar_content_for_model_width",
			width:        80,
			wantContains: " Ready",
		},
		{
			name:         "uses_view_width_not_fixed_value",
			width:        120,
			wantContains: " Ready",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			m.view.width = tt.width
			got := m.renderStatus()
			want := m.statusbar.View(tt.width).Content
			if got != want {
				t.Errorf("renderStatus() != statusbar.View(%d).Content:\ngot  %q\nwant %q", tt.width, got, want)
			}
			if tt.wantContains != "" && !strings.Contains(got, tt.wantContains) {
				t.Errorf("renderStatus() should contain %q; got %q", tt.wantContains, got)
			}
		})
	}
}

func TestRenderOptions(t *testing.T) {
	tests := []struct {
		name         string
		width        int
		wantContains []string
	}{
		{
			name:         "base_returns_non_empty_with_help",
			width:        80,
			wantContains: []string{"j/k", "g/G"},
		},
		{
			name:         "large_width_still_renders",
			width:        200,
			wantContains: []string{"tab", "esc"},
		},
		{
			name:         "small_width_uses_inner_width_at_least_one",
			width:        3,
			wantContains: []string{},
		},
		{
			name:         "width_two_inner_width_one",
			width:        2,
			wantContains: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			m.view.width = tt.width
			got := m.renderOptions()
			if got == "" {
				t.Error("renderOptions() returned empty string")
			}
			for _, sub := range tt.wantContains {
				if !strings.Contains(got, sub) {
					t.Errorf("renderOptions() should contain %q; got %q", sub, got)
				}
			}
		})
	}
}

func TestRenderDetail(t *testing.T) {
	tests := []struct {
		name         string
		setupTable   func(*Model)
		width        int
		wantContains []string
	}{
		{
			name: "no_cell_shows_placeholder",
			setupTable: func(m *Model) {
				// default New() has empty table; no setup
			},
			width:        60,
			wantContains: []string{"Ln 0, Col 0", "field: -", "type: -", "value: -"},
		},
		{
			name:         "no_cell_small_width_still_renders",
			setupTable:   func(m *Model) {},
			width:        10,
			wantContains: []string{"Ln 0", "field: -"},
		},
		{
			name: "with_cell_shows_field_and_value",
			setupTable: func(m *Model) {
				m.table = table.New(
					[]table.Column{{Key: "id", Title: "id", Type: "int", MinWidth: 2, Order: 1}},
					[]table.Row{{"id": "42"}},
				)
				m.table.SetSize(20, 5)
			},
			width:        60,
			wantContains: []string{"Ln 1", "Col 1", "field: id", "type: int", "value: 42"},
		},
		{
			name: "with_cell_empty_type_defaults_to_text",
			setupTable: func(m *Model) {
				m.table = table.New(
					[]table.Column{{Key: "name", Title: "name", Type: "", MinWidth: 4, Order: 1}},
					[]table.Row{{"name": "alice"}},
				)
				m.table.SetSize(20, 5)
			},
			width:        60,
			wantContains: []string{"field: name", "type: text", "value: alice"},
		},
		{
			name: "with_cell_narrow_width_truncates",
			setupTable: func(m *Model) {
				m.table = table.New(
					[]table.Column{{Key: "x", Title: "x", Type: "text", MinWidth: 2, Order: 1}},
					[]table.Row{{"x": "longvalue"}},
				)
				m.table.SetSize(20, 5)
			},
			width:        20,
			wantContains: []string{"Ln 1", "Col 1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			if tt.setupTable != nil {
				tt.setupTable(&m)
			}
			got := m.renderDetail(tt.width)
			if got == "" {
				t.Error("renderDetail() returned empty string")
			}
			for _, sub := range tt.wantContains {
				if !strings.Contains(got, sub) {
					t.Errorf("renderDetail() should contain %q; got %q", sub, got)
				}
			}
		})
	}
}

func TestRenderTable(t *testing.T) {
	tests := []struct {
		name          string
		setupTable    func(*Model)
		width         int
		height        int
		wantContains  []string
		noPlaceholder bool
	}{
		{
			name:         "empty_table_shows_placeholder",
			setupTable:   nil,
			width:        60,
			height:       10,
			wantContains: []string{"No data source connected", "Press Esc then q to exit", "Ctrl+C"},
		},
		{
			name:         "empty_table_small_dimensions_no_panic",
			setupTable:   nil,
			width:        20,
			height:       3,
			wantContains: []string{"No data source"},
		},
		{
			name: "table_with_data_shows_table_content",
			setupTable: func(m *Model) {
				m.table = table.New(
					[]table.Column{{Key: "col1", Title: "col1", Type: "text", MinWidth: 6, Order: 1}},
					[]table.Row{{"col1": "cell1"}},
				)
				m.table.SetSize(40, 5)
			},
			width:         60,
			height:        10,
			wantContains:  []string{"col1", "cell1"},
			noPlaceholder: true,
		},
		{
			name: "table_with_data_does_not_show_placeholder",
			setupTable: func(m *Model) {
				m.table = table.New(
					[]table.Column{{Key: "a", Title: "a", Type: "text", MinWidth: 2, Order: 1}},
					[]table.Row{{"a": "b"}},
				)
				m.table.SetSize(20, 5)
			},
			width:         40,
			height:        8,
			noPlaceholder: true,
			wantContains:  []string{"a"},
		},
		{
			name:         "empty_table_focus_does_not_affect_placeholder_text",
			setupTable:   nil,
			width:        50,
			height:       5,
			wantContains: []string{"No data source connected"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			if tt.setupTable != nil {
				tt.setupTable(&m)
			}
			got := m.renderTable(tt.width, tt.height, m.table)
			if got == "" {
				t.Error("renderTable() returned empty string")
			}
			for _, sub := range tt.wantContains {
				if !strings.Contains(got, sub) {
					t.Errorf("renderTable() should contain %q; got %q", sub, got)
				}
			}
			if tt.noPlaceholder && strings.Contains(got, "No data source connected") {
				t.Error("renderTable() should not show placeholder when table has data")
			}
		})
	}
}

func TestNormalizeCanvas(t *testing.T) {
	tests := []struct {
		name    string
		content string
		width   int
		height  int
		want    string
	}{
		{
			name:    "zero_width_returns_empty",
			content: "hello",
			width:   0,
			height:  1,
			want:    "",
		},
		{
			name:    "zero_height_returns_empty",
			content: "hello",
			width:   5,
			height:  0,
			want:    "",
		},
		{
			name:    "negative_width_returns_empty",
			content: "x",
			width:   -1,
			height:  1,
			want:    "",
		},
		{
			name:    "fewer_lines_than_height_pads_with_blank_lines",
			content: "a\nb",
			width:   3,
			height:  4,
			want:    "a  \nb  \n   \n   ",
		},
		{
			name:    "more_lines_than_height_truncates",
			content: "a\nb\nc\nd\ne",
			width:   3,
			height:  3,
			want:    "a  \nb  \nc  ",
		},
		{
			name:    "long_line_truncates_to_width",
			content: "hello",
			width:   3,
			height:  1,
			want:    "hel",
		},
		{
			name:    "short_line_pads_to_width",
			content: "ab",
			width:   5,
			height:  1,
			want:    "ab   ",
		},
		{
			name:    "empty_content_pads_to_height_and_width",
			content: "",
			width:   4,
			height:  2,
			want:    "    \n    ",
		},
		{
			name:    "exact_lines_and_width_unchanged",
			content: "ab\ncd",
			width:   2,
			height:  2,
			want:    "ab\ncd",
		},
		{
			name:    "single_line_exact_width",
			content: "xyz",
			width:   3,
			height:  1,
			want:    "xyz",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeCanvas(tt.content, tt.width, tt.height)
			if got != tt.want {
				t.Errorf("normalizeCanvas(%q, %d, %d):\ngot  %q\nwant %q",
					tt.content, tt.width, tt.height, got, tt.want)
			}
			// Verify the result has exactly height lines.
			if tt.want != "" {
				lines := strings.Split(got, "\n")
				if len(lines) != tt.height {
					t.Errorf("line count = %d, want %d", len(lines), tt.height)
				}
			}
		})
	}
}

func TestStatusBindings(t *testing.T) {
	t.Run("focus_none_includes_q_quit", func(t *testing.T) {
		m := New()
		m.view.focus = FocusNone
		bindings := m.statusBindings()
		assertBindingExists(t, bindings, "q", "quit")
	})

	t.Run("pane_focus_keeps_global_bindings", func(t *testing.T) {
		m := New()
		m.view.focus = FocusTable
		bindings := m.statusBindings()
		assertBindingExists(t, bindings, "tab", "focus panes")
		assertBindingExists(t, bindings, "shift+tab", "focus previous pane")
		assertBindingExists(t, bindings, "esc", "clear focus")
		assertBindingExists(t, bindings, "ctrl+n/b", "next/prev page")
	})
}

func assertBindingExists(t *testing.T, bindings []key.Binding, keyName, desc string) {
	t.Helper()
	for _, binding := range bindings {
		h := binding.Help()
		if h.Key != keyName {
			continue
		}
		if desc != "" && h.Desc != desc {
			t.Fatalf("binding %q desc = %q, want %q", keyName, h.Desc, desc)
		}
		return
	}
	t.Fatalf("expected binding %q not found", keyName)
}

func TestExpandedOptionsHeight(t *testing.T) {
	tests := []struct {
		name     string
		width    int
		bindings [][]key.Binding
	}{
		{
			name:  "expanded_help_height_with_two_bindings",
			width: 100,
			bindings: [][]key.Binding{
				{key.NewBinding(key.WithKeys("j"), key.WithHelp("j", "down"))},
				{key.NewBinding(key.WithKeys("k"), key.WithHelp("k", "up"))},
			},
		},
		{
			name:  "expanded_help_height_with_three_bindings",
			width: 100,
			bindings: [][]key.Binding{
				{key.NewBinding(key.WithKeys("j"), key.WithHelp("j", "down"))},
				{key.NewBinding(key.WithKeys("k"), key.WithHelp("k", "up"))},
				{key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "right"))},
			},
		},
		{
			name:  "expanded_help_height_with_four_bindings",
			width: 100,
			bindings: [][]key.Binding{
				{key.NewBinding(key.WithKeys("j"), key.WithHelp("j", "down"))},
				{key.NewBinding(key.WithKeys("k"), key.WithHelp("k", "up"))},
				{key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "right"))},
				{key.NewBinding(key.WithKeys(";"), key.WithHelp(";", "toggle help"))},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := lipgloss.Height(help.New().FullHelpView(tt.bindings)) + helpBorderHeight + helpTitleHeight
			got := expandedOptionsHeight(tt.width, tt.bindings)
			if got != want {
				t.Errorf("expandedOptionsHeight(%d, %v) = %d, want %d", tt.width, tt.bindings, got, want)
			}
		})
	}
}
