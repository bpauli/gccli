//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/joho/godotenv"
)

// LoadEnv loads .env from the project root and returns the GARMIN_EMAIL
// and GARMIN_PASSWORD values. It calls t.Skip() if the file is missing
// or the variables are not set, so tests degrade gracefully in CI without
// credentials.
func LoadEnv(t *testing.T) (email, password string) {
	t.Helper()

	envPath := findEnvFile()
	if envPath != "" {
		// Best-effort load; ignore errors (vars might be set externally).
		_ = godotenv.Load(envPath)
	}

	email = os.Getenv("GARMIN_EMAIL")
	password = os.Getenv("GARMIN_PASSWORD")

	if email == "" || password == "" {
		t.Skip("skipping E2E test: GARMIN_EMAIL and GARMIN_PASSWORD not set (create .env or export vars)")
	}
	return email, password
}

// findEnvFile walks up from the current source file to find .env in the
// project root.
func findEnvFile() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	dir := filepath.Dir(filename)
	for {
		candidate := filepath.Join(dir, ".env")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}
