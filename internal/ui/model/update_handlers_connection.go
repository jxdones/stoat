package model

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/jxdones/stoat/internal/config"
	"github.com/jxdones/stoat/internal/database"
	"github.com/jxdones/stoat/internal/ui/components/connectionpicker"
	"github.com/jxdones/stoat/internal/ui/components/statusbar"
	"github.com/jxdones/stoat/internal/ui/datasource"
)

// handleConnecting sets the status bar to "Connecting…" and fires the async
// ConnectCmd. The extra message hop gives the UI one render cycle to paint
// the status before the blocking provider call starts.
func (m Model) handleConnecting(msg ConnectingMsg) (tea.Model, tea.Cmd) {
	spinnerCmd := m.statusbar.StartSpinner("Connecting to "+msg.cfg.Name, statusbar.Info)
	return m, tea.Batch(spinnerCmd, ConnectCmd(msg.cfg))
}

// handleConnected stores the established data source and begins loading databases.
// It immediately populates the sidebar with the default database so the user
// sees something as soon as the connection is established, without waiting for
// the full Databases() round-trip to complete. The real schema list loads in
// parallel and replaces this placeholder when it arrives.
func (m Model) handleConnected(msg ConnectedMsg) (tea.Model, tea.Cmd) {
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

// handleConnectionFailed shows a sticky error in the status bar.
func (m Model) handleConnectionFailed(msg ConnectionFailedMsg) (tea.Model, tea.Cmd) {
	m.statusbar.StopSpinner()
	m.statusbar.SetStatus(" Connection failed: "+msg.err.Error(), statusbar.Error)
	return m, nil
}

func (m Model) handleKeyPressInConnectionPicker(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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
			dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s",
				selected.User, selected.Password, selected.Host, port, selected.Database)
			cfg.DBMS = database.DBMSPostgres
			cfg.Values = map[string]string{"dsn": dsn}
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
