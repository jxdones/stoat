package table

import (
	"maps"
	"regexp"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// stripANSI removes ANSI escape sequences for assertion on plain text content.
func stripANSI(s string) string {
	return regexp.MustCompile(`\x1b\[[0-9;]*m`).ReplaceAllString(s, "")
}

// keyMsg creates a tea.KeyMsg for testing. Supports: "up", "down", "left", "right", "home", "end", "g", "G", "k", "j", "h", "l".
func keyMsg(s string) tea.KeyMsg {
	switch s {
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "home":
		return tea.KeyMsg{Type: tea.KeyHome}
	case "end":
		return tea.KeyMsg{Type: tea.KeyEnd}
	case "g":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")}
	case "G":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")}
	default:
		if len(s) > 0 {
			return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
		}
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")}
	}
}

func Test_isDigitKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want bool
	}{
		{
			name: "digit_0",
			key:  "0",
			want: true,
		},
		{
			name: "digit_1",
			key:  "1",
			want: true,
		},
		{
			name: "digit_5",
			key:  "5",
			want: true,
		},
		{
			name: "digit_9",
			key:  "9",
			want: true,
		},
		{
			name: "letter_a",
			key:  "a",
			want: false,
		},
		{
			name: "letter_g",
			key:  "g",
			want: false,
		},
		{
			name: "letter_G",
			key:  "G",
			want: false,
		},
		{
			name: "space",
			key:  " ",
			want: false,
		},
		{
			name: "multi_char",
			key:  "ab",
			want: false,
		},
		{
			name: "up_arrow",
			key:  "up",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := keyMsg(tt.key)
			got := isDigitKey(msg)
			if got != tt.want {
				t.Errorf("isDigitKey(keyMsg(%q)) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func Test_parseBufferCount(t *testing.T) {
	tests := []struct {
		name   string
		buffer string
		want   int
	}{
		{
			name:   "empty_defaults_to_one",
			buffer: "",
			want:   1,
		},
		{
			name:   "one",
			buffer: "1",
			want:   1,
		},
		{
			name:   "nine",
			buffer: "9",
			want:   9,
		},
		{
			name:   "forty_two",
			buffer: "42",
			want:   42,
		},
		{
			name:   "large",
			buffer: "999",
			want:   999,
		},
		{
			name:   "zero_treated_as_invalid_returns_one",
			buffer: "0",
			want:   1,
		},
		{
			name:   "invalid_returns_one",
			buffer: "abc",
			want:   1,
		},
		{
			name:   "negative_returns_one",
			buffer: "-5",
			want:   1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseBufferCount(tt.buffer)
			if got != tt.want {
				t.Errorf("parseBufferCount(%q) = %d, want %d", tt.buffer, got, tt.want)
			}
		})
	}
}

func Test_normalizeColumns(t *testing.T) {
	tests := []struct {
		name                string
		columns             []Column
		wantLen             int
		wantTitles          []string
		wantOrders          []int
		wantKeys            []string
		wantMinWidths       []int
		checkInputUnchanged bool
		inputMinWidthValue  int
	}{
		{
			name:    "returns_empty_slice_when_input_nil",
			columns: nil,
			wantLen: 0,
		},
		{
			name:    "returns_empty_slice_when_input_empty",
			columns: []Column{},
			wantLen: 0,
		},
		{
			name: "enforces_min_width_floor",
			columns: []Column{
				{Key: "a", Title: "A", MinWidth: 1},
				{Key: "b", Title: "B", MinWidth: 10},
			},
			wantLen:       2,
			wantMinWidths: []int{columnMinWidthFloor, 10},
		},
		{
			name: "sorts_by_order_then_title",
			columns: []Column{
				{Key: "z", Title: "Z", Order: 2},
				{Key: "a", Title: "A", Order: 1},
				{Key: "m", Title: "M", Order: 1},
			},
			wantLen:    3,
			wantTitles: []string{"A", "M", "Z"},
		},
		{
			name: "sort_by_order_takes_precedence_over_title",
			columns: []Column{
				{Key: "c", Title: "C", Order: 10},
				{Key: "a", Title: "A", Order: 0},
				{Key: "b", Title: "B", Order: 5},
			},
			wantLen:    3,
			wantOrders: []int{0, 5, 10},
			wantTitles: []string{"A", "B", "C"},
		},
		{
			name: "sort_by_title_when_all_orders_equal",
			columns: []Column{
				{Key: "z", Title: "Z", Order: 0},
				{Key: "a", Title: "A", Order: 0},
				{Key: "m", Title: "M", Order: 0},
			},
			wantLen:    3,
			wantTitles: []string{"A", "M", "Z"},
		},
		{
			name: "stable_sort_preserves_order_when_order_and_title_equal",
			columns: []Column{
				{Key: "first", Title: "A", Order: 1},
				{Key: "second", Title: "A", Order: 1},
			},
			wantLen:  2,
			wantKeys: []string{"first", "second"},
		},
		{
			name: "single_column_applies_min_width_unchanged_order",
			columns: []Column{
				{Key: "a", Title: "A", MinWidth: 1},
			},
			wantLen:       1,
			wantTitles:    []string{"A"},
			wantMinWidths: []int{columnMinWidthFloor},
		},
		{
			name: "enforces_min_width_and_sorts_together",
			columns: []Column{
				{Key: "z", Title: "Z", Order: 2, MinWidth: 1},
				{Key: "a", Title: "A", Order: 1, MinWidth: 0},
			},
			wantLen:       2,
			wantTitles:    []string{"A", "Z"},
			wantMinWidths: []int{columnMinWidthFloor, columnMinWidthFloor},
		},
		{
			name: "does_not_mutate_input_slice",
			columns: []Column{
				{Key: "a", Title: "A", MinWidth: 5},
			},
			wantLen:             1,
			checkInputUnchanged: true,
			inputMinWidthValue:  5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeColumns(tt.columns)
			if len(got) != tt.wantLen {
				t.Errorf("len(got) = %d, want %d", len(got), tt.wantLen)
			}
			if tt.wantTitles != nil {
				for i, w := range tt.wantTitles {
					if i >= len(got) {
						break
					}
					if got[i].Title != w {
						t.Errorf("got[%d].Title = %q, want %q", i, got[i].Title, w)
					}
				}
			}
			if tt.wantOrders != nil {
				for i, w := range tt.wantOrders {
					if i >= len(got) {
						break
					}
					if got[i].Order != w {
						t.Errorf("got[%d].Order = %d, want %d", i, got[i].Order, w)
					}
				}
			}
			if tt.wantKeys != nil {
				for i, w := range tt.wantKeys {
					if i >= len(got) {
						break
					}
					if got[i].Key != w {
						t.Errorf("got[%d].Key = %q, want %q", i, got[i].Key, w)
					}
				}
			}
			if tt.wantMinWidths != nil {
				for i, w := range tt.wantMinWidths {
					if i >= len(got) {
						break
					}
					if got[i].MinWidth != w {
						t.Errorf("got[%d].MinWidth = %d, want %d", i, got[i].MinWidth, w)
					}
				}
			}
			if tt.checkInputUnchanged && len(tt.columns) > 0 {
				if tt.columns[0].MinWidth != tt.inputMinWidthValue {
					t.Errorf("input was mutated: columns[0].MinWidth = %d, want %d", tt.columns[0].MinWidth, tt.inputMinWidthValue)
				}
			}
		})
	}
}

func Test_padOrTrim(t *testing.T) {
	style := lipgloss.NewStyle() // used only to wrap; we strip ANSI for assertion
	tests := []struct {
		name   string
		s      string
		width  int
		expect string // expected plain text after stripping ANSI
	}{
		{
			name:   "width_zero_clamped_to_one",
			s:      "x",
			width:  0,
			expect: "x",
		},
		{
			name:   "exact_width_unchanged",
			s:      "abc",
			width:  3,
			expect: "abc",
		},
		{
			name:   "short_string_padded_right",
			s:      "hi",
			width:  5,
			expect: "hi   ",
		},
		{
			name:   "long_string_trimmed_with_tilde",
			s:      "hello",
			width:  3,
			expect: "he~",
		},
		{
			name:   "single_rune_width_one",
			s:      "x",
			width:  1,
			expect: "x",
		},
		{
			name:   "empty_string_padded",
			s:      "",
			width:  3,
			expect: "   ",
		},
		{
			name:   "utf8_runes_trimmed_by_runes",
			s:      "café",
			width:  3,
			expect: "ca~",
		},
		{
			name:   "utf8_runes_padded",
			s:      "é",
			width:  3,
			expect: "é  ",
		},
		{
			name:   "negative_width_clamped",
			s:      "a",
			width:  -1,
			expect: "a",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := padOrTrim(tt.s, tt.width, style)
			gotPlain := stripANSI(got)
			if gotPlain != tt.expect {
				t.Errorf("padOrTrim(%q, %d) plain = %q, want %q", tt.s, tt.width, gotPlain, tt.expect)
			}
		})
	}
}

