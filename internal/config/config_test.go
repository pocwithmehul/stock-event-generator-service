package config

import (
	"os"
	"path/filepath"
	"testing"
)

const testYAML = `
server:
  webhookUrl: http://localhost:8090/webhook
  port: 9090
tickers:
  - AAPL
  - MSFT
intervalMs: 2000
datadog:
  apiKey: test-key
  site: datadoghq.com
  env: test
`

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "config-*.yaml")
	if err != nil {
		t.Fatalf("create temp config: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	_ = f.Close()
	return f.Name()
}

func TestLoad_FromFile(t *testing.T) {
	path := writeTempConfig(t, testYAML)
	t.Setenv("CONFIG_PATH", path)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("expected port=9090, got %d", cfg.Server.Port)
	}
	if cfg.Server.WebhookURL != "http://localhost:8090/webhook" {
		t.Errorf("unexpected webhookURL: %s", cfg.Server.WebhookURL)
	}
	if cfg.IntervalMs != 2000 {
		t.Errorf("expected intervalMs=2000, got %d", cfg.IntervalMs)
	}
	if len(cfg.Tickers) != 2 || cfg.Tickers[0] != "AAPL" || cfg.Tickers[1] != "MSFT" {
		t.Errorf("unexpected tickers: %v", cfg.Tickers)
	}
	// DatadogConfig.APIKey has no yaml tag so it cannot be set from YAML; use env var instead
}

func TestLoad_FileNotFound(t *testing.T) {
	t.Setenv("CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.yaml"))

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	path := writeTempConfig(t, "not: valid: yaml: [[[")
	t.Setenv("CONFIG_PATH", path)

	_, err := Load()
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	path := writeTempConfig(t, testYAML)
	t.Setenv("CONFIG_PATH", path)
	t.Setenv("SERVER_PORT", "7070")
	t.Setenv("WEBHOOK_URL", "http://override/webhook")
	t.Setenv("TICKERS", "GOOG , TSLA")
	t.Setenv("INTERVAL_MS", "500")
	t.Setenv("DATADOG_API_KEY", "override-key")
	t.Setenv("DATADOG_SITE", "datadoghq.eu")
	t.Setenv("DATADOG_ENV", "staging")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Server.Port != 7070 {
		t.Errorf("expected port=7070, got %d", cfg.Server.Port)
	}
	if cfg.Server.WebhookURL != "http://override/webhook" {
		t.Errorf("unexpected webhookURL: %s", cfg.Server.WebhookURL)
	}
	if len(cfg.Tickers) != 2 || cfg.Tickers[0] != "GOOG" || cfg.Tickers[1] != "TSLA" {
		t.Errorf("unexpected tickers: %v", cfg.Tickers)
	}
	if cfg.IntervalMs != 500 {
		t.Errorf("expected intervalMs=500, got %d", cfg.IntervalMs)
	}
	if cfg.Datadog.APIKey != "override-key" {
		t.Errorf("unexpected datadog api key: %s", cfg.Datadog.APIKey)
	}
	if cfg.Datadog.Site != "datadoghq.eu" {
		t.Errorf("unexpected datadog site: %s", cfg.Datadog.Site)
	}
	if cfg.Datadog.Env != "staging" {
		t.Errorf("unexpected datadog env: %s", cfg.Datadog.Env)
	}
}

func TestLoad_InvalidPortEnv(t *testing.T) {
	path := writeTempConfig(t, testYAML)
	t.Setenv("CONFIG_PATH", path)
	t.Setenv("SERVER_PORT", "not-a-number")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	// Invalid port env is ignored; value from file is kept
	if cfg.Server.Port != 9090 {
		t.Errorf("expected port=9090 (from file), got %d", cfg.Server.Port)
	}
}

func TestSplitAndTrimCSV(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"AAPL,MSFT,GOOG", []string{"AAPL", "MSFT", "GOOG"}},
		{" AAPL , MSFT ", []string{"AAPL", "MSFT"}},
		{"AAPL,,MSFT", []string{"AAPL", "MSFT"}},
		{"", []string{}},
		{"  ,  ", []string{}},
		{"SINGLE", []string{"SINGLE"}},
	}

	for _, tt := range tests {
		got := splitAndTrimCSV(tt.input)
		if len(got) != len(tt.expected) {
			t.Errorf("splitAndTrimCSV(%q): got %v, want %v", tt.input, got, tt.expected)
			continue
		}
		for i := range got {
			if got[i] != tt.expected[i] {
				t.Errorf("splitAndTrimCSV(%q)[%d]: got %q, want %q", tt.input, i, got[i], tt.expected[i])
			}
		}
	}
}

func TestApplyEnvOverrides_NoEnv(t *testing.T) {
	cfg := &Config{
		Server:     ServerConfig{Port: 8080, WebhookURL: "http://original"},
		Tickers:    []string{"AAPL"},
		IntervalMs: 1000,
	}
	applyEnvOverrides(cfg)
	// Nothing should change when no env vars are set
	if cfg.Server.Port != 8080 {
		t.Errorf("expected port unchanged at 8080, got %d", cfg.Server.Port)
	}
	if cfg.Server.WebhookURL != "http://original" {
		t.Errorf("expected webhookURL unchanged")
	}
}
