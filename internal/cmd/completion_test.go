package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alecthomas/kong"

	"github.com/bpauli/gccli/internal/units"
)

func newTestParser(t *testing.T) *kong.Kong {
	t.Helper()
	var cli CLI
	parser, err := kong.New(&cli,
		kong.Name("gccli"),
		kong.Description("Garmin Connect CLI"),
		kong.Vars{"units": strings.Join(units.PreferenceNames(), ",")},
	)
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}
	return parser
}

func TestCollectCommands(t *testing.T) {
	parser := newTestParser(t)
	cmds := collectCommands(parser.Model.Node, nil)

	// Should have top-level commands.
	found := map[string]bool{}
	for _, cmd := range cmds {
		found[cmd.Path[0]] = true
	}

	for _, want := range []string{"auth", "activities", "activity", "workouts", "health", "body", "devices", "gear", "goals", "badges", "challenges", "records", "profile", "hydration", "nutrition", "training", "wellness", "events", "courses", "exercises", "reload", "completion"} {
		if !found[want] {
			t.Errorf("expected command %q in collectCommands output", want)
		}
	}
}

func TestCollectCommands_IncludesSubcommands(t *testing.T) {
	parser := newTestParser(t)
	cmds := collectCommands(parser.Model.Node, nil)

	// auth should have subcommands like login, status, remove, token.
	subCmds := map[string]bool{}
	for _, cmd := range cmds {
		if len(cmd.Path) == 2 && cmd.Path[0] == "auth" {
			subCmds[cmd.Path[1]] = true
		}
	}

	for _, want := range []string{"login", "status", "remove", "token", "export", "import"} {
		if !subCmds[want] {
			t.Errorf("expected auth subcommand %q", want)
		}
	}
}

func TestCollectFlags(t *testing.T) {
	parser := newTestParser(t)
	flags := collectFlags(parser.Model.Node)

	found := map[string]bool{}
	for _, f := range flags {
		found[f.Long] = true
	}

	for _, want := range []string{"json", "plain", "color", "account", "units"} {
		if !found[want] {
			t.Errorf("expected flag --%s in root flags", want)
		}
	}
}

func TestCollectFlags_EnumValues(t *testing.T) {
	parser := newTestParser(t)
	flags := collectFlags(parser.Model.Node)

	wantEnums := map[string][]string{
		"color": {"auto", "always", "never"},
		"units": {"auto", "metric", "statute_us", "statute_uk"},
	}
	for name, wantValues := range wantEnums {
		found := false
		for _, f := range flags {
			if f.Long != name {
				continue
			}
			found = true
			enumSet := map[string]bool{}
			for _, v := range f.Enum {
				enumSet[v] = true
			}
			for _, want := range wantValues {
				if !enumSet[want] {
					t.Errorf("expected enum value %q for --%s flag", want, name)
				}
			}
		}
		if !found {
			t.Errorf("--%s flag not found", name)
		}
	}
}

func TestCollectFlags_ShortFlags(t *testing.T) {
	parser := newTestParser(t)
	flags := collectFlags(parser.Model.Node)

	for _, f := range flags {
		if f.Long == "json" {
			if f.Short != 'j' {
				t.Errorf("expected short flag -j for --json, got %c", f.Short)
			}
			return
		}
	}
	t.Fatal("--json flag not found")
}

func TestCollectFlags_ExcludesHelp(t *testing.T) {
	parser := newTestParser(t)
	flags := collectFlags(parser.Model.Node)

	for _, f := range flags {
		if f.Long == "help" {
			t.Fatal("--help flag should be excluded from collectFlags")
		}
	}
}

func TestGenerateBash(t *testing.T) {
	parser := newTestParser(t)
	script := generateBash(parser)

	checks := []string{
		"_gccli_completions",
		"complete -F _gccli_completions gccli",
		"auth",
		"activities",
		"--json",
		"--color",
		"auto always never",
	}

	for _, want := range checks {
		if !strings.Contains(script, want) {
			t.Errorf("bash script missing %q", want)
		}
	}
}

func TestGenerateZsh(t *testing.T) {
	parser := newTestParser(t)
	script := generateZsh(parser)

	checks := []string{
		"#compdef gccli",
		"_gccli",
		"compdef _gccli gccli",
		"auth",
		"activities",
		"--json",
		"--color",
		"auto always never",
	}

	for _, want := range checks {
		if !strings.Contains(script, want) {
			t.Errorf("zsh script missing %q", want)
		}
	}
}

func TestGenerateFish(t *testing.T) {
	parser := newTestParser(t)
	script := generateFish(parser)

	checks := []string{
		"complete -c gccli",
		"__fish_use_subcommand",
		"auth",
		"activities",
		"-l json",
		"-l color",
		"auto always never",
	}

	for _, want := range checks {
		if !strings.Contains(script, want) {
			t.Errorf("fish script missing %q", want)
		}
	}
}