func Test_normalizeCellText_replaces_newline_tab_carriage_return_with_space(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "empty",
			input:  "",
			expect: "",
		},
		{
			name:   "no_special",
			input:  "hello",
			expect: "hello",
		},
		{
			name:   "newline",
			input:  "a\nb",
			expect: "a b",
		},
		{
			name:   "tab",
			input:  "a\tb",
			expect: "a b",
		},
		{
			name:   "carriage_return",
			input:  "a\rb",
			expect: "a b",
		},
		{
			name:   "mixed",
			input:  "a\nb\tc\rd",
			expect: "a b c d",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeCellText(tt.input)
			if got != tt.expect {
				t.Errorf("normalizeCellText(%q) = %q, want %q", tt.input, got, tt.expect)
			}
		})
	}
}

func Test_New(t *testing.T) {
	tests := []struct {
		name     string
		columns  []Column
		rows     []Row
		wantCols int
		wantRows int
		wantLine int
		wantCol  int
	}{
		{
			name: "with_data_sets_normalized_columns_and_cursor_1_1",
			columns: []Column{
				{Key: "id", Title: "ID", MinWidth: 2},
			},
			rows: []Row{
				{"id": "1"},
			},
			wantCols: 1,
			wantRows: 1,
			wantLine: 1,
			wantCol:  1,
		},
		{
			name: "with_empty_rows_sets_cursor_0_0",
			columns: []Column{
				{Key: "a", Title: "A", MinWidth: 4},
			},
			rows:     nil,
			wantCols: 1,
			wantRows: 0,
			wantLine: 0,
			wantCol:  0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(tt.columns, tt.rows)
			if m.ColumnCount() != tt.wantCols {
				t.Errorf("ColumnCount() = %d, want %d", m.ColumnCount(), tt.wantCols)
			}
			if m.RowCount() != tt.wantRows {
				t.Errorf("RowCount() = %d, want %d", m.RowCount(), tt.wantRows)
			}
			line, col := m.CursorPosition()
			if line != tt.wantLine || col != tt.wantCol {
				t.Errorf("CursorPosition() = (%d, %d), want (%d, %d)", line, col, tt.wantLine, tt.wantCol)
			}
		})
	}
}

