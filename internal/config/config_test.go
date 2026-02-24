package config

import "testing"

func TestParseDefaults(t *testing.T) {
	cfg, err := Parse(nil, func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.RunAddress != defaultRunAddress {
		t.Fatalf("RunAddress = %q, want %q", cfg.RunAddress, defaultRunAddress)
	}
	if cfg.DatabaseURI != defaultDatabaseURI {
		t.Fatalf("DatabaseURI = %q, want %q", cfg.DatabaseURI, defaultDatabaseURI)
	}
	if cfg.AccrualAddress != defaultAccrualAddress {
		t.Fatalf("AccrualAddress = %q, want %q", cfg.AccrualAddress, defaultAccrualAddress)
	}
	if cfg.LogLevel != defaultLogLevel {
		t.Fatalf("LogLevel = %q, want %q", cfg.LogLevel, defaultLogLevel)
	}
}

func TestParseEnvOverride(t *testing.T) {
	lookup := func(key string) (string, bool) {
		values := map[string]string{
			"RUN_ADDRESS":            "0.0.0.0:9000",
			"DATABASE_URI":           "postgres://example",
			"ACCRUAL_SYSTEM_ADDRESS": "http://accrual.local",
			"LOG_LEVEL":              "debug",
		}
		v, ok := values[key]
		return v, ok
	}

	cfg, err := Parse(nil, lookup)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.RunAddress != "0.0.0.0:9000" {
		t.Fatalf("RunAddress = %q", cfg.RunAddress)
	}
	if cfg.DatabaseURI != "postgres://example" {
		t.Fatalf("DatabaseURI = %q", cfg.DatabaseURI)
	}
	if cfg.AccrualAddress != "http://accrual.local" {
		t.Fatalf("AccrualAddress = %q", cfg.AccrualAddress)
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("LogLevel = %q", cfg.LogLevel)
	}
}

func TestParseFlagsOverrideEnv(t *testing.T) {
	lookup := func(key string) (string, bool) {
		values := map[string]string{
			"RUN_ADDRESS":            "0.0.0.0:9000",
			"DATABASE_URI":           "postgres://example",
			"ACCRUAL_SYSTEM_ADDRESS": "http://accrual.local",
			"LOG_LEVEL":              "debug",
		}
		v, ok := values[key]
		return v, ok
	}

	args := []string{"-a", "127.0.0.1:7777", "-d", "postgres://override", "-r", "http://accrual.override", "-l", "warn"}
	cfg, err := Parse(args, lookup)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.RunAddress != "127.0.0.1:7777" {
		t.Fatalf("RunAddress = %q", cfg.RunAddress)
	}
	if cfg.DatabaseURI != "postgres://override" {
		t.Fatalf("DatabaseURI = %q", cfg.DatabaseURI)
	}
	if cfg.AccrualAddress != "http://accrual.override" {
		t.Fatalf("AccrualAddress = %q", cfg.AccrualAddress)
	}
	if cfg.LogLevel != "warn" {
		t.Fatalf("LogLevel = %q", cfg.LogLevel)
	}
}

func TestParseInvalidFlag(t *testing.T) {
	_, err := Parse([]string{"-unknown"}, func(string) (string, bool) { return "", false })
	if err == nil {
		t.Fatalf("expected error")
	}
}
