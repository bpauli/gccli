package cmd

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alecthomas/kong"

	"github.com/bpauli/gccli/internal/garminapi"
	"github.com/bpauli/gccli/internal/garminauth"
	"github.com/bpauli/gccli/internal/testutil/requestlog"
)

type writeCase struct {
	args   []string
	files  map[string]string
	routes map[string]string
	result string
	prompt string

	json     string
	notes    string
	human    string
	fallback string
	table    string
	plain    string
}

const noContent = ""

const okOut = "{\n  \"ok\": true\n}\n"

var writeCases = map[string]writeCase{
	"activity upload": {
		args:   []string{"run.fit"},
		files:  map[string]string{"run.fit": "fit"},
		routes: map[string]string{"POST /upload-service/upload": `{"detailedImportResult":{"uploadId":5}}`},
		json:   "{\n  \"detailedImportResult\": {\n    \"uploadId\": 5\n  }\n}\n",
		human:  "Uploaded $DIR/run.fit\n",
	},
	"activity create": {
		args:     []string{"--name", "Swim", "--type", "swimming", "--duration", "30m", "--date", "2024-06-15T07:30:00"},
		routes:   map[string]string{"POST /activity-service/activity": `{"activityId":21234567890}`},
		json:     "{\n  \"activityId\": 21234567890\n}\n",
		human:    "Created activity 21234567890\n",
		fallback: "Activity created\n",
	},
	"activity rename": {
		args:   []string{"1", "Morning Run"},
		routes: map[string]string{"PUT /activity-service/activity/1": `{"activityId":1,"activityName":"Morning Run"}`},
		json:   "{\n  \"activityId\": 1,\n  \"activityName\": \"Morning Run\"\n}\n",
		human:  "Renamed activity 1 to \"Morning Run\"\n",
	},
	"activity retype": {
		args:   []string{"1", "--type-id", "1", "--type-key", "running"},
		routes: map[string]string{"PUT /activity-service/activity/1": `{"activityId":1}`},
		json:   "{\n  \"activityId\": 1\n}\n",
		human:  "Changed activity 1 type to running\n",
	},
	"activity exercise-sets set": {
		args:   []string{"1", "-e", "BENCH_PRESS/BARBELL_BENCH_PRESS:12@20"},
		routes: map[string]string{"PUT /activity-service/activity/1/exerciseSets": noContent},
		json:   okOut,
		human:  "Exercise sets updated for activity 1\n",
	},
	"body add-weight": {
		args:   []string{"80.5"},
		routes: map[string]string{"POST /weight-service/user-weight": `{"samplePk":3}`},
		json:   "{\n  \"samplePk\": 3\n}\n",
		human:  "Added weight: 80.5 kg\n",
	},
	"body add-composition": {
		args:   []string{"80"},
		routes: map[string]string{"POST /upload-service/upload": `{"detailedImportResult":{"uploadId":6}}`},
		json:   "{\n  \"detailedImportResult\": {\n    \"uploadId\": 6\n  }\n}\n",
		human:  "Added body composition: 80.0 kg\n",
	},
	"body add-blood-pressure": {
		args:   []string{"--systolic", "120", "--diastolic", "80", "--pulse", "60"},
		routes: map[string]string{"POST /bloodpressure-service/bloodpressure": `{"version":4}`},
		json:   "{\n  \"version\": 4\n}\n",
		human:  "Added blood pressure: 120/80 mmHg, pulse 60 bpm\n",
	},
	"workouts upload": {
		args:   []string{"w.json"},
		files:  map[string]string{"w.json": `{"workoutName":"Intervals"}`},
		routes: map[string]string{"POST /workout-service/workout": `{"workoutId":8}`},
		json:   "{\n  \"workoutId\": 8\n}\n",
		human:  "Uploaded workout from $DIR/w.json\n",
	},
	"workouts update": {
		args:   []string{"7", "w.json"},
		files:  map[string]string{"w.json": `{"workoutId":8,"workoutName":"Intervals"}`},
		routes: map[string]string{"PUT /workout-service/workout/7": noContent},
		json:   okOut,
		notes:  "Workout JSON has workoutId 8, updating workout 7 instead\n",
		human:  "Updated workout 7 from $DIR/w.json\n",
	},
	"workouts create": {
		args:     []string{"Tempo", "--type", "run", "--step", "warmup:10m"},
		routes:   map[string]string{"POST /workout-service/workout": `{"workoutId":9}`},
		json:     "{\n  \"workoutId\": 9\n}\n",
		human:    "Created workout \"Tempo\" (ID: 9)\n",
		fallback: "Created workout \"Tempo\"\n",
	},
	"workouts schedule add": {
		args:   []string{"9", "2024-01-15"},
		routes: map[string]string{"POST /workout-service/schedule/9": `{"workoutScheduleId":10}`},
		json:   "{\n  \"workoutScheduleId\": 10\n}\n",
		human:  "Scheduled workout 9 on 2024-01-15\n",
	},
	"courses import": {
		args:  []string{"route.gpx"},
		files: map[string]string{"route.gpx": "<gpx/>"},
		routes: map[string]string{
			"POST /course-service/course/import":                          `{"courseName":"Loop","geoPoints":[{"latitude":1,"longitude":2}]}`,
			"POST /course-service/course/elevation?smoothingEnabled=true": `[[1,2,100]]`,
			"POST /course-service/course":                                 `{"courseId":42,"courseName":"Loop","distanceMeter":10000,"elevationGainMeter":100,"elevationLossMeter":90}`,
		},
		result:   "POST /course-service/course",
		json:     "{\n  \"courseId\": 42,\n  \"courseName\": \"Loop\",\n  \"distanceMeter\": 10000,\n  \"elevationGainMeter\": 100,\n  \"elevationLossMeter\": 90\n}\n",
		notes:    "Importing route.gpx...\n",
		fallback: "Imported course from route.gpx\n",
		table:    "ID  NAME  DISTANCE  ELEV GAIN  ELEV LOSS\n42  Loop  10.00 km  100 m      90 m\n",
		plain:    "42\tLoop\t10.00 km\t100 m\t90 m\n",
	},
	"courses send": {
		args: []string{"42", "99"},
		routes: map[string]string{
			"GET /course-service/course/42":               `{"courseName":"Loop"}`,
			"POST /device-service/devicemessage/messages": `[{"deviceName":"Edge","messageId":7}]`,
		},
		result:   "POST /device-service/devicemessage/messages",
		json:     "[\n  {\n    \"deviceName\": \"Edge\",\n    \"messageId\": 7\n  }\n]\n",
		human:    "Sent course \"Loop\" to device Edge (message 7)\n",
		fallback: "Sent course \"Loop\" to device 99\n",
	},
	"events add": {
		args:     []string{"--name", "Race", "--date", "2024-06-01", "--type", "running"},
		routes:   map[string]string{"POST /calendar-service/event": `{"id":11,"eventName":"Race","date":"2024-06-01"}`},
		json:     "{\n  \"id\": 11,\n  \"eventName\": \"Race\",\n  \"date\": \"2024-06-01\"\n}\n",
		human:    "Added event \"Race\" on 2024-06-01 (ID: 11)\n",
		fallback: "Added event\n",
	},
	"gear add": {
		args:     []string{"Pegasus", "--brand", "Nike", "--model", "Pegasus", "--first-use", "2024-06-01"},
		routes:   map[string]string{"POST /gear-service/gear/v2": `{"uuid":"0b5c3d2e-1f4a-4b6c-8d7e-9f0a1b2c3d4e","name":"Pegasus"}`},
		json:     "{\n  \"uuid\": \"0b5c3d2e-1f4a-4b6c-8d7e-9f0a1b2c3d4e\",\n  \"name\": \"Pegasus\"\n}\n",
		human:    "Added gear \"Pegasus\" (UUID: 0b5c3d2e-1f4a-4b6c-8d7e-9f0a1b2c3d4e)\n",
		fallback: "Added gear \"Pegasus\"\n",
	},
	"gear update": {
		args: []string{"0b5c3d2e1f4a4b6c8d7e9f0a1b2c3d4e", "--name", "Old"},
		routes: map[string]string{
			"GET /gear-service/gear/v2/0b5c3d2e-1f4a-4b6c-8d7e-9f0a1b2c3d4e": `{"uuid":"0b5c3d2e-1f4a-4b6c-8d7e-9f0a1b2c3d4e","name":"Pegasus"}`,
			"PUT /gear-service/gear/v2/0b5c3d2e-1f4a-4b6c-8d7e-9f0a1b2c3d4e": noContent,
		},
		result:   "PUT /gear-service/gear/v2/0b5c3d2e-1f4a-4b6c-8d7e-9f0a1b2c3d4e",
		json:     okOut,
		human:    "Updated gear 0b5c3d2e-1f4a-4b6c-8d7e-9f0a1b2c3d4e\n",
		fallback: "Updated gear 0b5c3d2e-1f4a-4b6c-8d7e-9f0a1b2c3d4e\n",
	},
	"hydration add": {
		args:   []string{"250", "--date", "2024-01-15"},
		routes: map[string]string{"PUT /usersummary-service/usersummary/hydration/log": `{"valueInML":250}`},
		json:   "{\n  \"valueInML\": 250\n}\n",
		human:  "Added 250 ml of water on 2024-01-15\n",
	},
}

