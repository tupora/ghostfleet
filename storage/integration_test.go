//go:build integration

package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestPostgresSchemaAndClaimTransaction(t *testing.T) {
	dsn := os.Getenv("GHOSTFLEET_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("GHOSTFLEET_POSTGRES_DSN is not configured")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	if err := ApplyMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}

	unique := fmt.Sprintf("integration-%d", time.Now().UnixNano())
	versionID := "v-" + unique
	targetID := "target-" + unique
	deploymentID := "deployment-" + unique
	taskID := "task-" + unique
	defer func() {
		_, _ = db.ExecContext(ctx, `DELETE FROM leases WHERE task_id = $1`, taskID)
		_, _ = db.ExecContext(ctx, `DELETE FROM tasks WHERE task_id = $1`, taskID)
		_, _ = db.ExecContext(ctx, `DELETE FROM deployments WHERE deployment_id = $1`, deploymentID)
		_, _ = db.ExecContext(ctx, `DELETE FROM target_history WHERE target_id = $1`, targetID)
		_, _ = db.ExecContext(ctx, `DELETE FROM schema_versions WHERE version_id = $1`, versionID)
	}()

	if _, err := db.ExecContext(ctx, `INSERT INTO schema_versions(version_id, target_id, checksum, fingerprint, created_at)
		VALUES ($1, $2, 'checksum', 'fingerprint', CURRENT_TIMESTAMP)`, versionID, targetID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO deployments(deployment_id, target_id, version_id, source_fingerprint, desired_fingerprint, state, created_at)
		VALUES ($1, $2, $3, 'source', 'fingerprint', 'PENDING', CURRENT_TIMESTAMP)`, deploymentID, targetID, versionID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO deployments(deployment_id, target_id, version_id, source_fingerprint, desired_fingerprint, state, created_at)
		VALUES ($1, $2, $3, 'source', 'fingerprint', 'PENDING', CURRENT_TIMESTAMP)`, deploymentID+"-duplicate", targetID, versionID); err == nil {
		t.Fatal("duplicate target/version deployment unexpectedly succeeded")
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO tasks(task_id, deployment_id, state) VALUES ($1, $2, 'PENDING')`, taskID, deploymentID); err != nil {
		t.Fatal(err)
	}

	const claimers = 8
	results := make(chan error, claimers)
	var wait sync.WaitGroup
	for i := 0; i < claimers; i++ {
		wait.Add(1)
		go func(i int) {
			defer wait.Done()
			results <- claimPostgresTask(ctx, db, taskID, fmt.Sprintf("worker-%d", i), 1)
		}(i)
	}
	wait.Wait()
	close(results)
	var successfulClaims int
	for err := range results {
		if err == nil {
			successfulClaims++
		} else if err != sql.ErrNoRows {
			t.Fatalf("claim error = %v, want no-row for a fenced loser", err)
		}
	}
	if successfulClaims != 1 {
		t.Fatalf("successful claims = %d, want 1", successfulClaims)
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO target_history(target_id, version_id, fingerprint, observed_at)
		VALUES ($1, $2, 'fingerprint', CURRENT_TIMESTAMP)`, targetID, versionID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE target_history SET fingerprint = 'changed' WHERE target_id = $1 AND version_id = $2`, targetID, versionID); err == nil {
		t.Fatal("immutable target history update unexpectedly succeeded")
	}
}

func claimPostgresTask(ctx context.Context, db *sql.DB, taskID, workerID string, generation uint64) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var claimedID string
	err = tx.QueryRowContext(ctx, `UPDATE tasks
		SET state = 'RUNNING', worker_id = $2, generation = $3,
			attempt = attempt + 1, lease_expires_at = $4, idempotency_key = $5
		WHERE task_id = $1
		  AND (lease_expires_at IS NULL OR lease_expires_at <= CURRENT_TIMESTAMP)
		  AND generation < $3
		RETURNING task_id`, taskID, workerID, generation, time.Now().Add(time.Minute), "integration-"+workerID).Scan(&claimedID)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO leases(task_id, generation, worker_id, expires_at, claimed_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)`, claimedID, generation, workerID, time.Now().Add(time.Minute)); err != nil {
		return err
	}
	return tx.Commit()
}
