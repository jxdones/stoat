package cmd

import (
	"testing"

	"github.com/jxdones/stoat/internal/database"
)

func TestCheckDBMS(t *testing.T) {
	tests := []struct {
		name string
		dsn  string
		want database.DBMS
	}{
		{
			name: "postgres_scheme",
			dsn:  "postgres://user:password@host:5432/dbname",
			want: database.DBMSPostgres,
		},
		{
			name: "postgresql_scheme",
			dsn:  "postgresql://user:password@host:5432/dbname",
			want: database.DBMSPostgres,
		},
		{
			name: "mysql_scheme",
			dsn:  "mysql://user:password@host:3306/dbname",
			want: database.DBMSMySQL,
		},
		{
			name: "mysql_go_driver_tcp",
			dsn:  "user:password@tcp(host:3306)/dbname",
			want: database.DBMSMySQL,
		},
		{
			name: "mysql_go_driver_unix",
			dsn:  "user:password@unix(/var/run/mysqld/mysqld.sock)/dbname",
			want: database.DBMSMySQL,
		},
		{
			name: "unknown",
			dsn:  "something://user:password@host:1234/dbname",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkDBMS(tt.dsn)
			if got != tt.want {
				t.Errorf("checkDBMS(%q) = %q, want %q", tt.dsn, got, tt.want)
			}
		})
	}
}