type writeMode struct {
	name  string
	flags []string
	empty bool
}

var writeModes = []writeMode{
	{"json", []string{"--json"}, false},
	{"json-empty", []string{"--json"}, true},
	{"table", nil, false},
	{"table-empty", nil, true},
	{"plain", []string{"--plain"}, false},
}

func (tc writeCase) want(m writeMode) (stdout, stderr string) {
	switch m.name {
	case "json":
		return tc.json, tc.notes
	case "json-empty":
		return okOut, tc.notes
	case "table":
		return tc.table, tc.notes + tc.human
	case "table-empty":
		return "", tc.notes + tc.fallback
	default:
		return tc.plain, tc.notes + tc.human
	}
}

func tableWriteCases(t *testing.T) map[string]writeCase {
	t.Helper()
	cases := map[string]writeCase{}
	walkNodes(commands, "", func(path string, n Node) {
		l, ok := n.(Leaf)
		if !ok {
			return
		}
		w, ok := l.Output.(Write)
		if !ok {
			return
		}
		g, err := readGolden(goldenFile(path))
		if err != nil {
			t.Fatal(err)
		}
		args, in, err := g.input()
		if err != nil {
			t.Fatalf("%s: %v", goldenFile(path), err)
		}
		tc := writeCase{args: args[len(strings.Fields(path)):], files: in.files, routes: map[string]string{}}
		for route, reply := range in.routes {
			tc.routes[route] = reply.body
		}
		tc.result = tc.resultRoute(t)
		tc.json = g.sections["json stdout"]
		tc.human = g.sections["table stderr"]
		if w.Confirm != "" {
			tc.prompt, tc.human, _ = strings.Cut(tc.human, " [y/N]: ")
		}
		cases[path] = tc
	})
	return cases
}

