package cmd

import (
	"flag"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the outputs in testdata/golden and the command reference in the docs")

const goldenDir = "testdata/golden"

type goldenMode struct {
	name  string
	flags []string
}

var goldenModes = []goldenMode{
	{"table", nil},
	{"plain", []string{"--plain"}},
	{"json", []string{"--json"}},
}

var goldenOutputs = []string{"stdout", "stderr", "exit"}

func TestGolden(t *testing.T) {
	leaves := leafCommands(t)
	files, err := filepath.Glob(filepath.Join(goldenDir, "*.golden"))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if !slices.Contains(leaves, goldenCommand(file)) {
			t.Errorf("%s names no leaf command: remove the file", file)
		}
	}

	for _, path := range leaves {
		file := goldenFile(path)
		data, err := os.ReadFile(file)
		if err != nil {
			t.Errorf("%q has no golden file: create %s with an args section and the routes it calls, then run go test ./internal/cmd -run TestGolden -update", path, file)
			continue
		}
		g, err := parseGolden(string(data))
		if err != nil {
			t.Errorf("%s: %v", file, err)
			continue
		}
		args, in, err := g.input()
		if err != nil {
			t.Errorf("%s: %v", file, err)
			continue
		}
		if !slices.Equal(args[:min(len(args), len(strings.Fields(path)))], strings.Fields(path)) {
			t.Errorf("%s: args must start with %q", file, path)
			continue
		}

		passed := map[string]string{}
		requested := map[string]bool{}
		for _, m := range goldenModes {
			t.Run(path+"/"+m.name, func(t *testing.T) {
				h := newHarness(t, in)
				stdout, stderr, code := h.run(m.flags, args)
				h.mu.Lock()
				maps.Copy(requested, h.requested)
				h.mu.Unlock()
				out := map[string]string{
					"stdout": h.mask(stdout),
					"stderr": h.mask(stderr),
					"exit":   strconv.Itoa(code) + "\n",
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
			if !requested[route] {
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
}

func leafCommands(t *testing.T) []string {
	t.Helper()
	cmds := collectCommands(newTestParser(t).Model.Node, nil)
	var leaves []string
	for i, c := range cmds {
		if i+1 < len(cmds) && len(cmds[i+1].Path) > len(c.Path) {
			continue
		}
		leaves = append(leaves, strings.Join(c.Path, " "))
	}
	return leaves
}

func goldenFile(path string) string {
	return filepath.Join(goldenDir, strings.ReplaceAll(path, " ", "_")+".golden")
}

func goldenCommand(file string) string {
	return strings.ReplaceAll(strings.TrimSuffix(filepath.Base(file), ".golden"), "_", " ")
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
			return nil, in, fmt.Errorf("unknown section %q: want args, stdin, file <name>, route <METHOD> <uri> [status], or <mode> stdout|stderr|exit", name)
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
