package cmd

import (
	"bytes"
	"os"
	"path/filepath"
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
	for _, path := range leaves {
		args, in, err := goldenInput(goldenFile(path))
		if err != nil {
			t.Errorf("%q has no golden args: %v", path, err)
			continue
		}
		if a, ok := accountArgs[path]; ok {
			args = a
		}
		if _, exempt := noAccountLeaves[path]; exempt {
			for _, flag := range []string{"--plain", "--json"} {
				t.Run(path+"/"+strings.TrimPrefix(flag, "--"), func(t *testing.T) {
					_, stderr, code, _ := runWithoutAccount(t, in, flag, args)
					if code != 0 {
						t.Errorf("exit %d without an account, want 0: fix the command or remove %q from noAccountLeaves; stderr:\n%s", code, path, stderr)
					}
				})
			}
			continue
		}
		in.routes = nil
		checkNeedsAccount(t, path, args, in)
	}
}

// A positional date is checked at parse time, so every bad-date golden exits
// 2 without an account and sends no request. A leaf that checks its date
// after the login would exit 3 here.
// TestBadDatesAndTimesAreCheckedBeforeAccount runs every bad-date,
// empty-date, bad-time and empty-time golden without an account: a bad
// date or time must be a usage error at parse time, before the login.
func TestBadDatesAndTimesAreCheckedBeforeAccount(t *testing.T) {
	var files []string
	for _, suffix := range []string{"bad-date", "empty-date", "bad-time", "empty-time"} {
		matches, err := filepath.Glob(filepath.Join(goldenDir, "*"+suffix+".golden"))
		if err != nil {
			t.Fatal(err)
		}
		if len(matches) == 0 {
			t.Fatalf("no %s golden in %s", suffix, goldenDir)
		}
		files = append(files, matches...)
	}
	for _, file := range files {
		args, in, err := goldenInput(file)
		if err != nil {
			t.Errorf("%s: %v", file, err)
			continue
		}
		in.routes = nil
		for _, flag := range []string{"--plain", "--json"} {
			t.Run(filepath.Base(file)+"/"+strings.TrimPrefix(flag, "--"), func(t *testing.T) {
				stdout, stderr, code, requests := runWithoutAccount(t, in, flag, args)
				if code != kindUsage.exit {
					t.Errorf("exit %d, want %d: a bad date or time must fail at parse time, before the login; stderr:\n%s", code, kindUsage.exit, stderr)
				}
				if !strings.HasPrefix(stderr, "Usage: gccli ") || !strings.Contains(stderr, "\ngccli: error: ") {
					t.Errorf("stderr is not the usage block with the error line:\n%s", stderr)
				}
				if stdout != "" {
					t.Errorf("stdout is not empty:\n%s", stdout)
				}
				if len(requests) != 0 {
					t.Errorf("sent %d requests, first: %s", len(requests), requests[0].route)
				}
			})
		}
	}
}

func checkNeedsAccount(t *testing.T, path string, args []string, in harnessInput) {
	t.Helper()
	modes := []struct {
		flag, stderr string
	}{
		{"--plain", errNoAccount().Error() + "\n"},
		{"--json", `{"error":{"code":"auth_required","message":"` + errNoAccount().Error() + `"}}` + "\n"},
	}
	for _, m := range modes {
		t.Run(path+"/"+strings.TrimPrefix(m.flag, "--"), func(t *testing.T) {
			stdout, stderr, code, requests := runWithoutAccount(t, in, m.flag, args)
			if code != kindAuth.exit {
				t.Errorf("exit %d, want %d: if the golden args stop before Run, add args that reach Run to accountArgs; stderr:\n%s", code, kindAuth.exit, stderr)
			}
			if stderr != m.stderr {
				t.Errorf("stderr:\n%s\nwant:\n%s", stderr, m.stderr)
			}
			if stdout != "" {
				t.Errorf("stdout is not empty:\n%s", stdout)
			}
			if len(requests) != 0 {
				t.Errorf("sent %d requests without an account, first: %s", len(requests), requests[0].route)
			}
		})
	}
}

func runWithoutAccount(t *testing.T, in harnessInput, flag string, args []string) (stdout, stderr string, code int, requests []loggedRequest) {
	t.Helper()
	h := newHarness(t, in)
	t.Setenv("GCCLI_ACCOUNT", "")
	if err := os.Unsetenv("GCCLI_ACCOUNT"); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	code = execute(append([]string{"--color", "never", flag}, args...), "test", "none", "never", &out, &errOut)
	return out.String(), errOut.String(), code, h.requests()
}

func goldenInput(file string) ([]string, harnessInput, error) {
	g, err := readGolden(file)
	if err != nil {
		return nil, harnessInput{}, err
	}
	return g.input()
}
