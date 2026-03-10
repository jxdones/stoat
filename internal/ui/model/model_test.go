package model

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/jxdones/stoat/internal/database"
)

// keyMsg returns a tea.KeyPressMsg for testing. s must match msg.String() in model.Update (e.g. "q", "tab", "shift+tab", "ctrl+c").
func keyMsg(s string) tea.KeyPressMsg {
	switch s {
	case "q":
		return tea.KeyPressMsg(tea.Key{Code: 'q', Text: "q"})
	case "esc":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc})
	case "tab":
		// KeyTab.String() returns "\t", not "tab"; model expects "tab". Use for focusNext tests only.
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyTab, Text: "\t"})
	case "shift+tab":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyTab, Mod: tea.ModShift})
	case "ctrl+c":
		return tea.KeyPressMsg(tea.Key{Code: 'c', Mod: tea.ModCtrl})
	case "ctrl+s":
		return tea.KeyPressMsg(tea.Key{Code: 's', Mod: tea.ModCtrl})
	case "enter":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})
	default:
		return tea.KeyPressMsg(tea.Key{Code: rune(s[0]), Text: s})
	}
}

type mockDataSource struct {
	queryFn func(ctx context.Context, query string) (database.QueryResult, error)
}

func (m mockDataSource) Databases(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (m mockDataSource) Tables(ctx context.Context, databaseName string) ([]string, error) {
	return nil, nil
}

func (m mockDataSource) Rows(ctx context.Context, target database.DatabaseTarget, page database.PageRequest) (database.PageResult, error) {
	return database.PageResult{}, nil
}

func (m mockDataSource) Query(ctx context.Context, query string) (database.QueryResult, error) {
	if m.queryFn != nil {
		return m.queryFn(ctx, query)
	}
	return database.QueryResult{}, nil
}

func (m mockDataSource) Indexes(ctx context.Context, target database.DatabaseTarget) ([]database.Index, error) {
	return nil, nil
}

func (m mockDataSource) Constraints(ctx context.Context, target database.DatabaseTarget) ([]database.Constraint, error) {
	return nil, nil
}

func (m mockDataSource) ForeignKeys(ctx context.Context, target database.DatabaseTarget) ([]database.ForeignKey, error) {
	return nil, nil
}

func (m mockDataSource) Close() error {
	return nil
}

func TestView_Smoke(t *testing.T) {
	m := New()
	m.view.width = 80
	m.view.height = 24
	view := m.View()
	if view.Content == "" {
		t.Error("View() returned empty string")
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name       string
		checkFocus bool
		checkSize  bool
	}{
		{
			name:       "initial_focus_is_sidebar",
			checkFocus: true,
			checkSize:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			if tt.checkFocus && m.view.focus != FocusSidebar {
				t.Errorf("New() focus = %v, want FocusSidebar", m.view.focus)
			}
			if tt.checkSize && (m.view.width != 80 || m.view.height != 24) {
				t.Errorf("New() default size = %dx%d, want 80x24", m.view.width, m.view.height)
			}
		})
	}
}

func TestInit(t *testing.T) {
	tests := []struct {
		name    string
		wantNil bool
	}{
		{
			name:    "returns_nil_cmd",
			wantNil: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			cmd := m.Init()
			if tt.wantNil && cmd != nil {
				t.Error("Init() should return nil Cmd")
			}
		})
	}
}