func Test_SetSize_clamps_to_min_width_and_height(t *testing.T) {
	m := New([]Column{{Key: "a", Title: "A", MinWidth: 4}}, nil)
	m.SetSize(1, 1)
	// Model doesn't expose width/height; we only check View doesn't panic and has content
	view := m.View()
	if view == "" {
		t.Error("View() after SetSize(1,1) is empty")
	}
}

func Test_SetRows_replaces_data_and_clamps_selection(t *testing.T) {
	columns := []Column{
		{Key: "a", Title: "A", MinWidth: 4},
	}
	m := New(columns, []Row{
		{"a": "1"},
		{"a": "2"},
	})
	_, _ = m.Update(keyMsg("G")) // go to last row
	m.SetRows([]Row{
		{"a": "only"},
	})
	if m.RowCount() != 1 {
		t.Errorf("RowCount() = %d, want 1", m.RowCount())
	}
	line, _ := m.CursorPosition()
	if line != 1 {
		t.Errorf("CursorPosition() line = %d, want 1 (clamped)", line)
	}
}

func Test_SetColumns_replaces_columns_and_clamps_selection(t *testing.T) {
	columns := []Column{
		{Key: "a", Title: "A", MinWidth: 4},
		{Key: "b", Title: "B", MinWidth: 4},
	}
	m := New(columns, []Row{
		{"a": "1", "b": "2"},
	})
	_, _ = m.Update(keyMsg("l")) // move right to column 2
	m.SetColumns([]Column{
		{Key: "x", Title: "X", MinWidth: 4},
	})
	if m.ColumnCount() != 1 {
		t.Errorf("ColumnCount() = %d, want 1", m.ColumnCount())
	}
	_, col := m.CursorPosition()
	if col != 1 {
		t.Errorf("CursorPosition() col = %d, want 1 (clamped)", col)
	}
}

