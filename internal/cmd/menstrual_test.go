package cmd

import (
	"slices"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
)

// The enum tags of the set command and menstrualValues must list the same
// values, or a flag value maps to no Garmin value, or a Garmin value has no
// flag value.
func TestMenstrualLogSetEnumsMatchGarminValues(t *testing.T) {
	node := newTestParser(t).Model.Node
	for _, name := range []string{"wellness", "menstrual-log", "set"} {
		i := slices.IndexFunc(node.Children, func(c *kong.Node) bool { return c.Name == name })
		if i < 0 {
			t.Fatalf("no command %q", name)
		}
		node = node.Children[i]
	}

	seen := map[string]bool{}
	for _, flag := range node.Flags {
		if flag.Enum == "" {
			continue
		}
		field := strings.ReplaceAll(flag.Name, "-", "_")
		seen[field] = true
		garmin, ok := menstrualValues[field]
		if !ok {
			t.Errorf("--%s has an enum but menstrualValues has no %q", flag.Name, field)
			continue
		}
		var want []string
		for _, g := range garmin {
			want = append(want, menstrualFlagValue(g))
		}
		got := slices.DeleteFunc(strings.Split(flag.Enum, ","), func(v string) bool { return v == "" })
		if !slices.Equal(got, want) {
			t.Errorf("--%s enum = %v, menstrualValues[%q] gives %v", flag.Name, got, field, want)
		}
	}
	for field := range menstrualValues {
		if !seen[field] {
			t.Errorf("menstrualValues has %q but the set command has no enum flag for it", field)
		}
	}
}