func TestIsFocused(t *testing.T) {
	tests := []struct {
		name  string
		focus FocusedPanel
		panel FocusedPanel
		want  bool
	}{
		{
			name:  "sidebar_focused_when_focus_sidebar",
			focus: FocusSidebar,
			panel: FocusSidebar,
			want:  true,
		},
		{
			name:  "sidebar_not_focused_when_focus_filterbox",
			focus: FocusFilterbox,
			panel: FocusSidebar,
			want:  false,
		},
		{
			name:  "table_focused_when_focus_table",
			focus: FocusTable,
			panel: FocusTable,
			want:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			m.view.focus = tt.focus
			got := m.isFocused(tt.panel)
			if got != tt.want {
				t.Errorf("isFocused(%v) with focus %v = %v, want %v", tt.panel, tt.focus, got, tt.want)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	tests := []struct {
		name        string
		msg         tea.Msg
		setup       func(*Model)
		wantWidth   int
		wantHeight  int
		wantQuitCmd bool
		wantFocus   *FocusedPanel
	}{
		{
			name:       "WindowSizeMsg_updates_dimensions",
			msg:        tea.WindowSizeMsg{Width: 100, Height: 30},
			wantWidth:  100,
			wantHeight: 30,
		},
		{
			name: "key_q_returns_quit_cmd_when_no_focus",
			msg:  keyMsg("q"),
			setup: func(m *Model) {
				m.view.focus = FocusNone
			},
			wantQuitCmd: true,
		},
		{
			name: "key_q_does_not_quit_when_filter_focused",
			msg:  keyMsg("q"),
			setup: func(m *Model) {
				m.view.focus = FocusFilterbox
			},
		},
		{
			name: "key_q_does_not_quit_when_table_focused",
			msg:  keyMsg("q"),
			setup: func(m *Model) {
				m.view.focus = FocusTable
			},
		},
		{
			name: "key_q_does_not_quit_when_query_focused",
			msg:  keyMsg("q"),
			setup: func(m *Model) {
				m.view.focus = FocusQuerybox
			},
		},
		{
			name:        "key_ctrl_c_returns_quit_cmd",
			msg:         keyMsg("ctrl+c"),
			wantQuitCmd: true,
		},
		{
			name: "shift_tab_from_table_cycles_focus_backward",
			msg:  keyMsg("shift+tab"),
			setup: func(m *Model) {
				m.view.focus = FocusTable
			},
			wantFocus: ptrFocus(FocusFilterbox),
		},
		{
			name:       "WindowSizeMsg_returns_nil_cmd",
			msg:        tea.WindowSizeMsg{Width: 80, Height: 24},
			wantWidth:  80,
			wantHeight: 24,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			if tt.setup != nil {
				tt.setup(&m)
			}
			next, cmd := m.Update(tt.msg)
			got := next.(Model)
			if tt.wantQuitCmd {
				if cmd == nil {
					t.Error("Update() should return non-nil Quit Cmd")
				}
				return
			}
			if cmd != nil && !tt.wantQuitCmd {
				t.Errorf("Update() should return nil Cmd, got %v", cmd)
			}
			if tt.wantWidth != 0 && got.view.width != tt.wantWidth {
				t.Errorf("view.width = %d, want %d", got.view.width, tt.wantWidth)
			}
			if tt.wantHeight != 0 && got.view.height != tt.wantHeight {
				t.Errorf("view.height = %d, want %d", got.view.height, tt.wantHeight)
			}
			if tt.wantFocus != nil && got.view.focus != *tt.wantFocus {
				t.Errorf("view.focus = %v, want %v", got.view.focus, *tt.wantFocus)
			}
		})
	}
}

func ptrFocus(p FocusedPanel) *FocusedPanel {
	return &p
}

func TestUpdate_PasteMsg(t *testing.T) {
	tests := []struct {
		name         string
		focus        FocusedPanel
		initialValue string
		pasteContent string
		wantValue    string
		wantNilCmd   bool
	}{
		{
			name:         "forwarded_to_querybox_when_focused",
			focus:        FocusQuerybox,
			initialValue: "",
			pasteContent: "SELECT 1",
			wantValue:    "SELECT 1",
			wantNilCmd:   false, // textarea may return a cmd (e.g. blink)
		},
		{
			name:         "ignored_when_querybox_not_focused",
			focus:        FocusSidebar,
			initialValue: "existing",
			pasteContent: "pasted",
			wantValue:    "existing",
			wantNilCmd:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			m.view.focus = tt.focus
			m.querybox.SetValue(tt.initialValue)
			if tt.focus == FocusQuerybox {
				m.querybox.Focus()
			}

			next, cmd := m.Update(tea.PasteMsg{Content: tt.pasteContent})
			got := next.(Model)
			if got.querybox.Value() != tt.wantValue {
				t.Errorf("querybox value = %q, want %q", got.querybox.Value(), tt.wantValue)
			}
			if tt.wantNilCmd && cmd != nil {
				t.Errorf("Update(PasteMsg) want nil cmd, got %v", cmd)
			}
		})
	}
}

func TestUpdate_EscClearsFocusAndQQuits(t *testing.T) {
	m := New()
	m.view.focus = FocusQuerybox

	next, cmd := m.Update(keyMsg("esc"))
	if cmd != nil {
		t.Fatalf("Update(esc) should return nil cmd, got %v", cmd)
	}
	afterEsc := next.(Model)
	if afterEsc.view.focus != FocusNone {
		t.Fatalf("focus after esc = %v, want %v", afterEsc.view.focus, FocusNone)
	}

	_, cmd = afterEsc.Update(keyMsg("q"))
	if cmd == nil {
		t.Fatal("Update(q) should return non-nil quit cmd when focus is none")
	}
}

func TestUpdateFocused_sidebarEnter(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*Model)
		wantFocus  FocusedPanel
		wantOpenDB bool
	}{
		{
			name: "enter_on_db_list_opens_database",
			setup: func(m *Model) {
				m.sidebar.SetDatabases([]string{"mydb"})
				m.view.focus = FocusSidebar
			},
			wantFocus:  FocusSidebar,
			wantOpenDB: true,
		},
		{
			name: "enter_on_tables_section_moves_focus_to_table",
			setup: func(m *Model) {
				m.sidebar.SetDatabases([]string{"mydb"})
				m.sidebar.SetTables("mydb", []string{"users"})
				m.sidebar.OpenSelectedDatabase()
				m.view.focus = FocusSidebar
			},
			wantFocus: FocusTable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			if tt.setup != nil {
				tt.setup(&m)
			}
			msg := keyMsg("enter")
			next, cmd := m.Update(msg)
			if cmd != nil {
				t.Errorf("Update(Enter) should return nil Cmd, got %v", cmd)
			}
			got := next.(Model)
			if got.view.focus != tt.wantFocus {
				t.Errorf("after Enter: focus = %v, want %v", got.view.focus, tt.wantFocus)
			}
			if tt.wantOpenDB && got.sidebar.ActiveDB() == "" {
				t.Error("expected database to be opened")
			}
		})
	}
}

