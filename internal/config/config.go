package config

import (
	"fmt"
	"strings"
)

const (
	defaultEnvironment = "development"
	defaultLogLevel    = "info"
)

type LookupEnv func(string) (string, bool)

type Config struct {
	Environment string
	LogLevel    string
	PostgresDSN string
	MySQLDSN    string
}

func Load(lookup LookupEnv) (Config, error) {
	cfg := Config{
		Environment: valueOrDefault(lookup, "GHOSTFLEET_ENV", defaultEnvironment),
		LogLevel:    valueOrDefault(lookup, "GHOSTFLEET_LOG_LEVEL", defaultLogLevel),
		PostgresDSN: valueOrDefault(lookup, "GHOSTFLEET_POSTGRES_DSN", ""),
		MySQLDSN:    valueOrDefault(lookup, "GHOSTFLEET_MYSQL_DSN", ""),
	}

	switch cfg.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return Config{}, fmt.Errorf("GHOSTFLEET_LOG_LEVEL %q is invalid", cfg.LogLevel)
	}
	return cfg, nil
}

func valueOrDefault(lookup LookupEnv, key, fallback string) string {
	value, ok := lookup(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}
