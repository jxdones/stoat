package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/jxdones/stoat/internal/config"
	"github.com/jxdones/stoat/internal/database"
	"github.com/jxdones/stoat/internal/ui/model"
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "stoat",
	Short: "Stoat is a database client for people who don't leave the terminal.",
	RunE: func(cmd *cobra.Command, args []string) error {
		version, _ := cmd.Flags().GetBool("version")
		if version {
			PrintVersion()
			return nil
		}

		dbPath, err := cmd.Flags().GetString("db")
		if err != nil {
			return err
		}

		dbDSN, err := cmd.Flags().GetString("dsn")
		if err != nil {
			return err
		}

		debug, err := cmd.Flags().GetBool("debug")
		if err != nil {
			return err
		}

		readOnly, err := cmd.Flags().GetBool("read-only")
		if err != nil {
			return err
		}

		m := model.New()
		m.SetReadOnly(readOnly)
		if dbPath != "" {
			m.SetPendingConfig(database.Config{
				Name:     filepath.Base(dbPath),
				DBMS:     database.DBMSSQLite,
				Values:   map[string]string{"path": dbPath},
				ReadOnly: readOnly,
			})
		} else if dbDSN != "" {
			m.SetPendingConfig(database.Config{
				Name:     "postgres",
				DBMS:     database.DBMSPostgres,
				Values:   map[string]string{"dsn": dbDSN},
				ReadOnly: readOnly,
			})
		}

		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "load config: %v\n", err)
			os.Exit(1)
		}
		m.SetConfig(cfg)

		if dbPath == "" && dbDSN == "" {
			m.OpenConnectionPicker()
		}

		if debug {
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
		return nil
	},
}

func init() {
	RootCmd.Flags().StringP("db", "d", "", "path to SQLite database file (e.g. ./mydb.sqlite)")
	RootCmd.Flags().StringP("dsn", "s", "", "PostgreSQL connection string (e.g. postgres://user:password@host:port/database)")
	RootCmd.Flags().BoolP("debug", "D", false, "write per-call timings to ~/.stoat/debug.log")
	RootCmd.Flags().BoolP("read-only", "r", false, "open connection in read-only mode")
	RootCmd.Flags().BoolP("version", "v", false, "Print the version number of stoat")
}
