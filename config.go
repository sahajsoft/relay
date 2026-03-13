package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server    ServerConfig              `yaml:"server"`
	Providers map[string]ProviderConfig `yaml:"providers"`
}

type ServerConfig struct {
	Port   int    `yaml:"port"`
	LogDir string `yaml:"log_dir"`
}

type ProviderConfig struct {
	BaseURL    string `yaml:"base_url"`
	APIKey     string `yaml:"api_key"`
	AuthHeader string `yaml:"auth_header"`
	AuthScheme string `yaml:"auth_scheme"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	expanded := os.ExpandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}

	if cfg.Server.LogDir == "" {
		cfg.Server.LogDir = "logs"
	}

	if len(cfg.Providers) == 0 {
		return nil, fmt.Errorf("no providers configured")
	}

	for name, p := range cfg.Providers {
		if p.BaseURL == "" {
			return nil, fmt.Errorf("provider %q: base_url is required", name)
		}
	}

	return &cfg, nil
}
