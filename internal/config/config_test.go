package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoad_CreatesDefaultFileWhenMissing(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.OutputFormat != "table" {
		t.Errorf("OutputFormat = %q, want %q", cfg.OutputFormat, "table")
	}

	// File should now exist on disk.
	path, _ := Path()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("config file not created: %v", err)
	}
}

func TestLoad_ReadsExistingFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	dir := filepath.Join(tmp, "flume")
	_ = os.MkdirAll(dir, 0o700)
	content := "output_format: table\ndefault_device_id: \"123456\"\ndefault_location_id: \"789\"\n"
	_ = os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(content), 0o600)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.DefaultDeviceID != "123456" {
		t.Errorf("DefaultDeviceID = %q, want %q", cfg.DefaultDeviceID, "123456")
	}
	if cfg.DefaultLocationID != "789" {
		t.Errorf("DefaultLocationID = %q, want %q", cfg.DefaultLocationID, "789")
	}
}

func TestLoad_ErrorOnBadYAML(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	dir := filepath.Join(tmp, "flume")
	_ = os.MkdirAll(dir, 0o700)
	_ = os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(":\nnot: [valid yaml: {"), 0o600)

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error on bad YAML, got nil")
	}
}

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	original := &Config{
		OutputFormat:      "table",
		DefaultDeviceID:   "999000111222333",
		DefaultLocationID: "42",
	}
	if err := Save(original); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if loaded.OutputFormat != original.OutputFormat {
		t.Errorf("OutputFormat = %q, want %q", loaded.OutputFormat, original.OutputFormat)
	}
	if loaded.DefaultDeviceID != original.DefaultDeviceID {
		t.Errorf("DefaultDeviceID = %q, want %q", loaded.DefaultDeviceID, original.DefaultDeviceID)
	}
	if loaded.DefaultLocationID != original.DefaultLocationID {
		t.Errorf("DefaultLocationID = %q, want %q", loaded.DefaultLocationID, original.DefaultLocationID)
	}
}

func TestSave_SetsFilePermissions(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	if err := Save(&Config{OutputFormat: "table"}); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	path, _ := Path()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("config file not found: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("config file permissions = %o, want 0600", perm)
	}
}

func TestLoad_DefaultWarning(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Warning != 90 {
		t.Errorf("Warning = %d, want 90", cfg.Warning)
	}
}

func TestSaveAndLoad_WithConsumables(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	installed := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	w := 85
	original := &Config{
		OutputFormat: "table",
		Warning:      90,
		Consumables: map[string]map[string]*Consumable{
			"dev1": {
				"charcoal": {Installed: installed, Expires: 15000},
				"sediment": {Installed: installed, Expires: 10000, Warning: &w},
			},
		},
	}
	if err := Save(original); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	charcoal := loaded.Consumables["dev1"]["charcoal"]
	if charcoal == nil {
		t.Fatal("charcoal consumable missing after round-trip")
	}
	if !charcoal.Installed.Equal(installed) {
		t.Errorf("Installed = %v, want %v", charcoal.Installed, installed)
	}
	if charcoal.Expires != 15000 {
		t.Errorf("Expires = %d, want 15000", charcoal.Expires)
	}
	if charcoal.Warning != nil {
		t.Errorf("Warning = %v, want nil", charcoal.Warning)
	}

	sediment := loaded.Consumables["dev1"]["sediment"]
	if sediment == nil {
		t.Fatal("sediment consumable missing after round-trip")
	}
	if sediment.Warning == nil || *sediment.Warning != 85 {
		t.Errorf("sediment Warning = %v, want 85", sediment.Warning)
	}
}

func TestLoad_OmitEmptyFieldsOnDisk(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	// Default config has no device/location ID; they should be omitted from YAML.
	cfg := defaults()
	if err := Save(&cfg); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	path, _ := Path()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)
	if strings.Contains(content, "default_device_id") {
		t.Errorf("expected default_device_id omitted from YAML, got: %s", content)
	}
	if strings.Contains(content, "default_location_id") {
		t.Errorf("expected default_location_id omitted from YAML, got: %s", content)
	}
}
