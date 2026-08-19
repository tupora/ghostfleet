package testharness

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var secretPattern = regexp.MustCompile(`(?i)(password|passwd|token|authorization|dsn)(["']?\s*[:=]\s*["']?)([^"'\s,;}]+)`)

func Redact(data []byte) []byte {
	return secretPattern.ReplaceAll(data, []byte(`${1}${2}<redacted>`))
}

func CollectFailureArtifacts(root string, logs []byte, snapshots map[string][]byte) error {
	if strings.TrimSpace(root) == "" {
		return fmt.Errorf("artifact directory is required")
	}
	if err := os.MkdirAll(root, 0o750); err != nil {
		return fmt.Errorf("create artifact directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(root, "logs.txt"), Redact(logs), 0o640); err != nil {
		return fmt.Errorf("write logs: %w", err)
	}
	keys := make([]string, 0, len(snapshots))
	for name := range snapshots {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	for _, name := range keys {
		cleanName := filepath.Base(name)
		if cleanName == "." || cleanName == string(filepath.Separator) || cleanName == "" {
			return fmt.Errorf("invalid snapshot name %q", name)
		}
		if err := os.WriteFile(filepath.Join(root, cleanName), Redact(snapshots[name]), 0o640); err != nil {
			return fmt.Errorf("write snapshot %q: %w", name, err)
		}
	}
	return nil
}
