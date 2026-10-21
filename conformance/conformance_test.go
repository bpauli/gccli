package conformance

import (
	"cmp"
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"flag"
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

	"github.com/bpauli/gccli/internal/testutil/requestlog"
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
		"SSL_CERT_DIR=" + filepath.Dir(caFile),
		"TZ=UTC",
	}
}

func TestSandboxDistrustsSystemRoots(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the sandbox env selects the trusted roots only on linux")
	}
	roots := systemRoots()
	if len(roots) == 0 {
		t.Skip("no system roots found")
	}
	if flag.Arg(0) == "trusted-roots-probe" {
		pool, err := x509.SystemCertPool()
		if err != nil {
			t.Fatal(err)
		}
		for _, root := range roots {
			if _, err := root.Verify(x509.VerifyOptions{Roots: pool}); err == nil {
				t.Fatalf("the system root %q is trusted", root.Subject)
			}
		}
		return
	}
	s := newSandbox(t, "", http.NotFoundHandler())
	cmd := exec.Command(os.Args[0], "-test.run=^TestSandboxDistrustsSystemRoots$", "trusted-roots-probe")
	cmd.Env = s.env()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("a process with the sandbox env trusts the system roots, so a binary that ignores HTTPS_PROXY can reach the real Garmin: %v\n%s", err, out)
	}
}

func systemRoots() []*x509.Certificate {
	var roots []*x509.Certificate
	for _, dir := range []string{"/etc/ssl/certs", "/etc/pki/tls/certs"} {
		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			data, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				continue
			}
			for block, rest := pem.Decode(data); block != nil; block, rest = pem.Decode(rest) {
				if c, err := x509.ParseCertificate(block.Bytes); err == nil {
					roots = append(roots, c)
				}
			}
		}
	}
	return roots
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

type goldenMode struct {
	name  string
	flags []string
}

var (
	goldenGlobals = []string{"--account", "test@example.com", "--units", "metric"}
	goldenModes   = []goldenMode{
		{"table", nil},
		{"plain", []string{"--plain"}},
		{"plain-header", []string{"--plain", "--header"}},
		{"json", []string{"--json"}},
	}
)

const clockSkip = "a route asks for a date that the binary takes from its clock"

const napClockSkip = "the nap is for today and must end by now, which the binary takes from its clock"

const menstrualLogClockSkip = "the request body holds the time of the write, which the binary takes from its clock"

const foodLogClockSkip = "the request body holds the time of the write, and the message finds the new entry by it, which the binary takes from its clock"

var skipped = map[string]string{
	"auth login":                      "the harness stubs the browser SSO login",
	"nutrition add":                   foodLogClockSkip,
	"nutrition quick-add":             foodLogClockSkip,
	"health sleep naps add":           napClockSkip,
	"wellness menstrual-log set":      menstrualLogClockSkip,
	"body glucose":                    clockSkip,
	"body latest":                     clockSkip,
	"body weekly":                     clockSkip,
	"devices solar":                   clockSkip,
	"events list":                     clockSkip,
	"goals weight":                    clockSkip,
	"health calories weekly":          clockSkip,
	"health critical-swim-speed view": clockSkip,
	"health floors weekly":            clockSkip,
	"health hr weekly":                clockSkip,
	"health one-rep-max":              clockSkip,
	"health power-to-weight":          clockSkip,
	"health pushes monthly":           clockSkip,
	"health pushes weekly":            clockSkip,
	"health running-economy":          clockSkip,
	"health sleep naps view":          clockSkip,
	"health sleep weekly":             clockSkip,
	"health steps monthly":            clockSkip,
	"health steps weekly":             clockSkip,
	"health stress weekly":            clockSkip,
	"health training-status daily":    clockSkip,
	"hydration weekly":                clockSkip,
	"training active":                 clockSkip,
	"wellness jet-lag status":         clockSkip,
	"wellness menstrual-cycle":        clockSkip,
	"wellness menstrual-log view":     clockSkip,
	"wellness menstrual-summary":      clockSkip,
	"workouts schedule export":        clockSkip,
}

