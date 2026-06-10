package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	commonlogger "github.com/pocwithmehul/common-go-lib/pkg/logger"
	"gopkg.in/yaml.v3"
)

const defaultConfigPath = "config/dev/config.yaml"

type Config struct {
	Server     ServerConfig               `yaml:"server"`
	Tickers    []string                   `yaml:"tickers"`
	IntervalMs int                        `yaml:"intervalMs"`
	Datadog    commonlogger.DatadogConfig `yaml:"datadog"`
}

type ServerConfig struct {
	WebhookURL string `yaml:"webhookUrl"`
	Port       int    `yaml:"port"`
}

func Load() (*Config, error) {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = defaultConfigPath
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", configPath, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", configPath, err)
	}

	applyEnvOverrides(&cfg)

	return &cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if port := os.Getenv("SERVER_PORT"); port != "" {
		if parsed, err := strconv.Atoi(port); err == nil {
			cfg.Server.Port = parsed
		}
	}

	if webhookURL := os.Getenv("WEBHOOK_URL"); webhookURL != "" {
		cfg.Server.WebhookURL = webhookURL
	}

	if tickers := os.Getenv("TICKERS"); tickers != "" {
		cfg.Tickers = splitAndTrimCSV(tickers)
	}

	if interval := os.Getenv("INTERVAL_MS"); interval != "" {
		if parsed, err := strconv.Atoi(interval); err == nil {
			cfg.IntervalMs = parsed
		}
	}

	if apiKey := os.Getenv("DATADOG_API_KEY"); apiKey != "" {
		cfg.Datadog.APIKey = apiKey
	}

	if site := os.Getenv("DATADOG_SITE"); site != "" {
		cfg.Datadog.Site = site
	}

	if env := os.Getenv("DATADOG_ENV"); env != "" {
		cfg.Datadog.Env = env
	}
}

func splitAndTrimCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}
