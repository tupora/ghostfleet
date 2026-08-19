package testharness

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type recordingExecutor struct {
	statements []string
	err        error
}

func (e *recordingExecutor) ExecContext(_ context.Context, statement string, _ ...any) (sql.Result, error) {
	e.statements = append(e.statements, statement)
	if e.err != nil {
		return nil, e.err
	}
	return driverResult{}, nil
}

type driverResult struct{}

func (driverResult) LastInsertId() (int64, error) { return 0, nil }
func (driverResult) RowsAffected() (int64, error) { return 0, nil }

func TestRunnerLoadsAndCleansFixtures(t *testing.T) {
	db := &recordingExecutor{}
	fixture := Fixture{Name: "schema", Setup: []string{"CREATE TABLE probe"}, Cleanup: []string{"DROP TABLE probe"}}
	runner := Runner{}
	if err := runner.Load(context.Background(), db, fixture); err != nil {
		t.Fatal(err)
	}
	if err := runner.Cleanup(context.Background(), db, fixture); err != nil {
		t.Fatal(err)
	}
	if strings.Join(db.statements, ";") != "CREATE TABLE probe;DROP TABLE probe" {
		t.Fatalf("statements = %#v", db.statements)
	}
}

func TestFaultInjectionCoversDisconnectAndProcessCrash(t *testing.T) {
	injector := NewInjector()
	injector.FailNext(DatabaseDisconnect, nil)
	db := FaultyExecutor{Base: &recordingExecutor{}, Injector: injector}
	if _, err := db.ExecContext(context.Background(), "SELECT 1"); !strings.Contains(err.Error(), "database disconnect") {
		t.Fatalf("disconnect error = %v", err)
	}
	injector.FailNext(ProcessCrash, nil)
	if err := injector.Check(ProcessCrash); !errors.Is(err, ErrInjectedFault) {
		t.Fatalf("process crash error = %v", err)
	}
	if err := injector.Check(ProcessCrash); err != nil {
		t.Fatalf("fault was not one-shot: %v", err)
	}
}

func TestFailureArtifactsAreRedactedAndDeterministic(t *testing.T) {
	root := t.TempDir()
	if err := CollectFailureArtifacts(root, []byte("dsn=postgres://user:secret password=topsecret"), map[string][]byte{
		"state.json": []byte(`{"token":"secret"}`),
	}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"logs.txt", "state.json"} {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "secret") {
			t.Fatalf("%s contains an unredacted secret: %q", name, data)
		}
	}
}