func TestFocusNext(t *testing.T) {
	tests := []struct {
		name     string
		initial  FocusedPanel
		wantNext FocusedPanel
	}{
		{
			name:     "sidebar_to_filterbox",
			initial:  FocusSidebar,
			wantNext: FocusFilterbox,
		},
		{
			name:     "filterbox_to_table",
			initial:  FocusFilterbox,
			wantNext: FocusTable,
		},
		{
			name:     "table_to_querybox",
			initial:  FocusTable,
			wantNext: FocusQuerybox,
		},
		{
			name:     "querybox_to_sidebar",
			initial:  FocusQuerybox,
			wantNext: FocusSidebar,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			m.view.focus = tt.initial
			m.focusNext()
			if m.view.focus != tt.wantNext {
				t.Errorf("focusNext() from %v: focus = %v, want %v", tt.initial, m.view.focus, tt.wantNext)
			}
		})
	}
}

func TestFocusPrevious(t *testing.T) {
	tests := []struct {
		name     string
		initial  FocusedPanel
		wantPrev FocusedPanel
	}{
		{
			name:     "sidebar_to_querybox",
			initial:  FocusSidebar,
			wantPrev: FocusQuerybox,
		},

		{
			name:     "querybox_to_table",
			initial:  FocusQuerybox,
			wantPrev: FocusTable,
		},
		{
			name:     "table_to_filterbox",
			initial:  FocusTable,
			wantPrev: FocusFilterbox,
		},
		{
			name:     "filterbox_to_sidebar",
			initial:  FocusFilterbox,
			wantPrev: FocusSidebar,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			m.view.focus = tt.initial
			m.focusPrevious()
			if m.view.focus != tt.wantPrev {
				t.Errorf("focusPrevious() from %v: focus = %v, want %v", tt.initial, m.view.focus, tt.wantPrev)
			}
		})
	}
}

