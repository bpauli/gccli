package cmd

import (
	"flag"
	"fmt"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
)

var update = flag.Bool("update", false, "rewrite the outputs in testdata/golden, the group help in testdata/help.golden and the command reference in the docs")

const goldenDir = "testdata/golden"

type goldenMode struct {
	name  string
	flags []string
}

var goldenModes = []goldenMode{
	{"table", nil},
	{"plain", []string{"--plain"}},
	{"plain-header", []string{"--plain", "--header"}},
	{"json", []string{"--json"}},
}

var goldenOutputs = []string{"stdout", "stderr", "exit", "requests"}

func TestGolden(t *testing.T) {
	leaves := leafCommands(t)
	defaults := map[string]bool{}
	defaultChildren(newTestParser(t).Model.Node, nil, defaults)
	files, err := filepath.Glob(filepath.Join(goldenDir, "*.golden"))
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string][]string{}
	for _, file := range files {
		path, _, isCase := goldenCommand(file)
		switch {
		case !slices.Contains(leaves, path):
			t.Errorf("%s names no leaf command: remove the file", file)
		case isCase:
			cases[path] = append(cases[path], file)
		}
	}

	for _, path := range leaves {
		file := goldenFile(path)
		if _, err := os.Stat(file); err != nil {
			t.Errorf("%q has no golden file: create %s with an args section and the routes it calls, then run go test ./internal/cmd -run TestGolden -update", path, file)
			continue
		}
		for _, f := range append([]string{file}, cases[path]...) {
			runGolden(t, path, f, defaults[path])
		}
	}
}