func TestTableWriteCasesCoverWriteRows(t *testing.T) {
	var want []string
	walkNodes(commands, "", func(path string, n Node) {
		if l, ok := n.(Leaf); ok {
			if _, ok := l.Output.(Write); ok {
				want = append(want, path)
			}
		}
	})
	slices.Sort(want)
	got := slices.Sorted(maps.Keys(tableWriteCases(t)))
	if !slices.Equal(got, want) {
		t.Errorf("tableWriteCases covers %q\nwant the Leaf rows with a Write output %q", got, want)
	}
}

func allWriteCases(t *testing.T) map[string]writeCase {
	t.Helper()
	cases := maps.Clone(writeCases)
	for path, tc := range tableWriteCases(t) {
		if _, dup := cases[path]; dup {
			t.Errorf("writeCases[%q] names a Leaf row: remove the row", path)
		}
		cases[path] = tc
	}
	return cases
}

func TestWriteCommandOutput(t *testing.T) {
	cases := allWriteCases(t)
	for _, path := range slices.Sorted(maps.Keys(cases)) {
		tc := cases[path]
		for _, m := range writeModes {
			if m.name == "table-empty" && tc.fallback == "" {
				continue
			}
			t.Run(path+"/"+m.name, func(t *testing.T) {
				tc := tc
				if m.empty {
					tc.routes = maps.Clone(tc.routes)
					tc.routes[tc.resultRoute(t)] = noContent
				}
				h := newHarness(t, tc.input())
				args := append(strings.Fields(path), tc.args...)
				if tc.prompt != "" {
					args = append(args, "--force")
				}
				stdout, stderr, code := h.run(m.flags, args)
				wantOut, wantErr := tc.want(m)
				wantOut, wantErr = strings.ReplaceAll(wantOut, "$DIR", h.dir), strings.ReplaceAll(wantErr, "$DIR", h.dir)
				if code != 0 || stdout != wantOut || stderr != wantErr {
					t.Errorf("exit %d, want 0\nstdout %q\nwant   %q\nstderr %q\nwant   %q", code, stdout, wantOut, stderr, wantErr)
				}
				if len(h.writes()) == 0 {
					t.Error("no write request reached the mock server")
				}
			})
		}
	}
}

