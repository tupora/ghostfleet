package storage

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/tupora/ghostfleet/internal/apperr"
)

func TestMemoryStoreEnforcesVersionAndDeploymentUniqueness(t *testing.T) {
	store := NewMemoryStore()
	version := Version{ID: "v1", TargetID: "target-a", Checksum: "sum-1", Fingerprint: "fp-1"}
	if err := store.CreateVersion(context.Background(), version); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateVersion(context.Background(), version); err != nil {
		t.Fatalf("idempotent version creation: %v", err)
	}
	changed := version
	changed.Fingerprint = "fp-2"
	if err := store.CreateVersion(context.Background(), changed); !apperr.IsCode(err, apperr.CodeConflict) {
		t.Fatalf("changed version error = %v, want CONFLICT", err)
	}
	deployment := Deployment{ID: "d1", TargetID: "target-a", VersionID: "v1", DesiredFingerprint: "fp-1"}
	if err := store.CreateDeployment(context.Background(), deployment); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateDeployment(context.Background(), deployment); !apperr.IsCode(err, apperr.CodeConflict) {
		t.Fatalf("duplicate deployment error = %v, want CONFLICT", err)
	}
}

func TestMemoryStoreClaimIsAtomicAndFencesStaleWorker(t *testing.T) {
	store := NewMemoryStore()
	seedStore(t, store)
	if err := store.CreateTask(context.Background(), CreateTaskRequest{ID: "task-1", DeploymentID: "d1"}); err != nil {
		t.Fatal(err)
	}

	const workers = 16
	results := make(chan error, workers)
	var wait sync.WaitGroup
	for i := 0; i < workers; i++ {
		wait.Add(1)
		go func(i int) {
			defer wait.Done()
			_, err := store.ClaimTask(context.Background(), ClaimTaskRequest{
				TaskID: "task-1", WorkerID: "worker-" + string(rune('a'+i)), Generation: 1,
				LeaseDuration: time.Minute, IdempotencyKey: "request-1",
			})
			results <- err
		}(i)
	}
	wait.Wait()
	close(results)

	var successes int
	for err := range results {
		if err == nil {
			successes++
		} else if !apperr.IsCode(err, apperr.CodeConflict) {
			t.Fatalf("claim error = %v, want CONFLICT for losers", err)
		}
	}
	if successes != 1 {
		t.Fatalf("successful claims = %d, want 1", successes)
	}
	task, ok := store.Task("task-1")
	if !ok {
		t.Fatal("claimed task is missing")
	}
	if err := store.CompleteTask(context.Background(), "task-1", "stale-worker", task.Generation); !apperr.IsCode(err, apperr.CodeStaleFence) {
		t.Fatalf("stale completion error = %v, want STALE_FENCE", err)
	}
	if err := store.CompleteTask(context.Background(), "task-1", task.WorkerID, task.Generation); err != nil {
		t.Fatalf("current completion: %v", err)
	}
}

func TestMemoryStoreReconcilesByPausingOnUnknownOrMismatchedHistory(t *testing.T) {
	store := NewMemoryStore()
	seedStore(t, store)
	if got, err := store.ReconcileTarget(context.Background(), TargetObservation{TargetID: "target-a"}); err != nil || got.Action != ReconciliationPause {
		t.Fatalf("unknown target = %#v, %v; want PAUSE", got, err)
	}
	if err := store.CreateTask(context.Background(), CreateTaskRequest{ID: "task-1", DeploymentID: "d1"}); err != nil {
		t.Fatal(err)
	}
	claimed, err := store.ClaimTask(context.Background(), ClaimTaskRequest{
		TaskID: "task-1", WorkerID: "worker-a", Generation: 1, LeaseDuration: time.Minute, IdempotencyKey: "request-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CompleteTask(context.Background(), claimed.ID, claimed.WorkerID, claimed.Generation); err != nil {
		t.Fatal(err)
	}
	if err := store.CompleteDeployment(context.Background(), "d1", claimed.WorkerID, claimed.Generation); err != nil {
		t.Fatal(err)
	}
	if got, err := store.ReconcileTarget(context.Background(), TargetObservation{TargetID: "target-a", Known: true, VersionID: "v1", Fingerprint: "wrong"}); err != nil || got.Action != ReconciliationPause {
		t.Fatalf("mismatched target = %#v, %v; want PAUSE", got, err)
	}
	if got, err := store.ReconcileTarget(context.Background(), TargetObservation{TargetID: "target-a", Known: true, VersionID: "v1", Fingerprint: "fp-1"}); err != nil || got.Action != ReconciliationContinue {
		t.Fatalf("matching target = %#v, %v; want CONTINUE", got, err)
	}
}

func seedStore(t *testing.T, store *MemoryStore) {
	t.Helper()
	if err := store.CreateVersion(context.Background(), Version{ID: "v1", TargetID: "target-a", Checksum: "sum-1", Fingerprint: "fp-1"}); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateDeployment(context.Background(), Deployment{ID: "d1", TargetID: "target-a", VersionID: "v1", DesiredFingerprint: "fp-1"}); err != nil {
		t.Fatal(err)
	}
}
