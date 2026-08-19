//go:build integration

package testharness

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestIntegrationDatabaseFixtures(t *testing.T) {
	cases := []struct {
		name   string
		driver string
		dsn    string
	}{
		{name: "postgres", driver: "pgx", dsn: os.Getenv("GHOSTFLEET_POSTGRES_DSN")},
		{name: "mysql", driver: "mysql", dsn: os.Getenv("GHOSTFLEET_MYSQL_DSN")},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.dsn == "" {
				t.Skip("integration DSN is not configured")
			}
			db, err := sql.Open(testCase.driver, testCase.dsn)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			ctx := context.Background()
			if err := db.PingContext(ctx); err != nil {
				t.Fatal(err)
			}
			runner := Runner{}
			if err := runner.Load(ctx, db, SchemaFixture); err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := runner.Cleanup(ctx, db, SchemaFixture); err != nil {
					t.Errorf("cleanup: %v", err)
				}
			}()
		})
	}
}
