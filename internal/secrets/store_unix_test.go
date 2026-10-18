//go:build unix

package secrets

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/bpauli/gccli/internal/testutil"
)

func TestFileBackendKeepsTempDirOfWriteInFlight(t *testing.T) {
	testutil.TempConfigDir(t)
	credDir, left := leaveTempDirs(t)
	lock, err := os.OpenFile(filepath.Join(credDir, ".set.lock"), os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_SH); err != nil {
		t.Fatal(err)
	}
	s, err := Open("file")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	if err := s.Set("t@example.com", []byte("data")); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := s.Delete("t@example.com"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	for _, dir := range left {
		if _, err := os.Stat(dir); err != nil {
			t.Errorf("removed %s while a write held the lock: %v", filepath.Base(dir), err)
		}
	}

	_ = lock.Close()
	if err := s.Set("t@example.com", []byte("data")); err != nil {
		t.Fatalf("Set: %v", err)
	}
	assertGone(t, left...)
}
