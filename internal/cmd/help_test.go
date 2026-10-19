package cmd

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
)

const helpGolden = "testdata/help.golden"

// TestGroupHelp pins the --help output of the root and of every group.
// The docs and the leaf goldens pin only the leaf help, so a changed
// group help passed every test before this one.
func TestGroupHelp(t *testing.T) {
	var want strings.Builder
	for _, path := range groupCommands(t) {
		args := append(strings.Fields(path), "--help")
		var stdout, stderr bytes.Buffer
		code := execute(args, "test", "none", "never", &stdout, &stderr)
		if code != 0 || stderr.Len() != 0 {
			t.Fatalf("gccli %s: exit %d, stderr:\n%s", strings.Join(args, " "), code, stderr.String())
		}
		fmt.Fprintf(&want, "$ gccli %s\n%s\n", strings.Join(args, " "), stdout.String())
	}
	if *update {
		if err := os.WriteFile(helpGolden, []byte(want.String()), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	have, err := os.ReadFile(helpGolden)
	if err != nil {
		t.Fatalf("%v: run go test ./internal/cmd -run TestGroupHelp -update", err)
	}
	if string(have) == want.String() {
		return
	}
	t.Errorf("%s is out of date: run go test ./internal/cmd -run TestGroupHelp -update", helpGolden)
	line, haveLine, wantLine := firstDifference(string(have), want.String())
	t.Errorf("first difference at line %d:\nhave: %s\nwant: %s", line, haveLine, wantLine)
}

// groupCommands returns the root as "" and then every group path.
func groupCommands(t *testing.T) []string {
	t.Helper()
	cmds := collectCommands(newTestParser(t).Model.Node, nil)
	groups := []string{""}
	for i, c := range cmds {
		if hasChildren(cmds, i) {
			groups = append(groups, strings.Join(c.Path, " "))
		}
	}
	return groups
}
