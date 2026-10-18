package cmd

import (
	"bytes"
	"os"
	"slices"
	"strings"
	"testing"
)

// noAccountLeaves are the leaves that run without an account, with the reason.
var noAccountLeaves = map[string]string{
	"auth login":     "it creates the login; the account is its email argument",
	"auth import":    "it stores a login; the account comes from the imported token",
	"exercises list": "it reads the public exercise catalog, which needs no login",
	"completion":     "it prints a shell script made from the command tree",
}

// accountArgs replaces the golden args of a leaf whose golden file is a
// usage error, because Kong stops that error before Run asks for an account.
var accountArgs = map[string][]string{
	"training plan": {"training", "plan", "123"},
}

func TestEveryCommandHelps(t *testing.T) {
	cmds := append([]commandInfo{{Help: "Garmin Connect CLI"}}, collectCommands(newTestParser(t).Model.Node, nil)...)
	for _, c := range cmds {
		path := strings.Join(c.Path, " ")
		t.Run(strings.TrimSpace("gccli "+path), func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("XDG_CONFIG_HOME", home)
			var stdout, stderr bytes.Buffer
			code := execute(append(strings.Fields(path), "--help"), "test", "none", "never", &stdout, &stderr)
			if code != 0 {
				t.Errorf("exit %d, want 0; stderr:\n%s", code, stderr.String())
			}
			if want := strings.TrimSpace("Usage: gccli " + path); !strings.HasPrefix(stdout.String(), want+" ") && !strings.HasPrefix(stdout.String(), want+"\n") {
				t.Errorf("stdout does not start with %q:\n%s", want, stdout.String())
			}
			if c.Help == "" || !strings.Contains(stdout.String(), "\n"+c.Help+"\n") {
				t.Errorf("stdout does not show the help text %q:\n%s", c.Help, stdout.String())
			}
			if stderr.Len() != 0 {
				t.Errorf("stderr is not empty:\n%s", stderr.String())
			}
		})
	}
}

func TestEveryLeafNeedsAccount(t *testing.T) {
	leaves := leafCommands(t)
	for path := range noAccountLeaves {
		if !slices.Contains(leaves, path) {
			t.Errorf("noAccountLeaves names %q, which is not a leaf command", path)
		}
	}
	for path := range accountArgs {
		if !slices.Contains(leaves, path) {
			t.Errorf("accountArgs names %q, which is not a leaf command", path)
		}
	}
	modes := []struct {
		flag, stderr string
	}{
		{"--plain", errNoAccount().Error() + "\n"},
		{"--json", `{"error":{"code":"auth_required","message":"` + errNoAccount().Error() + `"}}` + "\n"},
	}
	for _, path := range leaves {
		args, in, err := goldenInput(path)
		if err != nil {
			t.Errorf("%q has no golden args: %v", path, err)
			continue
		}
		if a, ok := accountArgs[path]; ok {
			args = a
		}
		_, exempt := noAccountLeaves[path]
		if !exempt {
			in.routes = nil
		}
		for _, m := range modes {
			t.Run(path+"/"+strings.TrimPrefix(m.flag, "--"), func(t *testing.T) {
				h := newHarness(t, in)
				t.Setenv("GCCLI_ACCOUNT", "")
				if err := os.Unsetenv("GCCLI_ACCOUNT"); err != nil {
					t.Fatal(err)
				}
				var stdout, stderr bytes.Buffer
				code := execute(append([]string{"--color", "never", m.flag}, args...), "test", "none", "never", &stdout, &stderr)
				if exempt {
					if code != 0 {
						t.Errorf("exit %d without an account, want 0: fix the command or remove %q from noAccountLeaves; stderr:\n%s", code, path, stderr.String())
					}
					return
				}
				if code != kindAuth.exit {
					t.Errorf("exit %d, want %d: if the golden args stop before Run, add args that reach Run to accountArgs; stderr:\n%s", code, kindAuth.exit, stderr.String())
				}
				if stderr.String() != m.stderr {
					t.Errorf("stderr:\n%s\nwant:\n%s", stderr.String(), m.stderr)
				}
				if stdout.Len() != 0 {
					t.Errorf("stdout is not empty:\n%s", stdout.String())
				}
				if len(h.requested) != 0 {
					t.Errorf("sent requests without an account: %v", h.requested)
				}
			})
		}
	}
}

func goldenInput(path string) ([]string, harnessInput, error) {
	data, err := os.ReadFile(goldenFile(path))
	if err != nil {
		return nil, harnessInput{}, err
	}
	g, err := parseGolden(string(data))
	if err != nil {
		return nil, harnessInput{}, err
	}
	return g.input()
}
