package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/alecthomas/kong"

	"github.com/bpauli/gccli/internal/units"
)

func newTestParser(t *testing.T) *kong.Kong {
	t.Helper()
	cli := CLI{Plugins: attach(commands)}
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

// Kong prints the help of a long path on the next line, so the help part
// is optional.
var helpLeaf = regexp.MustCompile(`(?m)^  ([a-z0-9-]+(?: [a-z0-9-]+)*)(?: {2,}.*)?$`)

func TestCollectCommandsMatchesHelp(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	var stdout, stderr bytes.Buffer
	if code := execute([]string{"--help"}, "test", "none", "never", &stdout, &stderr); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}
	_, commandsSection, ok := strings.Cut(stdout.String(), "\nCommands:\n")
	if !ok {
		t.Fatalf("root help has no Commands section:\n%s", stdout.String())
	}
	var fromHelp []string
	for _, m := range helpLeaf.FindAllStringSubmatch(commandsSection, -1) {
		fromHelp = append(fromHelp, m[1])
	}
	if len(fromHelp) < 100 {
		t.Fatalf("root help lists %d leaves, want more than 100:\n%s", len(fromHelp), commandsSection)
	}
	if got := leafCommands(t); !slices.Equal(got, fromHelp) {
		t.Errorf("collectCommands leaves differ from the root help\ngot:  %q\nhelp: %q", got, fromHelp)
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

// Word-break sets of a real interactive bash. bash-completion removes "@".
const (
	plainBashWordbreaks      = " \t\n\"'@><=;|&(:"
	bashCompletionWordbreaks = " \t\n\"'><=;|&(:"
	whitespaceOnly           = ""
	noQuoteWordbreaks        = " \t\n@><=;|&(:"
)

func bashCompWords(line, wordbreaks string) []string {
	var words []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			words = append(words, cur.String())
			cur.Reset()
		}
	}
	for _, r := range line {
		switch {
		case r == ' ' || r == '\t':
			flush()
		case strings.ContainsRune(wordbreaks, r):
			flush()
			words = append(words, string(r))
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	if strings.HasSuffix(line, " ") {
		words = append(words, "")
	}
	return words
}

func TestBashCompWords(t *testing.T) {
	recordedCompWords := []struct {
		line       string
		wordbreaks string
		want       []string
	}{
		{"gccli activities ", plainBashWordbreaks, []string{"gccli", "activities", ""}},
		{"gccli activity download 123 --f", plainBashWordbreaks, []string{"gccli", "activity", "download", "123", "--f"}},
		{"gccli --account=x@example.com activities ", plainBashWordbreaks, []string{"gccli", "--account", "=", "x", "@", "example.com", "activities", ""}},
		{"gccli --account=x@example.com activities ", bashCompletionWordbreaks, []string{"gccli", "--account", "=", "x@example.com", "activities", ""}},
		{"gccli --units=", plainBashWordbreaks, []string{"gccli", "--units", "="}},
		{"gccli --units=m", plainBashWordbreaks, []string{"gccli", "--units", "=", "m"}},
		{"gccli --account x@example.com activities ", plainBashWordbreaks, []string{"gccli", "--account", "x", "@", "example.com", "activities", ""}},
		{"gccli --account x@example.com activities ", bashCompletionWordbreaks, []string{"gccli", "--account", "x@example.com", "activities", ""}},
		{"gccli --account=x@example.com activities ", whitespaceOnly, []string{"gccli", "--account=x@example.com", "activities", ""}},
		{"gccli --units=m", whitespaceOnly, []string{"gccli", "--units=m"}},
	}
	for _, tt := range recordedCompWords {
		got := bashCompWords(tt.line, tt.wordbreaks)
		if strings.Join(got, "\x00") != strings.Join(tt.want, "\x00") {
			t.Errorf("bashCompWords(%q) = %q, want %q", tt.line, got, tt.want)
		}
	}
}

func TestGenerateBash_Completions(t *testing.T) {
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
COMP_WORDBREAKS=$1
COMP_LINE=$2$3
COMP_POINT=${#2}
# bash before 4.3 reports COMP_POINT in bytes.
if (( BASH_VERSINFO[0] < 4 || BASH_VERSINFO[0] == 4 && BASH_VERSINFO[1] < 3 )); then
    COMP_POINT=$(LC_ALL=C; echo ${#2})
fi
COMP_CWORD=$4
shift 4
COMP_WORDS=("$@")
COMPREPLY=()
_gccli_completions
printf '%s\n' "${COMPREPLY[*]}"
`
	if err := os.WriteFile(driver, []byte(driverSrc), 0o600); err != nil {
		t.Fatal(err)
	}

	const (
		activitiesSubcommands = "list count search types pending"
		authSubcommands       = "login status remove token export import"
		unitValues            = "auto metric statute_us statute_uk"
	)
	tests := []struct {
		name                      string
		beforeCursor, afterCursor string
		want                      string
	}{
		{"subcommands", "gccli activities ", "", activitiesSubcommands},
		{"flag after positional", "gccli activity download 123 --f", "", "--format"},
		{"flag without positional", "gccli activity download --f", "", "--format"},
		{"positional goes to the default subcommand", "gccli activities 5 --t", "", "--type"},
		{"global value flag", "gccli --account x@example.com activities ", "", activitiesSubcommands},
		{"global value flag with equals", "gccli --account=x@example.com activities ", "", activitiesSubcommands},
		{"value with colon", "gccli --account=user:x activities ", "", activitiesSubcommands},
		{"global enum flag", "gccli --units metric activities ", "", activitiesSubcommands},
		{"global bool flag", "gccli --json activities ", "", activitiesSubcommands},
		{"enum values", "gccli --units ", "", unitValues},
		{"enum values after equals", "gccli --units=", "", unitValues},
		{"enum value prefix after equals", "gccli --units=m", "", "metric"},
		{"enum values of the command, not of a flag with the same name", "gccli body add-weight --unit ", "", "kg lbs"},
		{"enum value prefix after equals and positional", "gccli activity download 123 --format=g", "", "gpx"},
		{"enum values of the command after a positional", "gccli courses download 1 --format ", "", "fit gpx"},
		{"value flag wants a value", "gccli --account ", "", ""},
		{"split inside a value", "gccli --account=x@", "", ""},
		{"command value flag", "gccli activities list --limit 5 --t", "", "--type"},
		{"cursor before a word", "gccli activit", " list", "activities activity"},
		{"cursor inside a word", "gccli acti", "vit", "activities activity"},
		{"cursor before a flag after positional", "gccli activity download 123 --f", " --json", "--format"},
		{"open double quote", `gccli "au`, "", "auth"},
		{"open single quote", `gccli 'au`, "", "auth"},
		{"open quote in a subcommand", `gccli auth "st`, "", "status"},
		{"open quote in an enum value", `gccli activity download 123 --format "g`, "", "gpx"},
		{"open quote after equals", `gccli --units="m`, "", "metric"},
		{"quoted command before the cursor", `gccli "auth" `, "", authSubcommands},
		{"single-quoted command before a word", `gccli 'auth' st`, "", "status"},
		{"non-ASCII value before the cursor inside the line", "gccli --account Jürgen au", " status", "auth"},
		// bash replaces all text after an open quote, so a reply that does
		// not start with that text would delete what the user typed.
		{"open quote with a space before the cursor", `gccli "auth st`, "", ""},
		{"open quote with equals before the cursor", `gccli '--units=m`, "", ""},
		{"open quote with a space before the cursor in an enum value", `gccli activity download 123 "--format g`, "", ""},
		{"open quote in a value with a space", `gccli --account "Jane Doe au`, "", ""},
		{"apostrophe in a value", `gccli --account o'neil activities li`, "", ""},
		// bash keeps an escaped break character in the word it replaces, so
		// the word is Jane\ au and a reply for au would delete Jane\ .
		{"escaped space in a value", `gccli --account Jane\ au`, "", ""},
		{"escaped space in a positional", `gccli Jane\ au`, "", ""},
	}
	// bash 3.2 exports the plain set but splits at whitespace only.
	setups := []struct{ name, wordbreaks, splitAt string }{
		{"plain bash", plainBashWordbreaks, plainBashWordbreaks},
		{"bash-completion", bashCompletionWordbreaks, bashCompletionWordbreaks},
		{"bash 3.2", plainBashWordbreaks, whitespaceOnly},
		{"quotes removed from COMP_WORDBREAKS", noQuoteWordbreaks, noQuoteWordbreaks},
	}
	for _, setup := range setups {
		for _, tt := range tests {
			t.Run(setup.name+"/"+tt.name, func(t *testing.T) {
				cword := len(bashCompWords(tt.beforeCursor, setup.splitAt)) - 1
				args := []string{"--norc", "--noprofile", driver, script, setup.wordbreaks, tt.beforeCursor, tt.afterCursor, strconv.Itoa(cword)}
				args = append(args, bashCompWords(tt.beforeCursor+tt.afterCursor, setup.splitAt)...)
				out, err := exec.Command(bashPath, args...).CombinedOutput()
				if err != nil {
					t.Fatalf("bash failed: %v\n%s", err, out)
				}
				got := strings.TrimSpace(string(out))
				if got != tt.want {
					t.Errorf("line %q: COMPREPLY = %q, want %q", tt.beforeCursor+"<TAB>"+tt.afterCursor, got, tt.want)
				}
			})
		}
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
