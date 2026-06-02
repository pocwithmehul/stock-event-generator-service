package config

import (
	"fmt"
	"os"

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

	return &cfg, nil
}