func TestWriteCommandDeclined(t *testing.T) {
	cases := allWriteCases(t)
	for _, path := range slices.Sorted(maps.Keys(cases)) {
		tc := cases[path]
		if tc.prompt == "" {
			continue
		}
		for _, m := range writeModes {
			if m.empty {
				continue
			}
			t.Run(path+"/"+m.name, func(t *testing.T) {
				in := tc.input()
				in.stdin = "n\n"
				h := newHarness(t, in)

				stdout, stderr, code := h.run(m.flags, append(strings.Fields(path), tc.args...))
				want := tc.prompt + " [y/N]: Cancelled\n"
				if code != 0 || stdout != "" || stderr != want {
					t.Errorf("exit %d, want 0\nstdout %q, want empty\nstderr %q\nwant   %q", code, stdout, stderr, want)
				}
				if w := h.writes(); len(w) != 0 {
					t.Errorf("declined prompt sent writes: %v", w)
				}
			})
		}
	}
}

func TestWriteCommandNotInteractive(t *testing.T) {
	const msg = "stdin is not a terminal: use --force to confirm"
	cases := allWriteCases(t)
	for _, path := range slices.Sorted(maps.Keys(cases)) {
		tc := cases[path]
		if tc.prompt == "" {
			continue
		}
		for _, m := range writeModes {
			if m.empty {
				continue
			}
			t.Run(path+"/"+m.name, func(t *testing.T) {
				h := newHarness(t, tc.input())
				useNonTerminalStdin(t)

				stdout, stderr, code := h.run(m.flags, append(strings.Fields(path), tc.args...))
				want := msg + "\n"
				if m.name == "json" {
					want = `{"error":{"code":"error","message":"` + msg + `"}}` + "\n"
				}
				if code != 1 || stdout != "" || stderr != want {
					t.Errorf("exit %d, want 1\nstdout %q, want empty\nstderr %q\nwant   %q", code, stdout, stderr, want)
				}
				if w := h.writes(); len(w) != 0 {
					t.Errorf("non-interactive run sent writes: %v", w)
				}
			})
		}
	}
}

func TestWriteCommandNotJSON(t *testing.T) {
	tc := writeCases["activity rename"]
	tc.routes = map[string]string{tc.resultRoute(t): "<html>saved</html>"}
	h := newHarness(t, tc.input())

	stdout, stderr, code := h.run([]string{"--json"}, append([]string{"activity", "rename"}, tc.args...))
	want := `{"error":{"code":"error","message":"garmin saved the change, but its response is not JSON: \"<html>saved</html>\""}}` + "\n"
	if code != 1 || stdout != "" || stderr != want {
		t.Errorf("exit %d, want 1\nstdout %q, want empty\nstderr %q\nwant   %q", code, stdout, stderr, want)
	}
}

