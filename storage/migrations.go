package storage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"fmt"
)

//go:embed migrations/001_control_plane.sql
var migrationFS embed.FS

// ApplyMigrations applies numbered migrations atomically and rejects a changed
// checksum for an already-applied migration.
func ApplyMigrations(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("database is required")
	}
	migration, err := migrationFS.ReadFile("migrations/001_control_plane.sql")
	if err != nil {
		return fmt.Errorf("read control-plane migration: %w", err)
	}
	checksum := sha256.Sum256(migration)
	checksumText := hex.EncodeToString(checksum[:])
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration transaction: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		checksum TEXT NOT NULL,
		applied_at TIMESTAMPTZ NOT NULL
	)`); err != nil {
		return fmt.Errorf("create migration ledger: %w", err)
	}
	var storedChecksum string
	err = tx.QueryRowContext(ctx, `SELECT checksum FROM schema_migrations WHERE version = $1`, "001_control_plane").Scan(&storedChecksum)
	switch err {
	case nil:
		if storedChecksum != checksumText {
			return fmt.Errorf("migration 001_control_plane checksum changed")
		}
		return tx.Commit()
	case sql.ErrNoRows:
		// Apply the schema below and record the checksum in this transaction.
	default:
		return fmt.Errorf("read migration ledger: %w", err)
	}
	if _, err := tx.ExecContext(ctx, string(migration)); err != nil {
		return fmt.Errorf("apply control-plane migration: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations(version, checksum, applied_at)
		VALUES ($1, $2, CURRENT_TIMESTAMP)`, "001_control_plane", checksumText); err != nil {
		return fmt.Errorf("record control-plane migration: %w", err)
	}
	return tx.Commit()
}
