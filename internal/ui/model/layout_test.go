package model

import "testing"

func TestLayout_Smoke(t *testing.T) {
	layout := computeLayout(80, 24)
	if layout.columns.leftPane != 18 {
		t.Errorf("leftPane = %d, want 18", layout.columns.leftPane)
	}
}

func TestComputeColumns(t *testing.T) {
	tests := []struct {
		name        string
		width       int
		wantColumns layoutColumns
	}{
		{
			name:  "zero_width",
			width: 0,
			wantColumns: layoutColumns{
				leftPane: 0,
				mainPane: 0,
			},
		},
		{
			name:  "narrow_terminal",
			width: 50,
			wantColumns: layoutColumns{
				leftPane: 18, // minLeftPane is 18
				mainPane: 32,
			},
		},
		{
			name:  "wide_terminal",
			width: 100,
			wantColumns: layoutColumns{
				leftPane: 20,
				mainPane: 80,
			},
		},
		{
			name:  "very_small_width",
			width: 20,
			wantColumns: layoutColumns{
				leftPane: 0,
				mainPane: 20,
			},
		},
		{
			name:  "very_large_width",
			width: 1000,
			wantColumns: layoutColumns{
				leftPane: 200,
				mainPane: 800,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			columns := computeColumns(tt.width)
			if columns.leftPane != tt.wantColumns.leftPane {
				t.Errorf("leftPane = %d, want %d", columns.leftPane, tt.wantColumns.leftPane)
			}
			if columns.mainPane != tt.wantColumns.mainPane {
				t.Errorf("mainPane = %d, want %d", columns.mainPane, tt.wantColumns.mainPane)
			}
		})
	}
}

func TestComputeRows(t *testing.T) {
	tests := []struct {
		name     string
		height   int
		wantRows layoutRows
	}{
		{
			name:   "zero_height",
			height: 0,
			wantRows: layoutRows{
				mainContent: 0,
				statusRow:   0,
				optionsRow:  0,
			},
		},
		{
			name:   "small_height",
			height: 10,
			wantRows: layoutRows{
				mainContent: 7,
				statusRow:   1,
				optionsRow:  2,
			},
		},
		{
			name:   "large_height",
			height: 100,
			wantRows: layoutRows{
				mainContent: 97,
				statusRow:   1,
				optionsRow:  2,
			},
		},
		{
			name:   "very_small_height",
			height: 3,
			wantRows: layoutRows{
				mainContent: 2,
				statusRow:   0,
				optionsRow:  2,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows := computeRows(tt.height)
			if rows.mainContent != tt.wantRows.mainContent {
				t.Errorf("mainContent = %d, want %d", rows.mainContent, tt.wantRows.mainContent)
			}
			if rows.statusRow != tt.wantRows.statusRow {
				t.Errorf("statusRow = %d, want %d", rows.statusRow, tt.wantRows.statusRow)
			}
			if rows.optionsRow != tt.wantRows.optionsRow {
				t.Errorf("optionsRow = %d, want %d", rows.optionsRow, tt.wantRows.optionsRow)
			}
		})
	}
}

func TestComputeMainSections(t *testing.T) {
	tests := []struct {
		name         string
		height       int
		wantSections mainSections
	}{
		{
			name:   "tall_enough_table_gets_remaining",
			height: 20,
			wantSections: mainSections{
				header: mainHeaderRows,
				tabs:   mainTabsRows,
				detail: mainDetailRows,
				query:  mainQueryRows,
				table:  8,
			},
		},
		{
			name:   "exact_minimum_for_table_no_stealing",
			height: 13,
			wantSections: mainSections{
				header: mainHeaderRows,
				tabs:   mainTabsRows,
				detail: mainDetailRows,
				query:  mainQueryRows,
				table:  1,
			},
		},
		{
			name:   "steal_from_query_only",
			height: 12,
			wantSections: mainSections{
				header: mainHeaderRows,
				tabs:   mainTabsRows,
				detail: mainDetailRows,
				query:  2,
				table:  1,
			},
		},
		{
			name:   "steal_from_query_and_header",
			height: 10,
			wantSections: mainSections{
				header: 4,
				tabs:   mainTabsRows,
				detail: mainDetailRows,
				query:  1,
				table:  1,
			},
		},
		{
			name:   "very_short_steal_query_then_header_table_min_one",
			height: 7,
			wantSections: mainSections{
				header: 1,
				tabs:   mainTabsRows,
				detail: mainDetailRows,
				query:  1,
				table:  1,
			},
		},
		{
			name:   "zero_height_no_panic_table_clamped_to_one",
			height: 0,
			wantSections: mainSections{
				header: 1,
				tabs:   mainTabsRows,
				detail: mainDetailRows,
				query:  1,
				table:  1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeMainSections(tt.height)
			if got.header != tt.wantSections.header {
				t.Errorf("header = %d, want %d", got.header, tt.wantSections.header)
			}
			if got.tabs != tt.wantSections.tabs {
				t.Errorf("tabs = %d, want %d", got.tabs, tt.wantSections.tabs)
			}
			if got.detail != tt.wantSections.detail {
				t.Errorf("detail = %d, want %d", got.detail, tt.wantSections.detail)
			}
			if got.query != tt.wantSections.query {
				t.Errorf("query = %d, want %d", got.query, tt.wantSections.query)
			}
			if got.table != tt.wantSections.table {
				t.Errorf("table = %d, want %d", got.table, tt.wantSections.table)
			}
		})
	}
}

func TestClampRange(t *testing.T) {
	tests := []struct {
		name  string
		value int
		min   int
		max   int
		want  int
	}{
		{
			name:  "value_below_min_returns_min",
			value: 5,
			min:   10,
			max:   20,
			want:  10,
		},
		{
			name:  "value_above_max_returns_max",
			value: 25,
			min:   10,
			max:   20,
			want:  20,
		},
		{
			name:  "value_between_min_and_max_returns_value",
			value: 15,
			min:   10,
			max:   20,
			want:  15,
		},
		{
			name:  "value_equals_min_returns_min",
			value: 10,
			min:   10,
			max:   20,
			want:  10,
		},
		{
			name:  "value_equals_max_returns_max",
			value: 20,
			min:   10,
			max:   20,
			want:  20,
		},
		{
			name:  "min_equals_max_value_equals_returns_value",
			value: 7,
			min:   7,
			max:   7,
			want:  7,
		},
		{
			name:  "value_less_than_min_returns_min",
			value: 16,
			min:   18,
			max:   22,
			want:  18,
		},
		{
			name:  "min_equals_max_value_above_returns_max",
			value: 9,
			min:   7,
			max:   7,
			want:  7,
		},
		{
			name:  "negative_range",
			value: -5,
			min:   -10,
			max:   0,
			want:  -5,
		},
		{
			name:  "value_below_negative_min",
			value: -15,
			min:   -10,
			max:   0,
			want:  -10,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := clampRange(tt.value, tt.min, tt.max)
			if got != tt.want {
				t.Errorf("clampRange(%d, %d, %d) = %d, want %d", tt.value, tt.min, tt.max, got, tt.want)
			}
		})
	}
}