func runGolden(t *testing.T, path, file string, isDefault bool) {
	t.Helper()
	g, err := readGolden(file)
	if err != nil {
		t.Error(err)
		return
	}
	args, in, err := g.input()
	if err != nil {
		t.Errorf("%s: %v", file, err)
		return
	}
	_, caseName, _ := goldenCommand(file)
	if err := goldenArgs(path, caseName, isDefault, args); err != nil {
		t.Errorf("%s: %v", file, err)
		return
	}

	label := strings.ReplaceAll(strings.TrimSuffix(filepath.Base(file), ".golden"), "_", " ")
	passed := map[string]string{}
	requested := map[string]bool{}
	ran := 0
	for _, m := range goldenModes {
		t.Run(label+"/"+m.name, func(t *testing.T) {
			ran++
			h := newHarness(t, in)
			stdout, stderr, code := h.run(m.flags, args)
			var log strings.Builder
			for _, r := range h.requests() {
				requested[r.route] = true
				log.WriteString(r.entry)
			}
			out := map[string]string{
				"stdout":   h.mask(stdout),
				"stderr":   h.mask(stderr),
				"exit":     strconv.Itoa(code) + "\n",
				"requests": h.mask(log.String()),
			}
			for _, part := range goldenOutputs {
				name := m.name + " " + part
				if *update {
					passed[name] = out[part]
					continue
				}
				want, ok := g.sections[name]
				if !ok {
					t.Errorf("%s has no %q section: run go test ./internal/cmd -run TestGolden -update to write it", file, name)
					continue
				}
				if out[part] != want {
					t.Errorf("%q in %s mode: %s differs from %s (- want, + got):\n%s", path, m.name, part, file, lineDiff(want, out[part]))
				}
			}
			if t.Failed() {
				for _, part := range goldenOutputs {
					delete(passed, m.name+" "+part)
				}
			}
		})
	}
	for _, route := range slices.Sorted(maps.Keys(in.routes)) {
		if ran == len(goldenModes) && !requested[route] {
			t.Errorf("%s: no mode requests route %q: remove the section", file, route)
		}
	}
	if *update {
		for name, body := range passed {
			if sectionLine.MatchString(body) {
				t.Fatalf("%s: %s has a line that looks like a section marker", file, name)
			}
		}
		if err := os.WriteFile(file, g.withOutputs(passed).format(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGoldenArgs(t *testing.T) {
	tests := []struct {
		name      string
		caseName  string
		isDefault bool
		args      string
		ok        bool
	}{
		{"main file names the leaf", "", true, "health steps view 2026-10-15", true},
		{"main file omits the default child", "", true, "health steps 2026-10-15", false},
		{"other case omits the default child", "date", true, "health steps 2026-10-15", false},
		{"bare case omits the default child", "bare", true, "health steps 2026-10-15", true},
		{"bare case with no more args", "bare", true, "health steps", true},
		{"bare-bad-date case omits the default child", "bare-bad-date", true, "health steps 2026-02-30", true},
		{"bare case names the default child", "bare", true, "health steps view 2026-10-15", false},
		{"bare case of another group", "bare", true, "health hr 2026-10-15", false},
		{"bare case of a leaf that is not default", "bare", false, "health steps 2026-10-15", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := goldenArgs("health steps view", tt.caseName, tt.isDefault, strings.Fields(tt.args))
			if (err == nil) != tt.ok {
				t.Errorf("goldenArgs(%q) = %v, want ok %v", tt.args, err, tt.ok)
			}
		})
	}
}

func TestDefaultChildren(t *testing.T) {
	defaults := map[string]bool{}
	defaultChildren(newTestParser(t).Model.Node, nil, defaults)
	for path, want := range map[string]bool{"health steps view": true, "health steps daily": false, "health steps": false} {
		if defaults[path] != want {
			t.Errorf("defaults[%q] = %v, want %v", path, defaults[path], want)
		}
	}
}

func leafCommands(t *testing.T) []string {
	t.Helper()
	cmds := collectCommands(newTestParser(t).Model.Node, nil)
	var leaves []string
	for i, c := range cmds {
		if !hasChildren(cmds, i) {
			leaves = append(leaves, strings.Join(c.Path, " "))
		}
	}
	return leaves
}

// hasChildren reports whether cmds[i] is a group. collectCommands lists the
// children of a command right after it.
func hasChildren(cmds []commandInfo, i int) bool {
	return i+1 < len(cmds) && len(cmds[i+1].Path) > len(cmds[i].Path)
}

func goldenFile(path string) string {
	return filepath.Join(goldenDir, strings.ReplaceAll(path, " ", "_")+".golden")
}

func goldenCommand(file string) (path, caseName string, isCase bool) {
	leaf, caseName, isCase := strings.Cut(strings.TrimSuffix(filepath.Base(file), ".golden"), "@")
	return strings.ReplaceAll(leaf, "_", " "), caseName, isCase
}

func defaultChildren(node *kong.Node, path []string, into map[string]bool) {
	for _, child := range node.Children {
		if child.Type != kong.CommandNode {
			continue
		}
		childPath := append(slices.Clone(path), child.Name)
		if child == node.DefaultCmd {
			into[strings.Join(childPath, " ")] = true
		}
		defaultChildren(child, childPath, into)
	}
}

// goldenArgs checks the args of a golden file of leaf path. A case named
// bare* pins the call of a Default child without its name, so its args start
// with the group path and do not name the child next.
func goldenArgs(path, caseName string, isDefault bool, args []string) error {
	words := strings.Fields(path)
	if !strings.HasPrefix(caseName, "bare") {
		if !hasWords(args, words) {
			return fmt.Errorf("args must start with %q", path)
		}
		return nil
	}
	group, child := words[:len(words)-1], words[len(words)-1]
	switch {
	case !isDefault:
		return fmt.Errorf("case %q pins a bare group call, but %q is not the default command of its group: rename the case or remove the file", caseName, path)
	case !hasWords(args, group) || hasWords(args, words):
		return fmt.Errorf("case %q pins a bare group call: args must start with %q and must not name %q next", caseName, strings.Join(group, " "), child)
	}
	return nil
}

func hasWords(args, words []string) bool {
	return slices.Equal(args[:min(len(args), len(words))], words)
}

func (h *harness) mask(s string) string {
	return strings.NewReplacer(h.dir, "$DIR", h.home, "$HOME").Replace(s)
}

type golden struct {
	comment  string
	names    []string
	sections map[string]string
}

var sectionLine = regexp.MustCompile(`(?m)^-- (.+) --$`)

const noEOL = " noeol"

func parseGolden(data string) (golden, error) {
	g := golden{sections: map[string]string{}}
	marks := sectionLine.FindAllStringSubmatchIndex(data, -1)
	if len(marks) == 0 {
		return g, fmt.Errorf("no sections")
	}
	g.comment = data[:marks[0][0]]
	for i, m := range marks {
		name := data[m[2]:m[3]]
		end := len(data)
		if i+1 < len(marks) {
			end = marks[i+1][0]
		}
		body := strings.TrimPrefix(data[m[1]:end], "\n")
		if trimmed, ok := strings.CutSuffix(name, noEOL); ok {
			name, body = trimmed, strings.TrimSuffix(body, "\n")
		}
		if _, dup := g.sections[name]; dup {
			return g, fmt.Errorf("section %q appears twice", name)
		}
		g.names = append(g.names, name)
		g.sections[name] = body
	}
	return g, nil
}

func (g golden) format() []byte {
	var b strings.Builder
	b.WriteString(g.comment)
	for _, name := range g.names {
		body := g.sections[name]
		if body != "" && !strings.HasSuffix(body, "\n") {
			name, body = name+noEOL, body+"\n"
		}
		fmt.Fprintf(&b, "-- %s --\n%s", name, body)
	}
	return []byte(b.String())
}

func isOutput(name string) bool {
	mode, part, _ := strings.Cut(name, " ")
	return slices.Contains(goldenOutputs, part) && slices.ContainsFunc(goldenModes, func(m goldenMode) bool { return m.name == mode })
}

func (g golden) withOutputs(passed map[string]string) golden {
	out := golden{comment: g.comment, sections: maps.Clone(g.sections)}
	for _, name := range g.names {
		if !isOutput(name) {
			out.names = append(out.names, name)
		}
	}
	for _, m := range goldenModes {
		for _, part := range goldenOutputs {
			name := m.name + " " + part
			if body, ok := passed[name]; ok {
				out.sections[name] = body
			}
			if _, ok := out.sections[name]; ok {
				out.names = append(out.names, name)
			}
		}
	}
	return out
}

func (g golden) input() ([]string, harnessInput, error) {
	in := harnessInput{files: map[string]string{}, routes: map[string]mockReply{}}
	var args []string
	for _, name := range g.names {
		body := g.sections[name]
		kind, rest, _ := strings.Cut(name, " ")
		switch {
		case name == "args":
			args = strings.Split(strings.TrimSuffix(body, "\n"), "\n")
		case name == "stdin":
			in.stdin = body
		case kind == "file" && rest != "":
			in.files[rest] = body
		case kind == "route":
			f := strings.Fields(rest)
			if len(f) < 2 || len(f) > 3 {
				return nil, in, fmt.Errorf("section %q: want route <METHOD> <uri> [status]", name)
			}
			reply := mockReply{body: body}
			if len(f) == 3 {
				status, err := strconv.Atoi(f[2])
				if err != nil {
					return nil, in, fmt.Errorf("section %q: %v", name, err)
				}
				reply.status = status
			}
			in.routes[f[0]+" "+f[1]] = reply
		case isOutput(name):
		default:
			return nil, in, fmt.Errorf("unknown section %q: want args, stdin, file <name>, route <METHOD> <uri> [status], or <mode> stdout|stderr|exit|requests", name)
		}
	}
	if len(args) == 0 {
		return nil, in, fmt.Errorf("no args section")
	}
	return args, in, nil
}

func lineDiff(want, got string) string {
	a, b := strings.SplitAfter(want, "\n"), strings.SplitAfter(got, "\n")
	lcs := make([][]int, len(a)+1)
	for i := range lcs {
		lcs[i] = make([]int, len(b)+1)
	}
	for i := len(a) - 1; i >= 0; i-- {
		for j := len(b) - 1; j >= 0; j-- {
			if a[i] == b[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	var out strings.Builder
	for i, j := 0, 0; i < len(a) || j < len(b); {
		switch {
		case i < len(a) && j < len(b) && a[i] == b[j]:
			i, j = i+1, j+1
		case i < len(a) && (j == len(b) || lcs[i+1][j] >= lcs[i][j+1]):
			fmt.Fprintf(&out, "-%4d %q\n", i+1, a[i])
			i++
		default:
			fmt.Fprintf(&out, "+%4d %q\n", j+1, b[j])
			j++
		}
	}
	return out.String()
}

var errorCaseExempt = map[string]string{
	"auth export": "reads the keyring and sends no request",
	"auth import": "writes the keyring and sends no request",
	"auth login":  "the harness stubs the SSO login, so it sends no request",
	"auth remove": "writes the keyring and sends no request",
	"auth status": "reads the keyring and sends no request",
	"auth token":  "reads the keyring; with an expired token it sends only a refresh, whose error has no verb",
	"completion":  "prints a script and sends no request",
}

func TestGoldenErrorCases(t *testing.T) {
	leaves := leafCommands(t)
	for _, path := range slices.Sorted(maps.Keys(errorCaseExempt)) {
		if !slices.Contains(leaves, path) {
			t.Errorf("errorCaseExempt names %q, which is not a leaf command: remove the entry", path)
		}
	}
	for _, path := range leaves {
		main := goldenFile(path)
		cases, _ := filepath.Glob(strings.TrimSuffix(main, ".golden") + "@*.golden")
		var goldens []golden
		for _, file := range append([]string{main}, cases...) {
			g, err := readGolden(file)
			if err != nil {
				t.Error(err)
				continue
			}
			goldens = append(goldens, g)
		}
		var data string
		for _, g := range goldens {
			if data = dataRoute(g); data != "" {
				break
			}
		}
		why, exempt := errorCaseExempt[path]
		switch {
		case data == "" && !exempt:
			t.Errorf("no golden file of %q sends a request: add it to errorCaseExempt with the reason", path)
		case data == "":
		case exempt:
			t.Errorf("errorCaseExempt names %q (%s), but a golden file of it sends %s: remove the entry", path, why, data)
		case !slices.ContainsFunc(goldens, func(g golden) bool { return endsInError(g, data) }):
			t.Errorf("no golden file of %q ends in an error from %s, so nothing pins the verb of its error message: copy the input sections of %s to %s@error.golden, give that route status 400 and a short JSON body, drop the routes that the command no longer requests, then run go test ./internal/cmd -run TestGolden -update", path, data, main, strings.TrimSuffix(main, ".golden"))
		}
	}
}

func readGolden(file string) (golden, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return golden{}, err
	}
	g, err := parseGolden(string(data))
	if err != nil {
		return golden{}, fmt.Errorf("%s: %v", file, err)
	}
	return g, nil
}

func requestRoutes(section string) []string {
	var routes []string
	for _, line := range strings.Split(strings.TrimSuffix(section, "\n"), "\n") {
		if line != "" && line[0] != ' ' && line[0] != '\t' {
			routes = append(routes, line)
		}
	}
	return routes
}

func isLookup(route string) bool {
	return route == profileRoute || route == socialRoute || route == settingsRoute
}

func dataRoute(g golden) string {
	routes := requestRoutes(g.sections["json requests"])
	for _, r := range routes {
		if !isLookup(r) {
			return r
		}
	}
	if len(routes) == 0 {
		return ""
	}
	return routes[len(routes)-1]
}

func endsInError(g golden, data string) bool {
	routes := requestRoutes(g.sections["json requests"])
	if len(routes) == 0 || routes[len(routes)-1] != data {
		return false
	}
	_, in, err := g.input()
	if err != nil {
		return false
	}
	return verbInMessage(in.routes[data].status)
}

func verbInMessage(status int) bool {
	return status >= 400 && status != http.StatusUnauthorized && status != http.StatusTooManyRequests
}
