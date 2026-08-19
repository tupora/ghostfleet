package bootstrap

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunVersion(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if code := Run("ghostcli", []string{"-version"}, &stdout, &stderr); code != 0 {
		t.Fatalf("Run() code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if !strings.HasPrefix(stdout.String(), "ghostcli ") {
		t.Fatalf("Run() stdout = %q, want ghostcli version", stdout.String())
	}
}

func TestRunWithoutCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if code := Run("ghostcli", nil, &stdout, &stderr); code != 2 {
		t.Fatalf("Run() code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "no command selected") {
		t.Fatalf("Run() stderr = %q, want usage error", stderr.String())
	}
}
