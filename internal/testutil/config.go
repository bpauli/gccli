package testutil

import (
	"os"
	"testing"

	"github.com/bpauli/gccli/internal/config"
)

// TempConfigDir creates a temporary directory structure suitable for config
// tests and points os.UserConfigDir() into it. It sets both HOME (used on
// macOS) and XDG_CONFIG_HOME (used on Linux, where it takes precedence over
// HOME), so no test can reach the developer's real config directory.
// Returns the path to the app config directory (e.g., {tmp}/Library/Application Support/gccli on macOS).
// Cleanup is automatic via t.TempDir.
func TempConfigDir(t *testing.T) string {
	t.Helper()
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", tmpHome)

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
