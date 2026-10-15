//go:build linux || darwin

package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

const fsizeHelperArg = "fsize-helper"

func TestUpdate_InterruptedWriteKeepsOldFile(t *testing.T) {
	if flag.Arg(0) == fsizeHelperArg {
		runFsizeLimitedUpdate(flag.Arg(1))
		return
	}

	path := tempConfigPath(t)
	old := `{"default_account": "old@example.com", "future_key": 1}` + "\n"
	writeConfig(t, path, old)

	cmd := exec.Command(os.Args[0], "-test.run=^TestUpdate_InterruptedWriteKeepsOldFile$", fsizeHelperArg, os.Getenv("HOME"))
	out, err := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 7 {
		t.Fatalf("helper did not report a failed write: err=%v\n%s", err, out)
	}
	if !strings.Contains(string(out), "file too large") {
		t.Fatalf("helper write failed for another reason:\n%s", out)
	}
	if !strings.Contains(string(out), filepath.Dir(path)) {
		t.Fatalf("helper did not write to %s:\n%s", filepath.Dir(path), out)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != old {
		t.Errorf("config.json after interrupted write = %q, want the old content %q", data, old)
	}

	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != "config.json" && e.Name() != "config.lock" {
			t.Errorf("leftover file in config dir: %s", e.Name())
		}
	}
}

func runFsizeLimitedUpdate(home string) {
	// IsolatedMain in this child moved HOME away from the parent's config.
	_ = os.Setenv("HOME", home)
	_ = os.Setenv("XDG_CONFIG_HOME", home)
	var lim syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_FSIZE, &lim); err != nil {
		fmt.Println("getrlimit:", err)
		os.Exit(2)
	}
	lim.Cur = 32
	if err := syscall.Setrlimit(syscall.RLIMIT_FSIZE, &lim); err != nil {
		fmt.Println("setrlimit:", err)
		os.Exit(2)
	}
	err := Update(func(f *File) {
		f.DefaultAccount = "new@example.com"
		f.ActivitySummary = map[string][]string{"running": {strings.Repeat("x", 256)}}
	})
	if err != nil {
		fmt.Println("update:", err)
		os.Exit(7)
	}
	os.Exit(0)
}
