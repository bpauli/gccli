package cmd

import (
	"strings"
	"testing"

	"github.com/alecthomas/kong"
)

func newTestParser(t *testing.T) *kong.Kong {
	t.Helper()
	var cli CLI
	parser, err := kong.New(&cli,
		kong.Name("gccli"),
		kong.Description("Garmin Connect CLI"),
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

	for _, want := range []string{"auth", "activities", "activity", "workouts", "health", "body", "devices", "gear", "goals", "badges", "challenges", "records", "profile", "hydration", "training", "wellness", "events", "courses", "exercises", "reload", "completion"} {
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

	for _, want := range []string{"login", "status", "remove", "token"} {
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

	for _, want := range []string{"json", "plain", "color", "account"} {
		if !found[want] {
			t.Errorf("expected flag --%s in root flags", want)
		}
	}
}

func TestCollectFlags_EnumValues(t *testing.T) {
	parser := newTestParser(t)
	flags := collectFlags(parser.Model.Node)

	for _, f := range flags {
		if f.Long == "color" {
			if len(f.Enum) == 0 {
				t.Fatal("expected enum values for --color flag")
			}
			enumSet := map[string]bool{}
			for _, v := range f.Enum {
				enumSet[v] = true
			}
			for _, want := range []string{"auto", "always", "never"} {
				if !enumSet[want] {
					t.Errorf("expected enum value %q for --color flag", want)
				}
			}
			return
		}
	}
	t.Fatal("--color flag not found")
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