func TestPagingStateDefaults(t *testing.T) {
	m := New()
	if m.paging.currentAfter != "" {
		t.Fatalf("currentAfter = %q, want empty", m.paging.currentAfter)
	}
	if m.paging.currentHasMore {
		t.Fatalf("currentHasMore = true, want false")
	}
	if len(m.paging.afterStack) != 1 || m.paging.afterStack[0] != "" {
		t.Fatalf("afterStack = %v, want [\"\"]", m.paging.afterStack)
	}
	if m.paging.pendingNav != pageNavNone {
		t.Fatalf("pendingNav = %v, want pageNavNone", m.paging.pendingNav)
	}
}

func TestApplyPageResult(t *testing.T) {
	tests := []struct {
		name               string
		setup              func(*Model)
		nav                pageNav
		startAfter         string
		nextAfter          string
		hasMore            bool
		wantAfterStack     []string
		wantCurrentAfter   string
		wantCurrentHasMore bool
	}{
		{
			name: "next_navigation_pushes_start_cursor",
			setup: func(m *Model) {
				m.paging.afterStack = []string{""}
			},
			nav:                pageNavNext,
			startAfter:         "rowid:100",
			nextAfter:          "rowid:200",
			hasMore:            true,
			wantAfterStack:     []string{"", "rowid:100"},
			wantCurrentAfter:   "rowid:200",
			wantCurrentHasMore: true,
		},
		{
			name: "prev_navigation_pops_history",
			setup: func(m *Model) {
				m.paging.afterStack = []string{"", "rowid:100", "rowid:200"}
			},
			nav:                pageNavPrev,
			startAfter:         "rowid:100",
			nextAfter:          "rowid:200",
			hasMore:            true,
			wantAfterStack:     []string{"", "rowid:100"},
			wantCurrentAfter:   "rowid:200",
			wantCurrentHasMore: true,
		},
		{
			name: "none_navigation_resets_stack_to_start_cursor",
			setup: func(m *Model) {
				m.paging.afterStack = []string{"", "rowid:100"}
			},
			nav:                pageNavNone,
			startAfter:         "rowid:100",
			nextAfter:          "rowid:200",
			hasMore:            false,
			wantAfterStack:     []string{"rowid:100"},
			wantCurrentAfter:   "rowid:200",
			wantCurrentHasMore: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			if tt.setup != nil {
				tt.setup(&m)
			}
			m.setPendingPageNav(tt.nav)
			m.applyPageResult(tt.startAfter, tt.nextAfter, tt.hasMore)

			if len(m.paging.afterStack) != len(tt.wantAfterStack) {
				t.Fatalf("afterStack len = %d, want %d (%v)", len(m.paging.afterStack), len(tt.wantAfterStack), m.paging.afterStack)
			}
			for i := range tt.wantAfterStack {
				if m.paging.afterStack[i] != tt.wantAfterStack[i] {
					t.Fatalf("afterStack[%d] = %q, want %q", i, m.paging.afterStack[i], tt.wantAfterStack[i])
				}
			}
			if m.paging.currentAfter != tt.wantCurrentAfter {
				t.Fatalf("currentAfter = %q, want %q", m.paging.currentAfter, tt.wantCurrentAfter)
			}
			if m.paging.currentHasMore != tt.wantCurrentHasMore {
				t.Fatalf("currentHasMore = %v, want %v", m.paging.currentHasMore, tt.wantCurrentHasMore)
			}
			if m.paging.pendingNav != pageNavNone {
				t.Fatalf("pendingNav = %v, want %v", m.paging.pendingNav, pageNavNone)
			}
		})
	}
}

func TestResetPaging(t *testing.T) {
	m := New()
	m.paging.currentAfter = "rowid:999"
	m.paging.currentHasMore = true
	m.paging.afterStack = []string{"", "rowid:100"}
	m.paging.pendingNav = pageNavNext

	m.resetPaging()

	if m.paging.currentAfter != "" || m.paging.currentHasMore || m.paging.pendingNav != pageNavNone {
		t.Fatalf("unexpected reset paging state: %+v", m.paging)
	}
	if len(m.paging.afterStack) != 1 || m.paging.afterStack[0] != "" {
		t.Fatalf("afterStack = %v, want [\"\"]", m.paging.afterStack)
	}
}