func Test_CursorPosition(t *testing.T) {
	columns := []Column{
		{Key: "a", Title: "A", MinWidth: 4},
	}
	tests := []struct {
		name     string
		rows     []Row
		rowIndex int
		colIndex int
		wantLine int
		wantCol  int
	}{
		{
			name:     "empty_table_returns_zero_zero",
			rows:     nil,
			rowIndex: 0,
			colIndex: 0,
			wantLine: 0,
			wantCol:  0,
		},
		{
			name: "first_cell_is_one_one",
			rows: []Row{
				{"a": "x"},
			},
			rowIndex: 0,
			colIndex: 0,
			wantLine: 1,
			wantCol:  1,
		},
		{
			name: "second_row_second_col",
			rows: []Row{
				{"a": "1"},
				{"a": "2"},
			},
			rowIndex: 1,
			colIndex: 0,
			wantLine: 2,
			wantCol:  1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(columns, tt.rows)
			if len(tt.rows) > 0 {
				for i := 0; i < tt.rowIndex; i++ {
					m, _ = m.Update(keyMsg("j"))
				}
				for i := 0; i < tt.colIndex; i++ {
					m, _ = m.Update(keyMsg("l"))
				}
			}
			line, col := m.CursorPosition()
			if line != tt.wantLine || col != tt.wantCol {
				t.Errorf("CursorPosition() = (%d, %d), want (%d, %d)", line, col, tt.wantLine, tt.wantCol)
			}
		})
	}
}

func Test_ActiveCell(t *testing.T) {
	columns := []Column{
		{Key: "a", Title: "A", MinWidth: 4},
		{Key: "b", Title: "B", MinWidth: 4},
	}
	tests := []struct {
		name      string
		rows      []Row
		moveKeys  []string
		wantKey   string
		wantValue string
		wantOK    bool
	}{
		{
			name:      "empty_rows_returns_not_ok",
			rows:      nil,
			moveKeys:  nil,
			wantKey:   "",
			wantValue: "",
			wantOK:    false,
		},
		{
			name: "first_cell",
			rows: []Row{
				{"a": "1", "b": "2"},
			},
			moveKeys:  nil,
			wantKey:   "a",
			wantValue: "1",
			wantOK:    true,
		},
		{
			name: "second_column",
			rows: []Row{
				{"a": "1", "b": "2"},
			},
			moveKeys:  []string{"l"},
			wantKey:   "b",
			wantValue: "2",
			wantOK:    true,
		},
		{
			name: "second_row",
			rows: []Row{
				{"a": "1"},
				{"a": "2", "b": "x"},
			},
			moveKeys:  []string{"j"},
			wantKey:   "a",
			wantValue: "2",
			wantOK:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(columns, tt.rows)
			for _, k := range tt.moveKeys {
				m, _ = m.Update(keyMsg(k))
			}
			col, value, ok := m.ActiveCell()
			if ok != tt.wantOK {
				t.Errorf("ActiveCell() ok = %v, want %v", ok, tt.wantOK)
			}
			if !tt.wantOK {
				return
			}
			if col.Key != tt.wantKey || value != tt.wantValue {
				t.Errorf("ActiveCell() = (%q, %q), want (%q, %q)", col.Key, value, tt.wantKey, tt.wantValue)
			}
		})
	}
}