func (tc writeCase) resultRoute(t *testing.T) string {
	t.Helper()
	if tc.result != "" {
		return tc.result
	}
	var data []string
	for route := range tc.routes {
		if !isLookup(route) {
			data = append(data, route)
		}
	}
	if len(data) != 1 {
		t.Fatalf("the case has %d data routes %q: set result", len(data), data)
	}
	return data[0]
}

func TestResultRouteSkipsLookups(t *testing.T) {
	tc := writeCase{routes: map[string]string{profileRoute: profileBody, socialRoute: profileBody, "DELETE /x/1": noContent}}
	for range 20 {
		if got := tc.resultRoute(t); got != "DELETE /x/1" {
			t.Fatalf("resultRoute = %q, want the one data route", got)
		}
	}
}

type mockReply struct {
	status int
	body   string
}

type harnessInput struct {
	files  map[string]string
	routes map[string]mockReply
	stdin  string
}

func (tc writeCase) input() harnessInput {
	routes := map[string]mockReply{}
	for route, body := range tc.routes {
		reply := mockReply{body: body}
		if body == noContent {
			reply.status = http.StatusNoContent
		}
		routes[route] = reply
	}
	return harnessInput{files: tc.files, routes: routes}
}

var harnessNow = time.Date(2026, 10, 15, 9, 30, 0, 0, time.UTC)

const harnessAccount = "test@example.com"

func harnessTokens(email string) *garminauth.Tokens {
	return &garminauth.Tokens{
		OAuth2AccessToken: "test-access-token",
		OAuth2ExpiresAt:   time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC),
		Domain:            "garmin.com",
		Email:             email,
	}
}

type harness struct {
	dir  string
	home string
	mu   sync.Mutex
	log  []loggedRequest
}

type loggedRequest struct {
	method string
	route  string
	entry  string
}

func newHarness(t *testing.T, in harnessInput) *harness {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	dir := t.TempDir()
	t.Chdir(dir)
	for name, content := range in.files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	h := &harness{dir: dir, home: home}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := r.Method + " " + r.URL.RequestURI()
		entry, err := requestlog.Format(r)
		if err != nil {
			t.Errorf("%s: %v", route, err)
		}
		h.mu.Lock()
		h.log = append(h.log, loggedRequest{r.Method, route, entry})
		h.mu.Unlock()
		reply, ok := in.routes[route]
		switch {
		case !ok:
			t.Errorf("unexpected request %s", route)
			w.WriteHeader(http.StatusNotFound)
		default:
			if reply.body != "" {
				w.Header().Set("Content-Type", "application/json")
			}
			w.WriteHeader(cmp.Or(reply.status, http.StatusOK))
			_, _ = w.Write([]byte(reply.body))
		}
	}))
	t.Cleanup(server.Close)

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, harnessAccount, harnessTokens(harnessAccount))

	stub(t, &nowFn, func() time.Time { return harnessNow })
	stub(t, &confirmReader, io.Reader(strings.NewReader(in.stdin)))
	stub(t, &readStdinFn, func() ([]byte, error) { return []byte(in.stdin), nil })
	stub(t, &readPasswordFn, func(int) ([]byte, error) { return []byte("secret"), nil })
	stub(t, &loginBrowserFn, func(_ context.Context, email string, _ garminauth.LoginOptions) (*garminauth.Tokens, error) {
		return harnessTokens(email), nil
	})
	stub(t, &loginHeadlessFn, func(_ context.Context, email, _ string, _ garminauth.LoginOptions) (*garminauth.Tokens, error) {
		return harnessTokens(email), nil
	})
	stub(t, &refreshFn, func(context.Context, *garminauth.Tokens, garminauth.LoginOptions) (*garminauth.Tokens, error) {
		return nil, errors.New("the test harness does not refresh tokens")
	})
	// Mirrors garminapi.FetchExerciseCatalog, whose URL is fixed, so that the
	// golden files show the error text of the binary.
	stub(t, &fetchExerciseCatalogFn, func(ctx context.Context) (json.RawMessage, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/web-data/exercises/Exercises.json", nil)
		if err != nil {
			return nil, err
		}
		resp, err := server.Client().Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetch exercise catalog: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("fetch exercise catalog: HTTP %d", resp.StatusCode)
		}
		return io.ReadAll(resp.Body)
	})
	return h
}

