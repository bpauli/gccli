package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bpauli/gccli/internal/config"
)

func TestTempConfigDir_IgnoresOuterXDG(t *testing.T) {
	outer := t.TempDir()
	sentinelPath := filepath.Join(outer, "gccli", "config.json")
	sentinel := []byte(`{"default_account": "sentinel@example.org"}` + "\n")
	if err := os.MkdirAll(filepath.Dir(sentinelPath), 0o700); err != nil {
		t.Fatalf("mkdir outer config dir: %v", err)
	}
	if err := os.WriteFile(sentinelPath, sentinel, 0o600); err != nil {
		t.Fatalf("write sentinel: %v", err)
	}
	t.Setenv("XDG_CONFIG_HOME", outer)

	dir := TempConfigDir(t)

	if err := config.Write(&config.File{DefaultAccount: "test@example.com"}); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got, err := os.ReadFile(sentinelPath)
	if err != nil {
		t.Fatalf("read sentinel: %v", err)
	}
	if string(got) != string(sentinel) {
		t.Errorf("outer config was overwritten:\n got: %s\nwant: %s", got, sentinel)
	}

	written, err := config.ReadFrom(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatalf("ReadFrom helper dir: %v", err)
	}
	if written.DefaultAccount != "test@example.com" {
		t.Errorf("helper dir DefaultAccount = %q, want test@example.com", written.DefaultAccount)
	}
}
