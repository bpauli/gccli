package conformance

import (
	"cmp"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

var failWithoutBinary bool

var (
	ca     *authority
	caFile string
)

func TestMain(m *testing.M) {
	os.Exit(func() int {
		var err error
		if ca, err = newAuthority(); err != nil {
			fmt.Fprintln(os.Stderr, "conformance:", err)
			return 1
		}
		dir, err := os.MkdirTemp("", "gccli-conformance-ca-")
		if err != nil {
			fmt.Fprintln(os.Stderr, "conformance:", err)
			return 1
		}
		defer func() { _ = os.RemoveAll(dir) }()
		caFile = filepath.Join(dir, "ca.pem")
		if err := os.WriteFile(caFile, ca.PEM, 0o600); err != nil {
			fmt.Fprintln(os.Stderr, "conformance:", err)
			return 1
		}
		return m.Run()
	}())
}

func binary(t *testing.T) string {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skipf("the conformance suite runs only on linux: the binary must trust the mock CA through SSL_CERT_FILE, which is not reliable on %s", runtime.GOOS)
	}
	bin := os.Getenv("GCCLI_BIN")
	switch {
	case bin == "" && failWithoutBinary:
		t.Fatal("GCCLI_BIN is not set: name the gccli binary to check, as in make build && make conformance GCCLI_BIN=bin/gccli")
	case bin == "":
		t.Skip("GCCLI_BIN is not set: to run the conformance suite, run make build && make conformance GCCLI_BIN=bin/gccli")
	case !filepath.IsAbs(bin):
		t.Fatalf("GCCLI_BIN=%q is not an absolute path", bin)
	}
	info, err := os.Stat(bin)
	if err != nil {
		t.Fatalf("GCCLI_BIN: %v", err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("GCCLI_BIN=%q is not an executable file", bin)
	}
	return bin
}

type sandbox struct {
	bin   string
	home  string
	work  string
	proxy *proxy
}

func newSandbox(t *testing.T, bin string, handler http.Handler) *sandbox {
	t.Helper()
	p, err := ca.serve(handler)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.close)
	return &sandbox{bin: bin, home: t.TempDir(), work: t.TempDir(), proxy: p}
}

func (s *sandbox) env() []string {
	return []string{
		"HOME=" + s.home,
		"XDG_CONFIG_HOME=" + s.home,
		"GCCLI_KEYRING_BACKEND=file",
		"HTTPS_PROXY=" + s.proxy.URL,
		"HTTP_PROXY=" + s.proxy.URL,
		"SSL_CERT_FILE=" + caFile,
		"TZ=UTC",
	}
}

type result struct {
	stdout, stderr string
	code           int
}

func (s *sandbox) start(t *testing.T, typed string, args ...string) (wait func() result) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)
	cmd := exec.CommandContext(ctx, s.bin, args...)
	cmd.Dir = s.work
	cmd.Env = s.env()
	var stdout, stderr strings.Builder
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if typed != "" {
		ptmx, pts, err := openTerminal()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = ptmx.Close() })
		defer func() { _ = pts.Close() }()
		cmd.Stdin = pts
		go func() { _, _ = io.Copy(io.Discard, ptmx) }()
		if _, err := ptmx.WriteString(typed); err != nil {
			t.Fatal(err)
		}
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	return func() result {
		err := cmd.Wait()
		if ctx.Err() != nil {
			t.Fatalf("gccli %s: no exit after %s", strings.Join(args, " "), time.Minute)
		}
		var exit *exec.ExitError
		if err != nil && !errors.As(err, &exit) {
			t.Fatalf("gccli %s: %v", strings.Join(args, " "), err)
		}
		return result{stdout.String(), stderr.String(), cmd.ProcessState.ExitCode()}
	}
}

func (s *sandbox) run(t *testing.T, typed string, args ...string) result {
	t.Helper()
	return s.start(t, typed, args...)()
}

func (s *sandbox) importToken(t *testing.T, tokenJSON []byte) {
	t.Helper()
	res := s.run(t, "", "auth", "import", base64.StdEncoding.EncodeToString(tokenJSON))
	if res.code != 0 {
		t.Fatalf("auth import: exit %d\n%s", res.code, res.stderr)
	}
}

const goldenDir = "../internal/cmd/testdata/golden"

var (
	goldenGlobals = []string{"--account", "test@example.com", "--units", "metric"}
	goldenModes   = []struct {
		name  string
		flags []string
	}{
		{"table", nil},
		{"plain", []string{"--plain"}},
		{"json", []string{"--json"}},
	}
)

var skipped = map[string]string{
	"auth login": "the harness stubs the browser SSO login",
}

var dateInURI = regexp.MustCompile(`\d{4}-\d{2}-\d{2}`)

func (c goldenCase) clockDate() (string, bool) {
	args := strings.Join(c.args, " ")
	for route := range c.routes {
		for _, d := range dateInURI.FindAllString(route, -1) {
			if !strings.Contains(args, d) {
				return d, true
			}
		}
	}
	return "", false
}

