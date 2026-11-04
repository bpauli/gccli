package cmd

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/alecthomas/kong"

	"github.com/bpauli/gccli/internal/garminapi"
	"github.com/bpauli/gccli/internal/units"
)

func TestExecute_Version(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := execute([]string{"--version"}, "1.0.0", "abc123", "2024-01-01", &stdout, &stderr)
	if code != 0 || stdout.String() != "gccli 1.0.0 (abc123) built 2024-01-01\n" || stderr.Len() != 0 {
		t.Errorf("exit %d, stdout %q, stderr %q", code, stdout.String(), stderr.String())
	}
}

func TestExecute_Auth(t *testing.T) {
	code := Execute([]string{"auth"}, "1.0.0", "abc123", "2024-01-01")
	if code == 0 {
		t.Fatal("expected non-zero exit code for auth without subcommand")
	}
}

func TestExecute_UnknownCommand(t *testing.T) {
	code := Execute([]string{"nonexistent"}, "1.0.0", "abc123", "2024-01-01")
	if code == 0 {
		t.Fatal("expected non-zero exit code for unknown command")
	}
}

func TestExecute_NoArgs(t *testing.T) {
	code := Execute([]string{}, "1.0.0", "abc123", "2024-01-01")
	if code == 0 {
		t.Fatal("expected non-zero exit code when no command is given")
	}
}

func TestExitCode_Nil(t *testing.T) {
	if got := exitCode(nil); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}

func captureOutput(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()
	capture := func(target **os.File) func() string {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		orig := *target
		*target = w
		t.Cleanup(func() {
			_ = w.Close()
			*target = orig
		})
		done := make(chan string)
		go func() {
			b, _ := io.ReadAll(r)
			done <- string(b)
		}()
		return func() string {
			_ = w.Close()
			*target = orig
			return <-done
		}
	}
	finishOut := capture(&os.Stdout)
	finishErr := capture(&os.Stderr)
	fn()
	return finishOut(), finishErr()
}

func TestExecute_RateLimitShowsHint(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var code int
	_, stderr := captureOutput(t, func() {
		code = Execute([]string{"--color", "never", "--account", "test@example.com", "badges", "earned"}, "1.0.0", "abc123", "2024-01-01")
	})
	if code == 0 {
		t.Fatal("exit code = 0, want non-zero")
	}
	if want := "Rate limited. Wait 30 and retry."; !strings.Contains(stderr, want) {
		t.Fatalf("stderr = %q, want it to contain %q", stderr, want)
	}
}

func TestExecute_ExitCodes(t *testing.T) {
	tests := []struct {
		name   string
		status int
		args   []string
		want   int
	}{
		{name: "not found", status: http.StatusNotFound, args: []string{"badges", "earned"}, want: 4},
		{name: "rate limited", status: http.StatusTooManyRequests, args: []string{"badges", "earned"}, want: 5},
		{name: "server error", status: http.StatusInternalServerError, args: []string{"badges", "earned"}, want: 1},
		{name: "no stored credentials", args: []string{"--account", "nobody@example.com", "badges", "earned"}, want: 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			t.Setenv("HOME", tmp)
			t.Setenv("XDG_CONFIG_HOME", tmp)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
			}))
			defer server.Close()

			store := newTestSecretsStore(t)
			overrideLoadSecrets(t, store)
			overrideNewClient(t, server)
			storeTestTokens(t, store, "test@example.com", testTokens())

			args := append([]string{"--account", "test@example.com"}, tt.args...)
			var code int
			captureOutput(t, func() {
				code = Execute(args, "1.0.0", "abc123", "2024-01-01")
			})
			if code != tt.want {
				t.Fatalf("exit code = %d, want %d", code, tt.want)
			}
		})
	}
}

func TestExitCode_TokenExpired(t *testing.T) {
	err := fmt.Errorf("get goals: %w", &garminapi.TokenExpiredError{Email: "a@b.com"})
	if got := exitCode(err); got != 3 {
		t.Fatalf("exit code = %d, want 3", got)
	}
}

