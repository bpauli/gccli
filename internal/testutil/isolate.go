package testutil

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// IsolatedMain runs the tests of a package with HOME and XDG_CONFIG_HOME on a
// new empty directory and with every GCCLI_* variable unset, then exits. Call
// it from TestMain in each package whose tests link internal/config, so that
// a test can reach the real config, keyring file or environment of the person
// who runs it only when it sets them on purpose.
func IsolatedMain(m *testing.M) {
	os.Exit(runIsolated(m))
}

func runIsolated(m *testing.M) int {
	home, err := os.MkdirTemp("", "gccli-test-home-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "testutil:", err)
		return 1
	}
	defer func() { _ = os.RemoveAll(home) }()

	for _, kv := range os.Environ() {
		if name, _, _ := strings.Cut(kv, "="); strings.HasPrefix(name, "GCCLI_") {
			_ = os.Unsetenv(name)
		}
	}
	_ = os.Setenv("HOME", home)
	_ = os.Setenv("XDG_CONFIG_HOME", home)
	return m.Run()
}