func TestGolden(t *testing.T) {
	bin := binary(t)
	token, err := os.ReadFile("testdata/state/token_golden.json")
	if err != nil {
		t.Fatal(err)
	}
	seed := newSandbox(t, bin, http.NotFoundHandler())
	seed.importToken(t, token)

	files, err := filepath.Glob(filepath.Join(goldenDir, "*.golden"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatalf("no golden files in %s", goldenDir)
	}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		c, err := parseGolden(string(data))
		if err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		name := strings.ReplaceAll(strings.TrimSuffix(filepath.Base(file), ".golden"), "_", " ")
		for _, m := range goldenModes {
			t.Run(name+"/"+m.name, func(t *testing.T) {
				if why, ok := skipped[name]; ok {
					t.Skip(why)
				}
				if d, ok := c.clockDate(); ok {
					t.Skipf("a route asks for %s, which the binary takes from its clock", d)
				}
				t.Parallel()
				r := &replay{routes: c.routes}
				s := newSandbox(t, bin, r)
				if err := os.CopyFS(s.home, os.DirFS(seed.home)); err != nil {
					t.Fatal(err)
				}
				for f, content := range c.files {
					if err := os.WriteFile(filepath.Join(s.work, f), []byte(content), 0o600); err != nil {
						t.Fatal(err)
					}
				}
				res := s.run(t, c.stdin, slices.Concat(goldenGlobals, m.flags, c.args)...)
				mask := strings.NewReplacer(s.work, "$DIR", s.home, "$HOME")
				got := map[string]string{
					"stdout": mask.Replace(res.stdout),
					"stderr": mask.Replace(res.stderr),
					"exit":   strconv.Itoa(res.code) + "\n",
				}
				for _, part := range []string{"stdout", "stderr", "exit"} {
					want, ok := c.outputs[m.name+" "+part]
					if !ok {
						t.Fatalf("%s has no %q section", file, m.name+" "+part)
					}
					if got[part] != want {
						t.Errorf("%q in %s mode: %s differs from %s (- want, + got):\n%s", name, m.name, part, file, lineDiff(want, got[part]))
					}
				}
				for _, u := range r.unexpectedRequests() {
					t.Errorf("%q in %s mode sent %s, which %s has no route for", name, m.name, u, file)
				}
				for _, req := range s.proxy.requests() {
					if !strings.HasSuffix(req.Host, ".garmin.com") {
						t.Errorf("%q in %s mode sent %s to a host outside garmin.com", name, m.name, req)
					}
				}
			})
		}
	}
}

type replay struct {
	routes map[string]reply

	mu         sync.Mutex
	unexpected []string
}

type reply struct {
	status int
	body   string
}

func (r *replay) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	key := req.Method + " " + req.URL.RequestURI()
	rep, ok := r.routes[key]
	if !ok {
		r.mu.Lock()
		r.unexpected = append(r.unexpected, key)
		r.mu.Unlock()
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if rep.body != "" {
		w.Header().Set("Content-Type", "application/json")
	}
	w.WriteHeader(cmp.Or(rep.status, http.StatusOK))
	_, _ = w.Write([]byte(rep.body))
}

func (r *replay) unexpectedRequests() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.unexpected)
}

type goldenCase struct {
	args    []string
	stdin   string
	files   map[string]string
	routes  map[string]reply
	outputs map[string]string
}

var sectionLine = regexp.MustCompile(`(?m)^-- (.+) --$`)

func parseGolden(data string) (goldenCase, error) {
	c := goldenCase{files: map[string]string{}, routes: map[string]reply{}, outputs: map[string]string{}}
	marks := sectionLine.FindAllStringSubmatchIndex(data, -1)
	for i, m := range marks {
		name := data[m[2]:m[3]]
		end := len(data)
		if i+1 < len(marks) {
			end = marks[i+1][0]
		}
		body := strings.TrimPrefix(data[m[1]:end], "\n")
		if trimmed, ok := strings.CutSuffix(name, " noeol"); ok {
			name, body = trimmed, strings.TrimSuffix(body, "\n")
		}
		kind, rest, _ := strings.Cut(name, " ")
		switch {
		case name == "args":
			c.args = strings.Split(strings.TrimSuffix(body, "\n"), "\n")
		case name == "stdin":
			c.stdin = body
		case kind == "file":
			c.files[rest] = body
		case kind == "route":
			f := strings.Fields(rest)
			rep := reply{body: body}
			if len(f) == 3 {
				status, err := strconv.Atoi(f[2])
				if err != nil {
					return c, fmt.Errorf("section %q: %v", name, err)
				}
				rep.status = status
			} else if len(f) != 2 {
				return c, fmt.Errorf("section %q: want route <METHOD> <uri> [status]", name)
			}
			c.routes[f[0]+" "+f[1]] = rep
		default:
			c.outputs[name] = body
		}
	}
	if len(c.args) == 0 {
		return c, errors.New("no args section")
	}
	return c, nil
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
	i, j := 0, 0
	for i < len(a) || j < len(b) {
		switch {
		case i < len(a) && j < len(b) && a[i] == b[j]:
			fmt.Fprintf(&out, "  %q\n", a[i])
			i, j = i+1, j+1
		case j < len(b) && (i == len(a) || lcs[i][j+1] >= lcs[i+1][j]):
			fmt.Fprintf(&out, "+ %q\n", b[j])
			j++
		default:
			fmt.Fprintf(&out, "- %q\n", a[i])
			i++
		}
	}
	return out.String()
}
