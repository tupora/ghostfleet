package storage

import (
	"strings"
	"testing"
)

func TestControlPlaneMigrationContainsSafetyConstraints(t *testing.T) {
	migration, err := migrationFS.ReadFile("migrations/001_control_plane.sql")
	if err != nil {
		t.Fatal(err)
	}
	schema := string(migration)
	for _, required := range []string{
		"UNIQUE (target_id, version_id)",
		"PRIMARY KEY (task_id, generation)",
		"CREATE TABLE plans",
		"CREATE TABLE target_history",
		"CREATE TABLE audit_events",
		"ghostfleet_reject_immutable_mutation",
		"generation < $3",
	} {
		if !strings.Contains(schema, required) {
			t.Errorf("migration missing %q", required)
		}
	}
}
