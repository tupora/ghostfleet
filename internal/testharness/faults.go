package testharness

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
)

var ErrInjectedFault = errors.New("injected test fault")

type FaultPoint string

const (
	ProcessCrash       FaultPoint = "process-crash"
	DatabaseDisconnect FaultPoint = "database-disconnect"
)

type Injector struct {
	mu      sync.Mutex
	pending map[FaultPoint]error
}

func NewInjector() *Injector {
	return &Injector{pending: make(map[FaultPoint]error)}
}

func (i *Injector) FailNext(point FaultPoint, cause error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if cause == nil {
		cause = ErrInjectedFault
	}
	i.pending[point] = cause
}

func (i *Injector) Check(point FaultPoint) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	err, ok := i.pending[point]
	if ok {
		delete(i.pending, point)
	}
	return err
}

// FaultyExecutor injects a disconnect at the next SQL execution. Process
// crashes use the same injector through an explicit Check(ProcessCrash) hook,
// allowing a test to model restart and re-entry without terminating the test
// process itself.
type FaultyExecutor struct {
	Base     SQLExecutor
	Injector *Injector
}

func (e FaultyExecutor) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if e.Base == nil || e.Injector == nil {
		return nil, fmt.Errorf("faulty executor requires a base and injector")
	}
	if err := e.Injector.Check(DatabaseDisconnect); err != nil {
		return nil, fmt.Errorf("database disconnect: %w", err)
	}
	return e.Base.ExecContext(ctx, query, args...)
}
