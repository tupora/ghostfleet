package testharness

import (
	"context"
	"database/sql"
	"fmt"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

type Fixture struct {
	Name    string
	Setup   []string
	Cleanup []string
}

type Runner struct{}

func (Runner) Load(ctx context.Context, db SQLExecutor, fixture Fixture) error {
	return apply(ctx, db, fixture.Name, fixture.Setup)
}

func (Runner) Cleanup(ctx context.Context, db SQLExecutor, fixture Fixture) error {
	return apply(ctx, db, fixture.Name, fixture.Cleanup)
}

func apply(ctx context.Context, db SQLExecutor, name string, statements []string) error {
	if db == nil {
		return fmt.Errorf("%s fixture: database is required", name)
	}
	for index, statement := range statements {
		if statement == "" {
			continue
		}
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("%s fixture statement %d: %w", name, index+1, err)
		}
	}
	return nil
}