func TestGeneratePowerShell(t *testing.T) {
	parser := newTestParser(t)
	script := generatePowerShell(parser)

	checks := []string{
		"Register-ArgumentCompleter -CommandName gccli",
		"CompletionResult",
		"auth",
		"activities",
		"--json",
		"--color",
		"auto",
		"always",
		"never",
	}

	for _, want := range checks {
		if !strings.Contains(script, want) {
			t.Errorf("powershell script missing %q", want)
		}
	}
}

func TestFindNode(t *testing.T) {
	parser := newTestParser(t)

	// Find auth node.
	node := findNode(parser.Model.Node, []string{"auth"})
	if node == nil {
		t.Fatal("expected to find auth node")
	}
	if node.Name != "auth" {
		t.Errorf("expected node name 'auth', got %q", node.Name)
	}

	// Find auth login node.
	node = findNode(parser.Model.Node, []string{"auth", "login"})
	if node == nil {
		t.Fatal("expected to find auth login node")
	}
	if node.Name != "login" {
		t.Errorf("expected node name 'login', got %q", node.Name)
	}

	// Non-existent path returns nil.
	node = findNode(parser.Model.Node, []string{"nonexistent"})
	if node != nil {
		t.Error("expected nil for non-existent path")
	}
}

func TestCompletionCommand_Execute(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		t.Run(shell, func(t *testing.T) {
			code := Execute([]string{"completion", shell}, "1.0.0", "abc123", "2024-01-01")
			if code != 0 {
				t.Fatalf("expected exit code 0 for completion %s, got %d", shell, code)
			}
		})
	}
}

func TestEscapeShellString(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"it's", "it'\\''s"},
		{"no quotes", "no quotes"},
	}
	for _, tt := range tests {
		got := escapeShellString(tt.input)
		if got != tt.want {
			t.Errorf("escapeShellString(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestGenerateBash_CompletesAfterFlagValue(t *testing.T) {
	bashPath, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not on PATH")
	}

	dir := t.TempDir()
	script := filepath.Join(dir, "gccli.bash")
	if err := os.WriteFile(script, []byte(generateBash(newTestParser(t))), 0o600); err != nil {
		t.Fatal(err)
	}
	driver := filepath.Join(dir, "driver.sh")
	driverSrc := `source "$1"; shift
COMP_WORDS=("$@")
COMP_CWORD=$(( ${#COMP_WORDS[@]} - 1 ))
COMPREPLY=()
_gccli_completions
printf '%s\n' "${COMPREPLY[*]}"
`
	if err := os.WriteFile(driver, []byte(driverSrc), 0o600); err != nil {
		t.Fatal(err)
	}

	const activitiesSubcommands = "list count search types"
	tests := []struct {
		name  string
		words []string
		want  string
	}{
		{"no flag", []string{"gccli", "activities", ""}, activitiesSubcommands},
		{"global value flag", []string{"gccli", "--account", "x@example.com", "activities", ""}, activitiesSubcommands},
		{"global value flag with equals", []string{"gccli", "--account=x@example.com", "activities", ""}, activitiesSubcommands},
		{"global enum flag", []string{"gccli", "--units", "metric", "activities", ""}, activitiesSubcommands},
		{"global bool flag", []string{"gccli", "--json", "activities", ""}, activitiesSubcommands},
		{"command value flag", []string{"gccli", "activities", "list", "--limit", "5", "--t"}, "--type"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]string{"--norc", "--noprofile", driver, script}, tt.words...)
			out, err := exec.Command(bashPath, args...).CombinedOutput()
			if err != nil {
				t.Fatalf("bash failed: %v\n%s", err, out)
			}
			got := strings.TrimSpace(string(out))
			if got != tt.want {
				t.Errorf("words %q: COMPREPLY = %q, want %q", tt.words, got, tt.want)
			}
		})
	}
}

func TestCollectFlags_ValueTakingIsConsistentAcrossCommands(t *testing.T) {
	parser := newTestParser(t)
	root := parser.Model.Node

	type seenAt struct {
		isBool bool
		path   string
	}
	first := map[string]seenAt{}
	check := func(path string, node *kong.Node) {
		for _, f := range collectFlags(node) {
			words := []string{"--" + f.Long}
			if f.Short != 0 {
				words = append(words, fmt.Sprintf("-%c", f.Short))
			}
			for _, w := range words {
				prev, ok := first[w]
				if !ok {
					first[w] = seenAt{isBool: f.IsBool, path: path}
					continue
				}
				if prev.isBool != f.IsBool {
					t.Errorf("flag %s is boolean in one command (%q or %q) and takes a value in the other; generateBash uses one flat set of value-taking flags, which would skip or keep the wrong word", w, prev.path, path)
				}
			}
		}
	}
	check("(root)", root)
	for _, cmd := range collectCommands(root, nil) {
		if node := findNode(root, cmd.Path); node != nil {
			check(strings.Join(cmd.Path, " "), node)
		}
	}
}
