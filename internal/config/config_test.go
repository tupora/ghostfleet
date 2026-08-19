package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Environment != "development" || cfg.LogLevel != "info" {
		t.Fatalf("Load() = %#v, want development/info defaults", cfg)
	}
}

func TestLoadRejectsUnknownLogLevel(t *testing.T) {
	_, err := Load(func(key string) (string, bool) {
		if key == "GHOSTFLEET_LOG_LEVEL" {
			return "verbose", true
		}
		return "", false
	})
	if err == nil {
		t.Fatal("Load() error = nil, want invalid log-level error")
	}
}
