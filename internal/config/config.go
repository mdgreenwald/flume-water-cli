// Package config manages the on-disk configuration file for the Flume CLI.
// The file lives at $XDG_CONFIG_HOME/flume/config.yaml (defaulting to
// $HOME/.config/flume/config.yaml) with permissions 0600.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds user-configurable CLI settings.
type Config struct {
	// OutputFormat controls how command results are rendered.
	// Valid values: "table". Future values: "json", "csv".
	OutputFormat string `yaml:"output_format"`

	// DefaultDeviceID is used when --device-id is not provided.
	DefaultDeviceID string `yaml:"default_device_id,omitempty"`

	// DefaultLocationID is used when --location-id is not provided.
	DefaultLocationID string `yaml:"default_location_id,omitempty"`
}

func defaults() Config {
	return Config{
		OutputFormat: "table",
	}
}

// dir returns the flume configuration directory, following XDG Base Directory spec.
func dir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "flume"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".config", "flume"), nil
}

// Path returns the full path to the config file.
func Path() (string, error) {
	d, err := dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "config.yaml"), nil
}

// Load reads the config file. If the file does not exist it is created with
// defaults. A file that exists but cannot be parsed returns an error.
func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		cfg := defaults()
		if writeErr := write(&cfg, path); writeErr != nil {
			// Non-fatal: the user will still get working defaults this session.
			_, _ = fmt.Fprintf(os.Stderr, "warning: could not create config file %s: %v\n", path, writeErr)
		}
		return &cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	cfg := defaults()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	return &cfg, nil
}

// Save writes cfg to the config file.
func Save(cfg *Config) error {
	path, err := Path()
	if err != nil {
		return err
	}
	return write(cfg, path)
}

func write(cfg *Config, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write config %s: %w", path, err)
	}
	return nil
}