func Test_ActiveRow(t *testing.T) {
	columns := []Column{
		{Key: "a", Title: "A", MinWidth: 4},
	}
	tests := []struct {
		name     string
		rows     []Row
		moveDown int      // number of single "j" moves (ignored if keys is set)
		keys     []string // if non-nil, use these key sequences instead of moveDown
		wantRow  Row
		wantOK   bool
	}{
		{
			name:     "empty_returns_not_ok",
			rows:     nil,
			moveDown: 0,
			wantRow:  nil,
			wantOK:   false,
		},
		{
			name: "first_row",
			rows: []Row{
				{"a": "1"},
			},
			moveDown: 0,
			wantRow:  Row{"a": "1"},
			wantOK:   true,
		},
		{
			name: "second_row",
			rows: []Row{
				{"a": "1"},
				{"a": "2"},
			},
			moveDown: 1,
			wantRow:  Row{"a": "2"},
			wantOK:   true,
		},
		{
			name: "row_after_count_prefix_move",
			rows: []Row{
				{"a": "1"},
				{"a": "2"},
				{"a": "3"},
				{"a": "4"},
				{"a": "5"},
			},
			keys:    []string{"2", "j"},
			wantRow: Row{"a": "3"},
			wantOK:  true,
		},
		{
			name: "row_after_count_prefix_clamped_to_last",
			rows: []Row{
				{"a": "1"},
				{"a": "2"},
				{"a": "3"},
			},
			keys:    []string{"1", "0", "j"}, // two key presses "1" then "0" then "j"
			wantRow: Row{"a": "3"},
			wantOK:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(columns, tt.rows)
			if tt.keys != nil {
				for _, k := range tt.keys {
					m, _ = m.Update(keyMsg(k))
				}
			} else {
				for i := 0; i < tt.moveDown; i++ {
					m, _ = m.Update(keyMsg("j"))
				}
			}
			row, ok := m.ActiveRow()
			if ok != tt.wantOK {
				t.Errorf("ActiveRow() ok = %v, want %v", ok, tt.wantOK)
			}
			if !tt.wantOK {
				return
			}
			if !maps.Equal(row, tt.wantRow) {
				t.Errorf("ActiveRow() = %v, want %v", row, tt.wantRow)
			}
		})
	}
}

func Test_VisibleColumnCount(t *testing.T) {
	// visibleColumns uses: space = width - (rowNumberWidth + columnGapWidth).
	// With 0 rows, rowNumberWidth=2, columnGapWidth=1 → space = width - 3.
	// Each column consumes MinWidth + columnGapWidth.
	tests := []struct {
		name        string
		width       int
		columns     []Column
		rows        []Row
		wantVisible int
	}{
		{
			name:  "wide_viewport_three_cols_all_visible",
			width: 40,
			columns: []Column{
				{Key: "a", Title: "A", MinWidth: 10},
				{Key: "b", Title: "B", MinWidth: 10},
				{Key: "c", Title: "C", MinWidth: 10},
			},
			rows:        nil,
			wantVisible: 3,
		},
		{
			name:  "narrow_viewport_two_of_three_visible",
			width: 25,
			columns: []Column{
				{Key: "a", Title: "A", MinWidth: 10},
				{Key: "b", Title: "B", MinWidth: 10},
				{Key: "c", Title: "C", MinWidth: 10},
			},
			rows:        nil,
			wantVisible: 2,
		},
		{
			name:  "very_narrow_one_column_visible",
			width: 14,
			columns: []Column{
				{Key: "a", Title: "A", MinWidth: 10},
				{Key: "b", Title: "B", MinWidth: 10},
				{Key: "c", Title: "C", MinWidth: 10},
			},
			rows:        nil,
			wantVisible: 1,
		},
		{
			name:  "single_column_always_one_visible",
			width: 80,
			columns: []Column{
				{Key: "a", Title: "A", MinWidth: 4},
			},
			rows:        nil,
			wantVisible: 1,
		},
		{
			name:  "five_narrow_columns_all_visible",
			width: 30,
			columns: []Column{
				{Key: "a", Title: "A", MinWidth: 4},
				{Key: "b", Title: "B", MinWidth: 4},
				{Key: "c", Title: "C", MinWidth: 4},
				{Key: "d", Title: "D", MinWidth: 4},
				{Key: "e", Title: "E", MinWidth: 4},
			},
			rows:        nil,
			wantVisible: 5,
		},
		{
			name:  "five_columns_four_visible",
			width: 25,
			columns: []Column{
				{Key: "a", Title: "A", MinWidth: 4},
				{Key: "b", Title: "B", MinWidth: 4},
				{Key: "c", Title: "C", MinWidth: 4},
				{Key: "d", Title: "D", MinWidth: 4},
				{Key: "e", Title: "E", MinWidth: 4},
			},
			rows:        nil,
			wantVisible: 4,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(tt.columns, tt.rows)
			m.SetSize(tt.width, 10)
			got := m.VisibleColumnCount()
			if got != tt.wantVisible {
				t.Errorf("VisibleColumnCount() = %d, want %d (width=%d)", got, tt.wantVisible, tt.width)
			}
		})
	}
}