func TestExecute_UsageErrorGoesToStderrWithExitCode2(t *testing.T) {
	var code int
	stdout, stderr := captureOutput(t, func() {
		code = Execute([]string{"--bogus"}, "1.0.0", "abc123", "2024-01-01")
	})
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "unknown flag --bogus") || !strings.Contains(stderr, "Usage:") {
		t.Fatalf("stderr = %q, want the error and the usage text", stderr)
	}
}

func TestExecute_HeaderWithoutPlainIsUsageError(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		args []string
	}{
		{name: "table", args: []string{"--header"}},
		{name: "json", args: []string{"--json", "--header"}},
		{name: "json beats plain", args: []string{"--json", "--plain", "--header"}},
		{name: "env json", env: map[string]string{"GCCLI_JSON": "1"}, args: []string{"--plain", "--header"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			t.Setenv("HOME", tmp)
			t.Setenv("XDG_CONFIG_HOME", tmp)
			t.Setenv("GCCLI_ACCOUNT", "")
			for _, k := range []string{"GCCLI_JSON", "GCCLI_PLAIN", "GCCLI_HEADER"} {
				t.Setenv(k, cmp.Or(tt.env[k], "0"))
			}

			var code int
			stdout, stderr := captureOutput(t, func() {
				code = Execute(append(tt.args, "courses", "list"), "1.0.0", "abc123", "2024-01-01")
			})
			if code != 2 {
				t.Fatalf("exit code = %d, want 2", code)
			}
			if stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
			if !strings.Contains(stderr, "gccli: error: --header works only with --plain\n") || !strings.Contains(stderr, "Usage:") {
				t.Fatalf("stderr = %q, want the --header error and the usage text", stderr)
			}
		})
	}
}

func TestExecute_HeaderEnvAppliesOnlyToPlain(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		args []string
		want string
	}{
		{name: "table", want: "ID   NAME  TYPE     DISTANCE  CREATED\n111  Loop  running  8.20 km   2026-09-01\n"},
		{name: "json", args: []string{"--json"}, want: "{\n  \"coursesForUser\": [\n    {\n      \"courseId\": 111,"},
		{name: "env json", env: map[string]string{"GCCLI_JSON": "1"}, want: "{\n  \"coursesForUser\": [\n    {\n      \"courseId\": 111,"},
		{name: "plain", args: []string{"--plain"}, want: "ID\tNAME\tTYPE\tDISTANCE\tCREATED\n111\tLoop\trunning\t8.20 km\t2026-09-01\n"},
		{name: "env plain", env: map[string]string{"GCCLI_PLAIN": "1"}, want: "ID\tNAME\tTYPE\tDISTANCE\tCREATED\n111\tLoop\trunning\t8.20 km\t2026-09-01\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			t.Setenv("HOME", tmp)
			t.Setenv("XDG_CONFIG_HOME", tmp)
			t.Setenv("GCCLI_HEADER", "1")
			for _, k := range []string{"GCCLI_JSON", "GCCLI_PLAIN"} {
				t.Setenv(k, cmp.Or(tt.env[k], "0"))
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, `{"coursesForUser":[{"courseId":111,"courseName":"Loop","activityType":{"typeKey":"running"},"distanceInMeters":8200,"createdDate":1788256800000,"createdDateFormatted":"2026-09-01 10:00:00.0 GMT"}]}`)
			}))
			defer server.Close()

			store := newTestSecretsStore(t)
			overrideLoadSecrets(t, store)
			overrideNewClient(t, server)
			storeTestTokens(t, store, "test@example.com", testTokens())

			args := append([]string{"--account", "test@example.com", "--units", "metric"}, tt.args...)
			var code int
			stdout, stderr := captureOutput(t, func() {
				code = Execute(append(args, "courses", "list"), "1.0.0", "abc123", "2024-01-01")
			})
			if code != 0 {
				t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr)
			}
			if !strings.HasPrefix(stdout, tt.want) {
				t.Fatalf("stdout = %q, want it to start with %q", stdout, tt.want)
			}
		})
	}
}

