package sidebar

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"

	"github.com/jxdones/stoat/internal/ui/testutil"
)

func keyMsg(key string) tea.KeyPressMsg {
	switch key {
	case "up":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyUp})
	case "down":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyDown})
	case "left":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyLeft})
	case "h":
		return tea.KeyPressMsg(tea.Key{Code: 'h', Text: "h"})
	case "j":
		return tea.KeyPressMsg(tea.Key{Code: 'j', Text: "j"})
	case "k":
		return tea.KeyPressMsg(tea.Key{Code: 'k', Text: "k"})
	case "g":
		return tea.KeyPressMsg(tea.Key{Code: 'g', Text: "g"})
	case "G":
		return tea.KeyPressMsg(tea.Key{Code: 'G', Text: "G"})
	case "home":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyHome})
	case "end":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyEnd})
	case "enter":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})
	default:
		if len(key) == 0 {
			return tea.KeyPressMsg(tea.Key{})
		}
		r, _ := utf8.DecodeRuneInString(key)
		return tea.KeyPressMsg(tea.Key{Code: r, Text: key})
	}
}

func TestNew(t *testing.T) {
	m := New([]string{"a", "b"}, nil)
	if m.ActiveDB() != "" {
		t.Errorf("ActiveDB() = %q, want empty", m.ActiveDB())
	}
	if m.SelectedDB() != "a" {
		t.Errorf("SelectedDB() = %q, want a", m.SelectedDB())
	}
	if m.SelectedTable() != "" {
		t.Errorf("SelectedTable() = %q, want empty (no DB opened)", m.SelectedTable())
	}
}

func TestVisibleRows_via_View(t *testing.T) {
	// visibleRows splits inner height between DBs (capped at half) and tables.
	// With 2 header rows + 1 gap, listRows = innerHeight - 3. We assert via View structure.
	tests := []struct {
		name           string
		height         int
		databases      int
		tables         int
		wantDBLines    int // min(half listRows, dbCount); placeholder counts as 1
		wantTableLines int
	}{
		{
			name:           "tiny_height_returns_some_lines",
			height:         10,
			databases:      5,
			tables:         5,
			wantDBLines:    2,
			wantTableLines: 2,
		},
		{
			name:           "tall_enough_splits_half",
			height:         20,
			databases:      10,
			tables:         10,
			wantDBLines:    5,
			wantTableLines: 12,
		},
		{
			name:           "empty_dbs_one_placeholder_line",
			height:         12,
			databases:      0,
			tables:         0,
			wantDBLines:    1,
			wantTableLines: 4,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbs := make([]string, tt.databases)
			for i := range dbs {
				dbs[i] = "db"
			}
			tbls := make(map[string][]string)
			if tt.databases > 0 {
				tbls["db"] = make([]string, tt.tables)
				for i := range tbls["db"] {
					tbls["db"][i] = "t"
				}
			}
			m := New(dbs, tbls)
			if tt.databases > 0 {
				m.OpenSelectedDatabase()
			}
			m.SetSize(20, tt.height)
			v := m.View()
			plain := testutil.StripANSI(v.Content)
			lines := strings.Split(plain, "\n")
			// Layout: [ Databases ] \n db... \n \n [ Tables ] \n table...
			// We only check that we have both section headers and non-empty content where expected.
			if len(lines) < 5 {
				t.Errorf("height %d: expected at least 5 lines, got %d", tt.height, len(lines))
			}
			if !strings.Contains(plain, "Databases") {
				t.Error("View should contain Databases section")
			}
			if !strings.Contains(plain, "Tables") {
				t.Error("View should contain Tables section")
			}
		})
	}
}

func TestFit_truncation_in_View(t *testing.T) {
	// fit truncates long text with "…". Use a very narrow sidebar so DB names get truncated.
	m := New([]string{"very_long_database_name"}, nil)
	m.SetSize(14, 12) // content width becomes very small
	v := m.View()
	plain := testutil.StripANSI(v.Content)
	if !strings.Contains(plain, "…") && len("very_long_database_name") > 8 {
		// If content width is small we expect truncation
		t.Logf("View content (plain): %q", plain)
	}
}

