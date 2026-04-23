package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the schema-sync configuration
type Config struct {
	Environments map[string]EnvConfig `yaml:"environments"`
	Settings    Settings              `yaml:"settings"`
}

// EnvConfig holds connection info for an environment
type EnvConfig struct {
	DBType string `yaml:"db_type"`
	DSN    string `yaml:"dsn"`
}

// Settings holds general settings
type Settings struct {
	OutputDir    string `yaml:"output_dir"`
	SnapshotDir  string `yaml:"snapshot_dir"`
}

// Load reads config from .schema-sync/config.yaml
func Load(dir string) (*Config, error) {
	configPath := filepath.Join(dir, ".schema-sync", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}

// Save writes config to .schema-sync/config.yaml
func Save(dir string, cfg *Config) error {
	configPath := filepath.Join(dir, ".schema-sync", "config.yaml")

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Environments: make(map[string]EnvConfig),
		Settings: Settings{
			OutputDir:   "migrations",
			SnapshotDir: ".schema-sync/snapshots",
		},
	}
}