func TestExecute_AuthCommandsNeedLogin(t *testing.T) {
	tests := [][]string{
		{"auth", "status"},
		{"auth", "token"},
		{"auth", "export"},
		{"--account", "nobody@example.com", "auth", "status"},
		{"--account", "nobody@example.com", "auth", "token"},
		{"--account", "nobody@example.com", "auth", "export"},
	}
	for _, args := range tests {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			tmp := t.TempDir()
			t.Setenv("HOME", tmp)
			t.Setenv("XDG_CONFIG_HOME", tmp)
			t.Setenv("GCCLI_ACCOUNT", "")
			overrideLoadSecrets(t, newTestSecretsStore(t))

			var code int
			captureOutput(t, func() {
				code = Execute(args, "1.0.0", "abc123", "2024-01-01")
			})
			if code != 3 {
				t.Fatalf("exit code = %d, want 3", code)
			}
		})
	}
}

func TestExecute_AuthStatusWithoutCredentialsPrintsJSONError(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)
	overrideLoadSecrets(t, newTestSecretsStore(t))

	var code int
	stdout, stderr := captureOutput(t, func() {
		code = Execute([]string{"--json", "--account", "nobody@example.com", "auth", "status"}, "1.0.0", "abc123", "2024-01-01")
	})
	if code != 3 {
		t.Fatalf("exit code = %d, want 3", code)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	want := `{"error":{"code":"auth_required","message":"no credentials stored for nobody@example.com; run: gccli auth login nobody@example.com"}}` + "\n"
	if stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
}

