package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	"github.com/jxdones/stoat/internal/config"
	"github.com/jxdones/stoat/internal/database"
	"github.com/jxdones/stoat/internal/ui/model"
)

// version is set at build time via -ldflags "-X main.version=vX.Y.Z".
var version = "dev"

func main() {
	dbPath := flag.String("db", "", "path to SQLite database file (e.g. ./mydb.sqlite)")
	dbDSN := flag.String("dsn", "", "PostgreSQL connection string (e.g. postgres://user:password@host:port/database)")
	debug := flag.Bool("debug", false, "write per-call timings to ~/.stoat/debug.log")
	readOnly := flag.Bool("read-only", false, "open connection in read-only mode")

	ver := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *ver {
		fmt.Println(version)
		return
	}

	m := model.New()
	m.SetReadOnly(*readOnly)
	if *dbPath != "" {
		m.SetPendingConfig(database.Config{
			Name:     filepath.Base(*dbPath),
			DBMS:     database.DBMSSQLite,
			Values:   map[string]string{"path": *dbPath},
			ReadOnly: *readOnly,
		})
	} else if *dbDSN != "" {
		m.SetPendingConfig(database.Config{
			Name:     "postgres",
			DBMS:     database.DBMSPostgres,
			Values:   map[string]string{"dsn": *dbDSN},
			ReadOnly: *readOnly,
		})
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	m.SetConfig(cfg)

	if *dbPath == "" && *dbDSN == "" {
		m.OpenConnectionPicker()
	}

	if *debug {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "debug log: %v\n", err)
			os.Exit(1)
		}
		out, err := os.OpenFile(
			filepath.Join(home, ".stoat", "debug.log"),
			os.O_APPEND|os.O_CREATE|os.O_WRONLY,
			0o600,
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "debug log: %v\n", err)
			os.Exit(1)
		}
		m.SetDebugOutput(out)
	}

	program := tea.NewProgram(m)
	app, err := program.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "run: %v\n", err)
		os.Exit(1)
	}
	if m, ok := app.(model.Model); ok {
		m.Close()
	}
}