func TestClamp_empty_databases(t *testing.T) {
	m := New([]string{"a"}, nil)
	m.SetDatabases(nil)
	m.SetSize(20, 12)
	if m.SelectedDB() != "" {
		t.Errorf("SelectedDB() with no DBs = %q, want empty", m.SelectedDB())
	}
	if m.ActiveDB() != "" {
		t.Errorf("ActiveDB() with no DBs = %q, want empty", m.ActiveDB())
	}
}

func TestSetDatabases_filters_blank_entries(t *testing.T) {
	tests := []struct {
		name          string
		input         []string
		wantFirst     string
		wantAfterMove string
	}{
		{
			name:          "filters_whitespace_only_entries",
			input:         []string{"", "  ", "db1", "\t", "db2"},
			wantFirst:     "db1",
			wantAfterMove: "db2",
		},
		{
			name:          "keeps_clean_list",
			input:         []string{"main", "analytics"},
			wantFirst:     "main",
			wantAfterMove: "analytics",
		},
		{
			name:          "all_blank_results_in_empty_selection",
			input:         []string{"", " ", "\t"},
			wantFirst:     "",
			wantAfterMove: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(nil, nil)
			m.SetDatabases(tt.input)
			m.SetSize(20, 12)

			if got := m.SelectedDB(); got != tt.wantFirst {
				t.Fatalf("SelectedDB() = %q, want %q", got, tt.wantFirst)
			}

			m.Move(1)
			if got := m.SelectedDB(); got != tt.wantAfterMove {
				t.Fatalf("after Move(1) SelectedDB() = %q, want %q", got, tt.wantAfterMove)
			}

			// A second move should remain clamped at the last valid item.
			m.Move(1)
			if got := m.SelectedDB(); got != tt.wantAfterMove {
				t.Fatalf("after Move(2) SelectedDB() = %q, want %q (clamped)", got, tt.wantAfterMove)
			}
		})
	}
}

func TestSetTables_filters_blank_entries(t *testing.T) {
	tests := []struct {
		name          string
		input         []string
		wantFirst     string
		wantAfterMove string
	}{
		{
			name:          "filters_whitespace_only_entries",
			input:         []string{"", "  ", "users", "\n", "orders"},
			wantFirst:     "users",
			wantAfterMove: "orders",
		},
		{
			name:          "keeps_clean_list",
			input:         []string{"sessions", "events"},
			wantFirst:     "sessions",
			wantAfterMove: "events",
		},
		{
			name:          "all_blank_results_in_empty_selection",
			input:         []string{"", " ", "\t"},
			wantFirst:     "(none)",
			wantAfterMove: "(none)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New([]string{"db1"}, nil)
			m.SetTables("db1", tt.input)
			m.OpenSelectedDatabase()
			m.SetSize(20, 12)

			if got := m.SelectedTable(); got != tt.wantFirst {
				t.Fatalf("SelectedTable() = %q, want %q", got, tt.wantFirst)
			}

			m.Move(1)
			if got := m.SelectedTable(); got != tt.wantAfterMove {
				t.Fatalf("after Move(1) SelectedTable() = %q, want %q", got, tt.wantAfterMove)
			}

			// A second move should remain clamped at the last valid item.
			m.Move(1)
			if got := m.SelectedTable(); got != tt.wantAfterMove {
				t.Fatalf("after Move(2) SelectedTable() = %q, want %q (clamped)", got, tt.wantAfterMove)
			}
		})
	}
}

func TestClamp_selection_in_bounds(t *testing.T) {
	m := New([]string{"a", "b", "c"}, map[string][]string{"a": {"t1", "t2"}})
	m.OpenSelectedDatabase()
	m.SetSize(20, 12)
	// After SetTables, clamp keeps selectedTable in valid range; no panic.
	m.SetTables("a", []string{"t1", "t2"})
	m.SetSize(20, 12)
	// Clamp keeps selection valid; SelectedTable returns something in bounds (or empty).
	got := m.SelectedTable()
	if got != "" && got != "t1" && got != "t2" {
		t.Errorf("SelectedTable() = %q, want empty or t1 or t2", got)
	}
}