func stub[T any](t *testing.T, v *T, fake T) {
	orig := *v
	*v = fake
	t.Cleanup(func() { *v = orig })
}

func (h *harness) run(mode, args []string) (stdout, stderr string, code int) {
	var out, errOut bytes.Buffer
	global := append([]string{"--account", harnessAccount, "--color", "never", "--units", "metric"}, mode...)
	code = execute(append(global, args...), "test", "none", "never", &out, &errOut)
	return out.String(), errOut.String(), code
}

func (h *harness) writes() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	var writes []string
	for _, r := range h.log {
		if r.method != http.MethodGet {
			writes = append(writes, r.route)
		}
	}
	return writes
}

func (h *harness) requests() []loggedRequest {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.log)
}

func TestWriteCensus(t *testing.T) {
	fset := token.NewFileSet()
	cli := parseFuncs(t, fset, ".")
	methods := clientMethods(parseFuncs(t, fset, "../garminapi"))
	apiWrites := apiWriteMethods(methods, tableWrites(garminapi.Describe, cli, slices.Collect(maps.Values(methods))))
	for name := range apiWrites {
		fn, ok := methods[name]
		if !ok {
			continue
		}
		if res := fn.Type.Results; ast.IsExported(name) && (res == nil || len(res.List) != 2 ||
			types.ExprString(res.List[0].Type) != "json.RawMessage" || types.ExprString(res.List[1].Type) != "error") {
			t.Errorf("%s: Client.%s sends a write: return (json.RawMessage, error) so the command can print the response", fset.Position(fn.Pos()), name)
		}
	}

	paths := map[string][]string{}
	commandPaths(newTestParser(t).Model.Node, "", paths)
	banned := []string{"IsJSON", "ModeFromContext", "WriteJSON", "WriteRows", "writeHealthJSON", "Stdout", "Successf"}
	found := map[string]bool{}
	for fn, method := range runWrites(cli, apiWrites) {
		pos := fset.Position(fn.Pos())
		recv := recvName(fn)
		if len(paths[recv]) == 0 {
			t.Errorf("%s: %s.Run sends a write but is not a command in the CLI tree", pos, recv)
		}
		ids, called := idents(fn.Body), calls(fn.Body)
		for _, path := range paths[recv] {
			found[path] = true
			if _, ok := writeCases[path]; !ok {
				t.Errorf("%s: %q sends a write to Garmin through Client.%s: add a row to writeCases in write_test.go", pos, path, method)
			}
			if !called["reportWrite"] && !called["reportWriteTable"] {
				t.Errorf("%s: %q sends a write: end Run with reportWrite or reportWriteTable (write.go)", pos, path)
			}
			for _, name := range banned {
				if ids[name] {
					t.Errorf("%s: %q names %s: print the write result only through reportWrite or reportWriteTable", pos, path, name)
				}
			}
		}
	}
	for path := range writeCases {
		if !found[path] {
			t.Errorf("writeCases[%q] in write_test.go names no command that sends a write: remove the row", path)
		}
	}
}

