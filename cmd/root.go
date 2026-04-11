package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

		dbDSNEnv, err := cmd.Flags().GetString("dsn-env")
		if err != nil {
			return err
		}

		if dbDSN != "" && dbDSNEnv != "" {
			fmt.Fprintf(os.Stderr, "Only one dsn option should be set.\n")
			os.Exit(1)
		}

		if dbDSNEnv != "" {
			dbDSN = os.Getenv(dbDSNEnv)
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
			dbms := checkDBMS(dbDSN)
			name := ""

			switch dbms {
			case database.DBMSPostgres:
				name = "postgres"
			case database.DBMSMySQL:
				name = "mysql"
			case "":
				fmt.Fprintf(os.Stderr, "Invalid DBMS.\n")
				os.Exit(1)
			}

			m.SetPendingConfig(database.Config{
				Name:     name,
				DBMS:     dbms,
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
	RootCmd.Flags().StringP("dsn", "s", "", "PostgreSQL or MySQL/MariaDB connection string (e.g. postgres://user:password@host:port/database)")
	RootCmd.Flags().StringP("dsn-env", "", "", "name of the environment variable containing the DSN (e.g. MY_SECRET_DSN).\nIt keeps credentials out of shell history\n")
	RootCmd.Flags().BoolP("debug", "D", false, "write per-call timings to ~/.stoat/debug.log")
	RootCmd.Flags().BoolP("read-only", "r", false, "open connection in read-only mode")
	RootCmd.Flags().BoolP("version", "v", false, "Print the version number of stoat")
}

// checkDBMS returns the DBMS from the DSN string
func checkDBMS(dsn string) database.DBMS {
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		return database.DBMSPostgres
	}

	if strings.HasPrefix(dsn, "mysql://") || strings.Contains(dsn, "@tcp(") || strings.Contains(dsn, "@unix(") {
		return database.DBMSMySQL
	}
	return ""
}