func TestExecute_JSONModePrintsErrorObject(t *testing.T) {
	tests := []struct {
		name   string
		status int
		want   string
	}{
		{
			name:   "rate limited",
			status: http.StatusTooManyRequests,
			want:   `{"error":{"code":"rate_limited","status":429,"message":"Rate limited. Wait 30 and retry."}}`,
		},
		{
			name:   "not found",
			status: http.StatusNotFound,
			want:   `{"error":{"code":"not_found","status":404,"message":"get earned badges: garmin api: 404 Not Found: gone"}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			t.Setenv("HOME", tmp)
			t.Setenv("XDG_CONFIG_HOME", tmp)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Retry-After", "30")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte("gone"))
			}))
			defer server.Close()

			store := newTestSecretsStore(t)
			overrideLoadSecrets(t, store)
			overrideNewClient(t, server)
			storeTestTokens(t, store, "test@example.com", testTokens())

			var code int
			stdout, stderr := captureOutput(t, func() {
				code = Execute([]string{"--json", "--account", "test@example.com", "badges", "earned"}, "1.0.0", "abc123", "2024-01-01")
			})
			if code == 0 {
				t.Fatal("exit code = 0, want non-zero")
			}
			if stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
			if got := strings.TrimSpace(stderr); got != tt.want {
				t.Fatalf("stderr = %s\nwant     %s", got, tt.want)
			}
		})
	}
}

// swapCommands replaces the command table for the test and drops the idle
// parser, which execute built from the table before the swap.
func swapCommands(t *testing.T, nodes []Node) {
	t.Helper()
	saved := commands
	drop := func() {
		select {
		case <-idleParser:
		default:
		}
	}
	commands = nodes
	drop()
	t.Cleanup(func() {
		commands = saved
		drop()
	})
}

func TestExecuteRejectsBadTable(t *testing.T) {
	swapCommands(t, []Node{Leaf{Name: "nope", Help: "Nope.", Endpoint: "nope.nope", Output: JSON{}}})
	var stdout, stderr bytes.Buffer
	code := execute([]string{"--help"}, "test", "none", "never", &stdout, &stderr)
	if code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
	if want := "gccli: command table: nope: endpoint \"nope.nope\" is not in the endpoint table\n"; stderr.String() != want {
		t.Errorf("stderr:\n%s\nwant:\n%s", stderr.String(), want)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout is not empty:\n%s", stdout.String())
	}
}

func TestExecuteRunsAreIsolated(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	storeTestTokens(t, store, harnessAccount, harnessTokens(harnessAccount))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(map[string]string{"uri": r.URL.RequestURI()})
	}))
	t.Cleanup(server.Close)
	overrideNewClient(t, server)

	const n = 40
	outs := make([]string, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			args := []string{"--account", harnessAccount, "--color", "never", "--units", "metric", "--json", "activities", "list", "--limit", strconv.Itoa(i + 1)}
			if i%2 == 0 {
				args = append(args, "--type", "t"+strconv.Itoa(i))
			}
			var stdout, stderr bytes.Buffer
			if code := execute(args, "test", "none", "never", &stdout, &stderr); code != 0 {
				t.Errorf("run %d: exit %d: %s", i, code, stderr.String())
			}
			outs[i] = stdout.String()
		}()
	}
	wg.Wait()
	for i, out := range outs {
		want := fmt.Sprintf("/activitylist-service/activities/search/activities?start=0&limit=%d", i+1)
		if i%2 == 0 {
			want += "&activityType=t" + strconv.Itoa(i)
		}
		if !strings.Contains(out, strconv.Quote(want)) {
			t.Errorf("run %d requested another run's values: %s\nwant %s", i, strings.TrimSpace(out), want)
		}
	}
}

// TestExecuteResetsFlagsBetweenRuns runs three commands in one process on
// the reused parser. The first run gives pointer, list and negated flags,
// the second run leaves them out, so its request must not carry them, and
// the third run gives the bool flag in its positive form, which the negated
// form of the first run must not flip.
func TestExecuteResetsFlagsBetweenRuns(t *testing.T) {
	const log = "/periodichealth-service/menstrualcycle/dailylog/2026-10-01"
	saved := `{"calendarDate":"2026-10-01","flow":"LIGHT","notes":"x","ovulationDay":true}`
	h := newHarness(t, harnessInput{routes: map[string]mockReply{
		"GET " + log:  {body: saved},
		"POST " + log: {body: saved},
	}})
	for _, args := range [][]string{
		{"wellness", "menstrual-log", "set", "2026-10-01", "--flow", "heavy", "--symptoms", "acne,cramps", "--no-ovulation", "--notes", "y"},
		{"wellness", "menstrual-log", "set", "2026-10-01", "--moods", "happy"},
		{"wellness", "menstrual-log", "set", "2026-10-01", "--ovulation"},
	} {
		if _, stderr, code := h.run(nil, args); code != 0 {
			t.Fatalf("%v: exit %d: %s", args, code, stderr)
		}
	}
	reqs := h.requests()
	if len(reqs) != 6 {
		t.Fatalf("got %d requests, want 6", len(reqs))
	}
	body := func(fields string) string {
		return "POST " + log + "\n  {\n    \"calendarDate\": \"2026-10-01\",\n    \"flow\": \"LIGHT\",\n" + fields +
			"    \"notes\": \"x\",\n    \"ovulationDay\": true,\n    \"reportTimestamp\": \"2026-10-15T09:30:00.000\"\n  }"
	}
	for i, want := range map[int]string{
		3: body("    \"moods\": [\n      \"HAPPY\"\n    ],\n"),
		5: body(""),
	} {
		if got := strings.TrimRight(reqs[i].entry, "\n"); got != want {
			t.Errorf("request %d:\n%s\nwant:\n%s", i, got, want)
		}
	}
}

// TestExecuteResetsParseChecksBetweenRuns covers the checks that read what
// the Parse before left in the Kong model: a required flag the first run
// gave and the second run leaves out, and a file positional the first run
// gave and the second run points at a missing file. --help stops the first
// run after the trace, before a request.
func TestExecuteResetsParseChecksBetweenRuns(t *testing.T) {
	h := newHarness(t, harnessInput{files: map[string]string{"workout.json": "{}"}})
	for _, tt := range []struct {
		name          string
		first, second []string
		wantErr       string
	}{
		{
			name:    "required flag",
			first:   []string{"health", "progress", "summary", "--start", "2026-01-01", "--end", "2026-01-02", "--metric", "distance", "--help"},
			second:  []string{"health", "progress", "summary", "--start", "2026-01-01", "--end", "2026-01-02"},
			wantErr: "gccli: error: missing flags: --metric=METRIC,...",
		},
		{
			name:    "existing file",
			first:   []string{"workouts", "upload", "workout.json", "--help"},
			second:  []string{"workouts", "upload", "missing.json"},
			wantErr: "/missing.json: no such file or directory",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, stderr, code := h.run(nil, tt.first); code != 0 {
				t.Fatalf("first run: exit %d: %s", code, stderr)
			}
			_, stderr, code := h.run(nil, tt.second)
			if code != kindUsage.exit {
				t.Errorf("second run: exit %d, want %d", code, kindUsage.exit)
			}
			if !strings.Contains(stderr, tt.wantErr) {
				t.Errorf("second run stderr:\n%s\nwant a line %q", stderr, tt.wantErr)
			}
		})
	}
	if reqs := h.requests(); len(reqs) != 0 {
		t.Errorf("sent %d requests, want none", len(reqs))
	}
}

// TestCommandStateIsResetByParse lists every field of the command tree that
// Kong does not reset in Parse: an unexported field or a kong:"-" field.
// execute reuses the command structs, so such a field keeps its value from
// the run before. Each one must be set on every run before Run reads it, or
// never change after attach. Add a field here after you check that.
func TestCommandStateIsResetByParse(t *testing.T) {
	known := map[string]bool{
		"cmd.ChallengesAdhocCreateCmd.first": true,
		"cmd.ChallengesAdhocCreateCmd.last":  true,
		"cmd.GroupsLeaderboardCmd.first":     true,
		"cmd.GroupsLeaderboardCmd.last":      true,
		"cmd.specLeafG.leaf":                 true,
	}
	used := map[string]bool{}
	seen := map[reflect.Type]bool{}
	var walk func(typ reflect.Type)
	walk = func(typ reflect.Type) {
		for typ.Kind() == reflect.Pointer {
			typ = typ.Elem()
		}
		if typ.Kind() != reflect.Struct || seen[typ] || typ.PkgPath() != reflect.TypeFor[CLI]().PkgPath() {
			return
		}
		seen[typ] = true
		typeName, _, _ := strings.Cut(typ.String(), "[")
		for i := range typ.NumField() {
			f := typ.Field(i)
			name := typeName + "." + f.Name
			switch {
			case f.Anonymous:
				walk(f.Type)
			case !f.IsExported(), f.Tag.Get("kong") == "-":
				if !known[name] {
					t.Errorf("%s is not reset by Parse and is not in the known list", name)
				}
				used[name] = true
			default:
				walk(f.Type)
			}
		}
	}
	for _, p := range flatten(attach(commands)) {
		walk(reflect.TypeOf(p).Elem().Field(0).Type)
	}
	for name := range known {
		if !used[name] {
			t.Errorf("%s is in the known list but not in the command tree", name)
		}
	}
}

func flatten(plugins kong.Plugins) []any {
	var out []any
	for _, p := range plugins {
		if g, ok := reflect.ValueOf(p).Elem().Field(0).Interface().(group); ok {
			out = append(out, flatten(g.Plugins)...)
			continue
		}
		out = append(out, p)
	}
	return out
}

func TestUnitsEnumListsEveryPreference(t *testing.T) {
	for _, flag := range newTestParser(t).Model.Flags {
		if flag.Name == "units" {
			if want := strings.Join(units.PreferenceNames(), ","); flag.Enum != want {
				t.Errorf("--units enum = %q, units.PreferenceNames gives %q", flag.Enum, want)
			}
			return
		}
	}
	t.Fatal("no --units flag")
}