func TestWriteCensusFindsIndirectWrites(t *testing.T) {
	fset := token.NewFileSet()
	api := parseSource(t, fset, `package garminapi
func (c *Client) PatchNote(ctx context.Context) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, "PATCH", "/note", nil)
}
func (c *Client) SaveNote(ctx context.Context) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodPost, "/note", nil)
}
func (c *Client) GetNote(ctx context.Context) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodGet, "/note", nil)
}`)
	cli := parseSource(t, fset, `package cmd
func (c *LiteralVerbCmd) Run(g *Globals) error {
	_, err := client.PatchNote(g.Context)
	return err
}
func (c *HelperCmd) Run(g *Globals) error { return c.save(g) }
func (c *HelperCmd) save(g *Globals) error { return storeNote(g) }
func storeNote(g *Globals) error {
	_, err := client.SaveNote(g.Context)
	return err
}
func (c *ReadCmd) Run(g *Globals) error {
	_, err := client.GetNote(g.Context)
	return err
}`)

	got := map[string]string{}
	for fn, method := range runWrites(cli, apiWriteMethods(clientMethods(api), nil)) {
		got[recvName(fn)] = method
	}
	want := map[string]string{"LiteralVerbCmd": "PatchNote", "HelperCmd": "SaveNote"}
	if !maps.Equal(got, want) {
		t.Errorf("runWrites = %v, want %v", got, want)
	}
}

func TestWriteCensusFindsCallWrites(t *testing.T) {
	fset := token.NewFileSet()
	api := parseSource(t, fset, `package garminapi
func (c *Client) RenameNote(ctx context.Context, id string) (json.RawMessage, error) {
	body, err := json.Marshal(id)
	if err != nil {
		return nil, err
	}
	return c.Call(ctx, "notes.update", map[string]string{"id": id}, body)
}`)
	cli := parseSource(t, fset, `package cmd
func (c *DeleteCmd) Run(g *Globals) error {
	_, err := client.Call(g.Context, "notes.delete", map[string]string{"id": c.ID}, nil)
	return err
}
func (c *HelperCmd) Run(g *Globals) error { return c.rename(g) }
func (c *HelperCmd) rename(g *Globals) error {
	_, err := client.RenameNote(g.Context, c.ID)
	return err
}
func (c *ReadCmd) Run(g *Globals) error {
	_, err := client.Call(g.Context, "notes.get", map[string]string{"id": c.ID}, nil)
	return err
}`)
	table := map[string]string{"notes.delete": http.MethodDelete, "notes.update": http.MethodPut, "notes.get": http.MethodGet}
	lookup := func(name string) (garminapi.Endpoint, bool) {
		m, ok := table[name]
		return garminapi.Endpoint{Method: m}, ok
	}

	got := map[string]string{}
	methods := clientMethods(api)
	for fn, method := range runWrites(cli, apiWriteMethods(methods, tableWrites(lookup, cli, api))) {
		got[recvName(fn)] = method
	}
	want := map[string]string{"DeleteCmd": `Call("notes.delete")`, "HelperCmd": "RenameNote"}
	if !maps.Equal(got, want) {
		t.Errorf("runWrites = %v, want %v", got, want)
	}
}

func clientMethods(fns []*ast.FuncDecl) map[string]*ast.FuncDecl {
	methods := map[string]*ast.FuncDecl{}
	for _, fn := range fns {
		if recvName(fn) == "Client" {
			methods[fn.Name.Name] = fn
		}
	}
	return methods
}

type clientMethod = string

func apiWriteMethods(methods map[string]*ast.FuncDecl, table map[string]clientMethod) map[string]clientMethod {
	writes := map[string]clientMethod{}
	maps.Copy(writes, table)
	for name, fn := range methods {
		if namesWriteVerb(fn.Body) {
			writes[name] = name
		}
	}
	return reachWrites(slices.Collect(maps.Values(methods)), writes)
}

func runWrites(fns []*ast.FuncDecl, apiWrites map[string]clientMethod) map[*ast.FuncDecl]clientMethod {
	writes := map[string]clientMethod{}
	for name := range apiWrites {
		writes[name] = name
	}
	var helpers, runs []*ast.FuncDecl
	for _, fn := range fns {
		if fn.Name.Name == "Run" && fn.Recv != nil {
			runs = append(runs, fn)
		} else {
			helpers = append(helpers, fn)
		}
	}
	writes = reachWrites(helpers, writes)
	out := map[*ast.FuncDecl]clientMethod{}
	for _, fn := range runs {
		if callee, ok := firstCallIn(fn.Body, writes); ok {
			out[fn] = writes[callee]
		}
	}
	return out
}