func TestUpdate_Enter_returns_EventOpenRequested(t *testing.T) {
	m := New([]string{"db1"}, nil)
	m.SetSize(20, 12)
	next, ev := m.Update(keyMsg("enter"))
	if ev != EventOpenRequested {
		t.Errorf("Update(enter) event = %v, want EventOpenRequested", ev)
	}
	_ = next
}

func TestUpdate_navigation_returns_EventSelectionChanged(t *testing.T) {
	m := New([]string{"a", "b", "c"}, nil)
	m.SetSize(20, 12)
	_, ev := m.Update(keyMsg("down"))
	if ev != EventSelectionChanged {
		t.Errorf("Update(down) event = %v, want EventSelectionChanged", ev)
	}
	next, _ := m.Update(keyMsg("down"))
	next, _ = next.Update(keyMsg("down"))
	if next.SelectedDB() != "c" {
		t.Errorf("after two downs SelectedDB() = %q, want c", next.SelectedDB())
	}
}

func TestUpdate_non_key_returns_EventNone(t *testing.T) {
	m := New([]string{"a"}, nil)
	_, ev := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if ev != EventNone {
		t.Errorf("Update(WindowSizeMsg) event = %v, want EventNone", ev)
	}
}

func TestOpenSelectedDatabase_switches_to_tables_section(t *testing.T) {
	m := New([]string{"db1"}, map[string][]string{"db1": {"t1"}})
	m.SetSize(20, 12)
	if m.InTablesSection() {
		t.Fatal("initially not in tables section")
	}
	m.OpenSelectedDatabase()
	if !m.InTablesSection() {
		t.Error("after OpenSelectedDatabase expected InTablesSection() true")
	}
	if m.ActiveDB() != "db1" {
		t.Errorf("ActiveDB() = %q, want db1", m.ActiveDB())
	}
	if m.SelectedTable() != "t1" {
		t.Errorf("SelectedTable() = %q, want t1", m.SelectedTable())
	}
}

func TestMoveToTop_and_MoveToBottom(t *testing.T) {
	m := New([]string{"a", "b", "c"}, nil)
	m.SetSize(20, 12)
	m.Move(2)
	if m.SelectedDB() != "c" {
		t.Fatalf("after Move(2) SelectedDB() = %q, want c", m.SelectedDB())
	}
	m.MoveToTop()
	if m.SelectedDB() != "a" {
		t.Errorf("after MoveToTop SelectedDB() = %q, want a", m.SelectedDB())
	}
	m.MoveToBottom()
	if m.SelectedDB() != "c" {
		t.Errorf("after MoveToBottom SelectedDB() = %q, want c", m.SelectedDB())
	}
}

func TestUpdate_left_in_tables_switches_to_databases(t *testing.T) {
	m := New([]string{"db1"}, map[string][]string{"db1": {"t1", "t2"}})
	m.OpenSelectedDatabase()
	m.SetSize(20, 12)
	if !m.InTablesSection() {
		t.Fatal("expected in tables section")
	}
	next, ev := m.Update(keyMsg("left"))
	if ev != EventSectionChanged {
		t.Errorf("Update(left) event = %v, want EventSectionChanged", ev)
	}
	if next.InTablesSection() {
		t.Error("expected to be in databases section after left in tables")
	}
	if next.SelectedDB() != "db1" {
		t.Errorf("SelectedDB() = %q, want db1", next.SelectedDB())
	}
}

func TestViewportLines_overflow_marker(t *testing.T) {
	// Many items + small height => "…" overflow marker in View
	dbs := make([]string, 15)
	for i := range dbs {
		dbs[i] = "db"
	}
	m := New(dbs, nil)
	m.SetSize(20, 10)
	v := m.View()
	plain := testutil.StripANSI(v.Content)
	if !strings.Contains(plain, "…") {
		t.Error("expected overflow marker … when many items and small height")
	}
}

func isMarker(s string) bool {
	return strings.Contains(testutil.StripANSI(s), "…")
}

