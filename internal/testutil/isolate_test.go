package testutil

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bpauli/gccli/internal/config"
)

const isolationProbeArg = "isolation-probe"

func TestIsolatedMain_HidesCallerConfigAndEnv(t *testing.T) {
	if flag.Arg(0) == isolationProbeArg {
		f, err := config.Read()
		if err != nil {
			t.Fatal(err)
		}
		if f.DefaultAccount != "" {
			t.Errorf("config.Read() saw the caller's config: default_account = %q", f.DefaultAccount)
		}
		for _, kv := range os.Environ() {
			if strings.HasPrefix(kv, "GCCLI_") {
				t.Errorf("the caller's %s reached the test", kv)
			}
		}
		return
	}

	caller := t.TempDir()
	cfg := filepath.Join(caller, "gccli", "config.json")
	if err := os.MkdirAll(filepath.Dir(cfg), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte(`{"default_account": "caller@example.com"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(caller, "Library", "Application Support"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Dir(cfg), filepath.Join(caller, "Library", "Application Support", "gccli")); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestIsolatedMain_HidesCallerConfigAndEnv$", isolationProbeArg)
	cmd.Env = append(os.Environ(),
		"HOME="+caller,
		"XDG_CONFIG_HOME="+caller,
		"GCCLI_ACCOUNT=caller@example.com",
		"GCCLI_KEYRING_BACKEND=secret-service",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("the test process saw the caller's config or GCCLI_* variables: %v\n%s", err, out)
	}
}