func reachWrites(fns []*ast.FuncDecl, writes map[string]clientMethod) map[string]clientMethod {
	for grew := true; grew; {
		grew = false
		for _, fn := range fns {
			if _, ok := writes[fn.Name.Name]; ok {
				continue
			}
			if callee, ok := firstCallIn(fn.Body, writes); ok {
				writes[fn.Name.Name], grew = writes[callee], true
			}
		}
	}
	return writes
}

func firstCallIn(body ast.Node, writes map[string]clientMethod) (string, bool) {
	for _, name := range slices.Sorted(maps.Keys(calls(body))) {
		if _, ok := writes[name]; ok {
			return name, true
		}
	}
	return "", false
}

var writeVerbs = map[string]bool{
	"MethodPost": true, "MethodPut": true, "MethodPatch": true, "MethodDelete": true,
	`"POST"`: true, `"PUT"`: true, `"PATCH"`: true, `"DELETE"`: true,
}

func namesWriteVerb(body ast.Node) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.Ident:
			found = found || writeVerbs[n.Name]
		case *ast.BasicLit:
			found = found || (n.Kind == token.STRING && writeVerbs[n.Value])
		}
		return !found
	})
	return found
}

func recvName(fn *ast.FuncDecl) string {
	if fn.Recv == nil {
		return ""
	}
	return strings.TrimPrefix(types.ExprString(fn.Recv.List[0].Type), "*")
}

func parseFuncs(t *testing.T, fset *token.FileSet, dir string) []*ast.FuncDecl {
	t.Helper()
	names, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	var fns []*ast.FuncDecl
	for _, name := range names {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		fns = append(fns, funcDecls(f)...)
	}
	return fns
}

func parseSource(t *testing.T, fset *token.FileSet, src string) []*ast.FuncDecl {
	t.Helper()
	f, err := parser.ParseFile(fset, "fixture.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	return funcDecls(f)
}

func funcDecls(f *ast.File) []*ast.FuncDecl {
	var fns []*ast.FuncDecl
	for _, d := range f.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Body != nil {
			fns = append(fns, fn)
		}
	}
	return fns
}

func idents(n ast.Node) map[string]bool {
	out := map[string]bool{}
	ast.Inspect(n, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			out[id.Name] = true
		}
		return true
	})
	return out
}

func calls(n ast.Node) map[string]bool {
	out := map[string]bool{}
	ast.Inspect(n, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			switch fun := call.Fun.(type) {
			case *ast.Ident:
				out[fun.Name] = true
			case *ast.SelectorExpr:
				out[fun.Sel.Name] = true
				if name, ok := endpointName(call); ok {
					out[callKey(name)] = true
				}
			}
		}
		return true
	})
	return out
}

func callKey(endpoint string) string { return `Call("` + endpoint + `")` }

func endpointName(call *ast.CallExpr) (string, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Call" || len(call.Args) != 4 {
		return "", false
	}
	lit, ok := call.Args[1].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	name, err := strconv.Unquote(lit.Value)
	return name, err == nil
}

func tableWrites(lookup func(string) (garminapi.Endpoint, bool), fnSets ...[]*ast.FuncDecl) map[string]clientMethod {
	writes := map[string]clientMethod{}
	for _, fns := range fnSets {
		for _, fn := range fns {
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				name, ok := endpointName(call)
				if !ok {
					return true
				}
				if e, known := lookup(name); known && e.Method != http.MethodGet {
					writes[callKey(name)] = callKey(name)
				}
				return true
			})
		}
	}
	return writes
}

func commandPaths(n *kong.Node, prefix string, out map[string][]string) {
	for _, child := range n.Children {
		path := prefix
		if child.Type == kong.CommandNode {
			path = strings.TrimSpace(prefix + " " + child.Name)
			name := reflect.Indirect(child.Target).Type().Name()
			out[name] = append(out[name], path)
		}
		commandPaths(child, path, out)
	}
}