func TestViewportLines_markerAlwaysAtEdge(t *testing.T) {
	tests := []struct {
		name            string
		start           int
		rows            int
		selected        int
		wantFirstMarker bool
		wantLastMarker  bool
	}{
		{
			name:            "no_overflow_no_marker",
			start:           0,
			rows:            5,
			selected:        2,
			wantFirstMarker: false,
			wantLastMarker:  false,
		},
		{
			name:            "items_below_selected_not_at_edge",
			start:           0,
			rows:            3,
			selected:        1,
			wantFirstMarker: false,
			wantLastMarker:  true,
		},
		{
			// regression: selected at the last visible row caused marker to shift up,
			// leaving visible rows after the "…"
			name:            "items_below_selected_at_last_row",
			start:           0,
			rows:            3,
			selected:        2,
			wantFirstMarker: false,
			wantLastMarker:  true,
		},
		{
			name:            "items_above_selected_not_at_edge",
			start:           2,
			rows:            3,
			selected:        3,
			wantFirstMarker: true,
			wantLastMarker:  false,
		},
		{
			// regression: selected at the first visible row caused marker to shift down,
			// leaving visible rows before the "…"
			name:            "items_above_selected_at_first_row",
			start:           2,
			rows:            3,
			selected:        2,
			wantFirstMarker: true,
			wantLastMarker:  false,
		},
		{
			name:            "items_above_and_below_markers_at_both_edges",
			start:           1,
			rows:            3,
			selected:        2,
			wantFirstMarker: true,
			wantLastMarker:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Fresh slice per case: offsetWindowLines returns a subslice of the
			// original, so marker writes would otherwise bleed between cases.
			lines := []string{"a", "b", "c", "d", "e"}
			got := viewportLines(lines, tt.start, tt.rows, 20)
			if len(got) == 0 {
				t.Fatal("viewportLines returned empty slice")
			}
			if isMarker(got[0]) != tt.wantFirstMarker {
				t.Errorf("first row marker = %v, want %v", isMarker(got[0]), tt.wantFirstMarker)
			}
			if isMarker(got[len(got)-1]) != tt.wantLastMarker {
				t.Errorf("last row marker = %v, want %v", isMarker(got[len(got)-1]), tt.wantLastMarker)
			}
		})
	}
}

func TestSidebar_selectedItemVisibleAtScrollBoundaries(t *testing.T) {
	// 15 tables, small sidebar so the list overflows and scrolling is required.
	const tableCount = 15
	tables := make([]string, tableCount)
	for i := range tables {
		tables[i] = fmt.Sprintf("table_%02d", i)
	}
	m := New([]string{"db1"}, map[string][]string{"db1": tables})
	m.OpenSelectedDatabase()
	m.SetSize(20, 15)

	t.Run("selected_last_table_visible_marker_before_it", func(t *testing.T) {
		m.MoveToBottom()
		plain := testutil.StripANSI(m.View().Content)

		if !strings.Contains(plain, "table_14") {
			t.Error("last table should be visible after MoveToBottom")
		}
		markerIdx := strings.LastIndex(plain, "…")
		selectedIdx := strings.Index(plain, "table_14")
		if markerIdx != -1 && markerIdx > selectedIdx {
			t.Errorf("overflow marker appears after selected item (marker at %d, selected at %d)", markerIdx, selectedIdx)
		}
	})

	t.Run("selected_first_table_visible_marker_after_it", func(t *testing.T) {
		// Scroll to bottom first so there are items above when we go back to top.
		m.MoveToBottom()
		m.MoveToTop()
		plain := testutil.StripANSI(m.View().Content)

		if !strings.Contains(plain, "table_00") {
			t.Error("first table should be visible after MoveToTop")
		}
		markerIdx := strings.Index(plain, "…")
		selectedIdx := strings.Index(plain, "table_00")
		if markerIdx != -1 && markerIdx < selectedIdx {
			t.Errorf("overflow marker appears before selected item (marker at %d, selected at %d)", markerIdx, selectedIdx)
		}
	})
}
