package testutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/bpauli/gccli/internal/config"
)

// TempConfigDir creates a temporary directory structure suitable for config
// tests and sets HOME so that os.UserConfigDir() resolves into it.
// Returns the path to the app config directory (e.g., {tmp}/Library/Application Support/gccli on macOS).
// Cleanup is automatic via t.TempDir.
func TempConfigDir(t *testing.T) string {
	t.Helper()
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Ensure the config dir exists.
	dir, err := config.ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir: %v", err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	return dir
}

// TempConfigFile creates a temporary config file with the given File contents
// and returns the file path. Sets HOME so config.Read() finds it.
func TempConfigFile(t *testing.T, f *config.File) string {
	t.Helper()
	dir := TempConfigDir(t)
	path := filepath.Join(dir, "config.json")
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}
