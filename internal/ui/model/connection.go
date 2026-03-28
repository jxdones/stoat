package model

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/jxdones/stoat/internal/config"
	"github.com/jxdones/stoat/internal/database"
	"github.com/jxdones/stoat/internal/database/provider"
	"github.com/jxdones/stoat/internal/ui/components/connectionpicker"
	"github.com/jxdones/stoat/internal/ui/components/statusbar"
	"github.com/jxdones/stoat/internal/ui/datasource"
)

// ConnectingMsg is sent immediately when an async connection is requested.
// Handling it in Update allows the UI to render "Connecting…" before the
// blocking provider.FromConfig call begins.
type ConnectingMsg struct {
	cfg database.Config
}

// ConnectedMsg is sent when the async database connection succeeds.
type ConnectedMsg struct {
	source   datasource.DataSource
	name     string
	readOnly bool
}

// ConnectionFailedMsg is sent when the async database connection fails.
type ConnectionFailedMsg struct {
	err error
}

// handleConnectionPickerKey handles key presses while the connection picker modal is active.
func (m Model) handleConnectionPickerKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	next, event := m.connectionPicker.Update(msg)
	m.connectionPicker = next

	switch event {
	case connectionpicker.EventSelected:
		selected := m.connectionPicker.Selected()
		cfg := database.Config{Name: selected.Name, ReadOnly: selected.ReadOnly}
		cfg.ReadOnly = selected.ReadOnly || m.forceReadOnly
		switch database.DBMS(strings.ToLower(selected.Type)) {
		case database.DBMSPostgres:
			port := selected.Port
			if port == 0 {
				port = config.DefaultPostgresPort
			}
			sslmode := selected.SSLMode
			if sslmode == "" {
				sslmode = "disable"
			}
			dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
				selected.User, selected.Password, selected.Host, port, selected.Database, sslmode)
			cfg.DBMS = database.DBMSPostgres
			cfg.Values = map[string]string{"dsn": dsn}
		case database.DBMSMySQL:
			port := selected.Port
			if port == 0 {
				port = config.DefaultMySQLPort
			}
			tlsMode := selected.TLSMode
			if tlsMode == "" {
				tlsMode = "false"
			}
			dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?tls=%s",
				selected.User, selected.Password, selected.Host, port, selected.Database, tlsMode)
			cfg.DBMS = database.DBMSMySQL
			cfg.Values = map[string]string{"dsn": dsn, "database": selected.Database}
		default:
			cfg.DBMS = database.DBMSSQLite
			cfg.Values = map[string]string{"path": selected.Path}
		}
		m.SetPendingConfig(cfg)
		m.view.focus = FocusSidebar
		m.activeModal = modalNone
		return m, func() tea.Msg { return ConnectingMsg{cfg: cfg} }
	case connectionpicker.EventClosed:
		m.activeModal = modalNone
	}

	return m, nil
}

// onConnecting sets the status bar to "Connecting…" and fires the async
// ConnectCmd. The extra message hop gives the UI one render cycle to paint
// the status before the blocking provider call starts.
func (m Model) onConnecting(msg ConnectingMsg) (tea.Model, tea.Cmd) {
	spinnerCmd := m.statusbar.StartSpinner("Connecting to "+msg.cfg.Name, statusbar.Info)
	return m, tea.Batch(spinnerCmd, ConnectCmd(msg.cfg))
}

// onConnected stores the established data source and begins loading databases.
// It immediately populates the sidebar with the default database so the user
// sees something as soon as the connection is established, without waiting for
// the full Databases() round-trip to complete.
func (m Model) onConnected(msg ConnectedMsg) (tea.Model, tea.Cmd) {
	m.source = msg.source
	if m.debugOutput != nil {
		m.source = datasource.WithTiming(m.source, m.debugOutput)
	}
	m.sidebar.SetDatabaseLabel(m.source.DatabaseLabel())

	if conn, ok := m.connectionPicker.ConnectionByName(msg.name); ok {
		m.savedQueries = toModelSavedQueries(conn.SavedQueries)
	}

	m.readOnly = msg.readOnly || m.forceReadOnly
	m.statusbar.SetConnectionName(msg.name)
	m.statusbar.SetReadOnly(m.readOnly)

	defaultDB, err := m.source.DefaultDatabase(context.Background())
	if err != nil || defaultDB == "" {
		spinnerCmd := m.statusbar.StartSpinner("Loading databases", statusbar.Info)
		return m, tea.Batch(spinnerCmd, LoadDatabasesCmd(m.source))
	}
	m.sidebar.SetDatabases([]string{defaultDB})
	m.sidebar.OpenSelectedDatabase()
	spinnerCmd := m.statusbar.StartSpinner("Loading tables", statusbar.Info)
	return m, tea.Batch(
		spinnerCmd,
		LoadDatabasesCmd(m.source),
		LoadTablesCmd(m.source, defaultDB),
	)
}

// onConnectionFailed shows a sticky error in the status bar.
func (m Model) onConnectionFailed(msg ConnectionFailedMsg) (tea.Model, tea.Cmd) {
	m.statusbar.StopSpinner()
	m.statusbar.SetStatus(" Connection failed: "+msg.err.Error(), statusbar.Error)
	return m, nil
}

// ConnectCmd establishes a database connection asynchronously using the given
// config. On success it sends ConnectedMsg; on failure ConnectionFailedMsg.
func ConnectCmd(cfg database.Config) tea.Cmd {
	return func() tea.Msg {
		conn, err := provider.FromConfig(cfg)
		if err != nil {
			return ConnectionFailedMsg{err: err}
		}
		return ConnectedMsg{
			source:   datasource.FromConnection(conn),
			name:     cfg.Name,
			readOnly: cfg.ReadOnly,
		}
	}
}

// toModelSavedQueries converts a list of config.SavedQuery to a list of SavedQuery.
func toModelSavedQueries(savedQueries []config.SavedQuery) []SavedQuery {
	modelSavedQueries := make([]SavedQuery, len(savedQueries))
	for i, savedQuery := range savedQueries {
		modelSavedQueries[i] = SavedQuery{
			Name:  savedQuery.Name,
			Query: savedQuery.Query,
		}
	}
	return modelSavedQueries
}