func TestCtrlSInQuerybox(t *testing.T) {
	tests := []struct {
		name               string
		query              string
		source             mockDataSource
		withConnection     bool
		wantCmd            bool
		wantQueryCmd       bool
		wantStatusContains string
		wantColumns        int
		wantRows           int
		wantCleared        bool
	}{
		{
			name:               "no_connection_sets_warning",
			query:              "select 1",
			withConnection:     false,
			wantCmd:            true,
			wantQueryCmd:       false,
			wantStatusContains: "No active connection",
			wantCleared:        false,
		},
		{
			name:               "empty_query_sets_warning",
			query:              "   \n\t ",
			withConnection:     true,
			source:             mockDataSource{},
			wantCmd:            true,
			wantQueryCmd:       false,
			wantStatusContains: "Query is empty",
			wantCleared:        false,
		},
		{
			name:           "executes_query_and_updates_table",
			query:          "select id, name from users",
			withConnection: true,
			source: mockDataSource{
				queryFn: func(ctx context.Context, query string) (database.QueryResult, error) {
					return database.QueryResult{
						Columns: []database.Column{
							{Key: "id", Title: "id", Type: "int", MinWidth: 4, Order: 1},
							{Key: "name", Title: "name", Type: "text", MinWidth: 8, Order: 2},
						},
						Rows: []database.Row{
							{"id": "1", "name": "alice"},
						},
						RowsAffected: 0,
					}, nil
				},
			},
			wantCmd:            true,
			wantQueryCmd:       true,
			wantStatusContains: "Query ok: 1 row(s) returned",
			wantColumns:        2,
			wantRows:           1,
			wantCleared:        true,
		},
		{
			name:           "query_error_sets_error_status",
			query:          "bad query",
			withConnection: true,
			source: mockDataSource{
				queryFn: func(ctx context.Context, query string) (database.QueryResult, error) {
					return database.QueryResult{}, errors.New("syntax error")
				},
			},
			wantCmd:            true,
			wantQueryCmd:       true,
			wantStatusContains: "Query failed: syntax error",
			wantCleared:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			m.view.focus = FocusQuerybox
			m.querybox.SetValue(tt.query)
			if tt.withConnection {
				m.SetDataSource(tt.source)
			}

			next, cmd := m.Update(keyMsg("ctrl+s"))
			if tt.wantCmd && cmd == nil {
				t.Fatal("expected non-nil command")
			}
			if !tt.wantCmd && cmd != nil {
				t.Fatalf("expected nil command, got %v", cmd)
			}

			if cmd != nil && tt.wantQueryCmd {
				msg := cmd()
				runMsg, ok := msg.(QueryRunRequestedMsg)
				if !ok {
					t.Fatalf("expected QueryRunRequestedMsg, got %T", msg)
				}
				nextModel, runCmd := next.(Model).Update(runMsg)
				if runCmd == nil {
					t.Fatal("expected non-nil run command")
				}
				execMsg := runCmd()
				queryMsg, ok := execMsg.(QueryExecutedMsg)
				if !ok {
					t.Fatalf("expected QueryExecutedMsg, got %T", execMsg)
				}
				next, _ = nextModel.(Model).Update(queryMsg)
			}

			got := next.(Model)
			if tt.wantColumns > 0 && got.table.ColumnCount() != tt.wantColumns {
				t.Fatalf("table columns = %d, want %d", got.table.ColumnCount(), tt.wantColumns)
			}
			if tt.wantRows > 0 && got.table.RowCount() != tt.wantRows {
				t.Fatalf("table rows = %d, want %d", got.table.RowCount(), tt.wantRows)
			}
			if tt.wantStatusContains != "" && !strings.Contains(got.statusbar.View(120).Content, tt.wantStatusContains) {
				t.Fatalf("expected status to contain %q", tt.wantStatusContains)
			}
			if tt.wantCleared && got.querybox.Value() != "" {
				t.Fatalf("querybox value = %q, want empty after successful query", got.querybox.Value())
			}
			if !tt.wantCleared && got.querybox.Value() == "" {
				t.Fatal("querybox value was cleared unexpectedly")
			}
		})
	}
}