// clockBodies names the leaves that send a time from the clock in a request
// body. The suite compares their output, but not their requests.
var clockBodies = map[string]bool{
	"body add-blood-pressure": true,
	"body add-composition":    true,
	"body add-weight":         true,
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

type goldenFile struct {
	path, name, leaf string
	goldenCase
}

func goldenFiles(t *testing.T) []goldenFile {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(goldenDir, "*.golden"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatalf("no golden files in %s", goldenDir)
	}
	var files []goldenFile
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		c, err := parseGolden(string(data))
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		for section := range c.outputs {
			mode, _, _ := strings.Cut(section, " ")
			if !slices.ContainsFunc(goldenModes, func(m goldenMode) bool { return m.name == mode }) {
				t.Fatalf("%s: section %q names a mode that goldenModes does not run", path, section)
			}
		}
		name := strings.ReplaceAll(strings.TrimSuffix(filepath.Base(path), ".golden"), "_", " ")
		leaf, _, _ := strings.Cut(name, "@")
		files = append(files, goldenFile{path, name, leaf, c})
	}
	return files
}

func TestGoldenSkips(t *testing.T) {
	fromClock := map[string]bool{}
	for _, f := range goldenFiles(t) {
		d, clock := f.clockDate()
		fromClock[f.leaf] = fromClock[f.leaf] || clock
		if clock && skipped[f.leaf] != clockSkip {
			t.Errorf("%s: a route asks for %s, which the binary takes from its clock: give the date in the args, or add %q to skipped", f.path, d, f.leaf)
		}
	}
	for name := range clockBodies {
		if _, ok := fromClock[name]; !ok {
			t.Errorf("clockBodies names %q, which has no golden file in %s", name, goldenDir)
		}
	}
	for name, why := range skipped {
		clock, ok := fromClock[name]
		switch {
		case !ok:
			t.Errorf("skipped names %q, which has no golden file in %s", name, goldenDir)
		case why == clockSkip && !clock:
			t.Errorf("skipped names %q as a clock case, but no route of its golden files asks for a date from the clock: remove it from skipped", name)
		}
	}
}

func TestGolden(t *testing.T) {
	bin := binary(t)
	token, err := os.ReadFile("testdata/state/token_golden.json")
	if err != nil {
		t.Fatal(err)
	}
	seed := newSandbox(t, bin, http.NotFoundHandler())
	seed.importToken(t, token)

	for _, f := range goldenFiles(t) {
		file, name, c := f.path, f.name, f.goldenCase
		for _, m := range goldenModes {
			t.Run(name+"/"+m.name, func(t *testing.T) {
				if why, ok := skipped[f.leaf]; ok {
					t.Skip(why)
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
					"stdout":   mask.Replace(res.stdout),
					"stderr":   mask.Replace(res.stderr),
					"exit":     strconv.Itoa(res.code) + "\n",
					"requests": mask.Replace(r.log()),
				}
				for _, part := range []string{"stdout", "stderr", "exit", "requests"} {
					want, ok := c.outputs[m.name+" "+part]
					if !ok {
						t.Fatalf("%s has no %q section", file, m.name+" "+part)
					}
					switch {
					case part == "requests" && clockBodies[f.leaf]:
						if got[part] == want {
							t.Errorf("%q in %s mode: the requests match %s, so no body takes a time from the clock: remove %q from clockBodies", name, m.name, file, f.leaf)
						}
					case got[part] != want:
						t.Errorf("%q in %s mode: %s differs from %s (- want, + got):\n%s", name, m.name, part, file, lineDiff(want, got[part]))
					}
				}
				for _, u := range r.unexpectedRequests() {
					t.Errorf("%q in %s mode sent %s, which %s has no route for", name, m.name, u, file)
				}
			})
		}
	}
}

type replay struct {
	routes map[string]reply

	mu         sync.Mutex
	unexpected []string
	entries    []string
}

type reply struct {
	status int
	body   string
}

func (r *replay) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	key := req.Method + " " + req.Host + req.URL.RequestURI()
	entry, err := requestlog.Format(req)
	if err != nil {
		entry = fmt.Sprintf("%s: %v\n", key, err)
	}
	rep, ok := r.routes[key]
	r.mu.Lock()
	r.entries = append(r.entries, entry)
	if !ok {
		r.unexpected = append(r.unexpected, key)
	}
	r.mu.Unlock()
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if rep.body != "" {
		w.Header().Set("Content-Type", "application/json")
	}
	w.WriteHeader(cmp.Or(rep.status, http.StatusOK))
	_, _ = w.Write([]byte(rep.body))
}

func (r *replay) log() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return strings.Join(r.entries, "")
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
			c.routes[f[0]+" "+routeHost(f[1])+f[1]] = rep
		default:
			c.outputs[name] = body
		}
	}
	if len(c.args) == 0 {
		return c, errors.New("no args section")
	}
	return c, nil
}

func routeHost(uri string) string {
	if strings.HasPrefix(uri, "/web-data/") {
		return "connect.garmin.com"
	}
	return "connectapi.garmin.com"
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
