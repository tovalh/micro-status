package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeConfig drops a YAML file in a temp dir and returns its path.
func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("writing config: %v", err)
	}
	return path
}

func TestLoad(t *testing.T) {
	path := writeConfig(t, `
services:
  - name: Orders API
    url: http://localhost:8081/health
    interval: 5s
    dependencies_url: http://localhost:8081/health/dependencies
  - name: Public Echo
    url: https://example.com
    interval: 10s
webhook:
  url: https://hooks.example.com/abc
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Services) != 2 {
		t.Fatalf("got %d services, want 2", len(cfg.Services))
	}
	if cfg.Services[0].Interval != 5*time.Second {
		t.Errorf("interval = %v, want 5s", cfg.Services[0].Interval)
	}
	if cfg.Services[0].DependenciesURL == "" {
		t.Error("expected dependencies_url to be parsed")
	}
	if cfg.Webhook.URL != "https://hooks.example.com/abc" {
		t.Errorf("webhook = %q, want the configured URL", cfg.Webhook.URL)
	}
}

func TestLoadErrors(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		if _, err := Load(filepath.Join(t.TempDir(), "nope.yaml")); err == nil {
			t.Fatal("expected an error for a missing file")
		}
	})

	t.Run("no services", func(t *testing.T) {
		path := writeConfig(t, "webhook:\n  url: \"\"\n")
		if _, err := Load(path); err == nil {
			t.Fatal("expected an error when no services are defined")
		}
	})

	t.Run("missing url", func(t *testing.T) {
		path := writeConfig(t, "services:\n  - name: Orders API\n    interval: 5s\n")
		if _, err := Load(path); err == nil {
			t.Fatal("expected an error when a service has no url")
		}
	})

	t.Run("bad interval", func(t *testing.T) {
		path := writeConfig(t, "services:\n  - name: Orders API\n    url: http://x\n    interval: soon\n")
		if _, err := Load(path); err == nil {
			t.Fatal("expected an error for an unparseable interval")
		}
	})
}

func TestLoadTimingDefaults(t *testing.T) {
	// Clear the vars so we exercise the fallbacks.
	for _, k := range []string{"REQUEST_TIMEOUT", "DOWN_CONFIRM_ATTEMPTS", "DOWN_CONFIRM_DELAY", "UP_CONFIRM_ATTEMPTS", "UP_CONFIRM_DELAY"} {
		t.Setenv(k, "")
	}

	tm := LoadTiming()
	if tm.RequestTimeout != 5*time.Second {
		t.Errorf("RequestTimeout = %v, want 5s", tm.RequestTimeout)
	}
	if tm.DownAttempts != 2 {
		t.Errorf("DownAttempts = %d, want 2", tm.DownAttempts)
	}
	if tm.UpDelay != 10*time.Second {
		t.Errorf("UpDelay = %v, want 10s", tm.UpDelay)
	}
}

func TestLoadTimingOverrides(t *testing.T) {
	t.Setenv("REQUEST_TIMEOUT", "2s")
	t.Setenv("DOWN_CONFIRM_ATTEMPTS", "5")

	tm := LoadTiming()
	if tm.RequestTimeout != 2*time.Second {
		t.Errorf("RequestTimeout = %v, want 2s", tm.RequestTimeout)
	}
	if tm.DownAttempts != 5 {
		t.Errorf("DownAttempts = %d, want 5", tm.DownAttempts)
	}
}

func TestEnvFallbacks(t *testing.T) {
	// A garbage value falls back; a valid one is used.
	t.Setenv("X", "garbage")
	if got := durationEnv("X", time.Second); got != time.Second {
		t.Errorf("durationEnv fallback = %v, want 1s", got)
	}
	if got := intEnv("X", 3); got != 3 {
		t.Errorf("intEnv fallback = %d, want 3", got)
	}

	t.Setenv("X", "7")
	if got := intEnv("X", 3); got != 7 {
		t.Errorf("intEnv = %d, want 7", got)
	}

	// Negative durations are rejected too.
	t.Setenv("X", "-1s")
	if got := durationEnv("X", time.Second); got != time.Second {
		t.Errorf("durationEnv negative fallback = %v, want 1s", got)
	}
}