func Test_Update_key_movement(t *testing.T) {
	columns := []Column{
		{Key: "a", Title: "A", MinWidth: 4},
		{Key: "b", Title: "B", MinWidth: 4},
	}
	rows := []Row{
		{"a": "1", "b": "2"},
		{"a": "3", "b": "4"},
	}
	tests := []struct {
		name     string
		keys     []string
		wantLine int
		wantCol  int
	}{
		{
			name:     "down_j",
			keys:     []string{"j"},
			wantLine: 2,
			wantCol:  1,
		},
		{
			name:     "up_k",
			keys:     []string{"j", "k"},
			wantLine: 1,
			wantCol:  1,
		},
		{
			name:     "right_l",
			keys:     []string{"l"},
			wantLine: 1,
			wantCol:  2,
		},
		{
			name:     "left_h",
			keys:     []string{"l", "h"},
			wantLine: 1,
			wantCol:  1,
		},
		{
			name:     "goto_bottom_G",
			keys:     []string{"G"},
			wantLine: 2,
			wantCol:  1,
		},
		{
			name:     "goto_top_g",
			keys:     []string{"G", "g"},
			wantLine: 1,
			wantCol:  1,
		},
		{
			name:     "goto_top_home",
			keys:     []string{"G", "home"},
			wantLine: 1,
			wantCol:  1,
		},
		{
			name:     "goto_bottom_end",
			keys:     []string{"end"},
			wantLine: 2,
			wantCol:  1,
		},
		{
			name:     "move_up_at_top_stays",
			keys:     []string{"k", "k"},
			wantLine: 1,
			wantCol:  1,
		},
		{
			name:     "move_down_at_bottom_stays",
			keys:     []string{"G", "j", "j"},
			wantLine: 2,
			wantCol:  1,
		},
		{
			name:     "move_left_at_start_stays",
			keys:     []string{"h", "h"},
			wantLine: 1,
			wantCol:  1,
		},
		{
			name:     "count_prefix_4j_moves_down_clamped",
			keys:     []string{"4", "j"},
			wantLine: 2,
			wantCol:  1,
		},
		{
			name:     "count_prefix_3l_moves_right_clamped",
			keys:     []string{"3", "l"},
			wantLine: 1,
			wantCol:  2,
		},
		{
			name:     "count_prefix_2k_from_second_row",
			keys:     []string{"j", "2", "k"},
			wantLine: 1,
			wantCol:  1,
		},
		{
			name:     "non_motion_clears_buffer_then_j_moves_one",
			keys:     []string{"4", "g", "j"},
			wantLine: 2,
			wantCol:  1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(columns, rows)
			for _, k := range tt.keys {
				m, _ = m.Update(keyMsg(k))
			}
			line, col := m.CursorPosition()
			if line != tt.wantLine || col != tt.wantCol {
				t.Errorf("after keys %v CursorPosition() = (%d, %d), want (%d, %d)", tt.keys, line, col, tt.wantLine, tt.wantCol)
			}
		})
	}
}

func Test_Update_non_key_message_returns_unchanged_model(t *testing.T) {
	columns := []Column{
		{Key: "a", Title: "A", MinWidth: 4},
	}
	m := New(columns, []Row{
		{"a": "1"},
	})
	m2, cmd := m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	if m2.RowCount() != m.RowCount() || m2.ColumnCount() != m.ColumnCount() {
		t.Error("Update(non-KeyMsg) should leave model unchanged")
	}
	if cmd != nil {
		t.Error("Update(non-KeyMsg) should return nil cmd")
	}
	_ = cmd
}

func Test_Columns_returns_copy_not_internal_slice(t *testing.T) {
	columns := []Column{
		{Key: "a", Title: "A", MinWidth: 4},
	}
	m := New(columns, nil)
	got := m.Columns()
	got[0].Title = "mutated"
	if m.Columns()[0].Title == "mutated" {
		t.Error("Columns() should return a copy; mutating it mutated the model")
	}
}

func Test_View_returns_non_empty_and_has_header_and_body_lines(t *testing.T) {
	columns := []Column{
		{Key: "a", Title: "A", MinWidth: 4},
	}
	rows := []Row{
		{"a": "1"},
	}
	m := New(columns, rows)
	view := m.View()
	if view == "" {
		t.Error("View() returned empty string")
	}
	lines := 0
	for _, r := range view {
		if r == '\n' {
			lines++
		}
	}
	if lines < 1 {
		t.Error("View() should contain at least one newline (header + body)")
	}
}

func Test_HelpBindings_returns_non_empty_bindings(t *testing.T) {
	bindings := HelpBindings()
	if len(bindings) == 0 {
		t.Error("HelpBindings() returned empty slice")
	}
}
