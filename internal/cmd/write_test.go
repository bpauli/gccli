package cmd

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/alecthomas/kong"
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

const (
	deletedBody = `{"deleted":true}`
	deletedOut  = "{\n  \"deleted\": true\n}\n"
)

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
	"activity delete": {
		args:   []string{"1"},
		routes: map[string]string{"DELETE /activity-service/activity/1": deletedBody},
		prompt: "Delete activity 1?",
		json:   deletedOut,
		human:  "Deleted activity 1\n",
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
	"body delete-weight": {
		args:   []string{"3", "--date", "2024-01-15"},
		routes: map[string]string{"DELETE /weight-service/weight/2024-01-15/byversion/3": deletedBody},
		prompt: "Delete weigh-in 3 on 2024-01-15?",
		json:   deletedOut,
		human:  "Deleted weigh-in 3\n",
	},
	"body add-blood-pressure": {
		args:   []string{"--systolic", "120", "--diastolic", "80", "--pulse", "60"},
		routes: map[string]string{"POST /bloodpressure-service/bloodpressure": `{"version":4}`},
		json:   "{\n  \"version\": 4\n}\n",
		human:  "Added blood pressure: 120/80 mmHg, pulse 60 bpm\n",
	},
	"body delete-blood-pressure": {
		args:   []string{"4", "--date", "2024-01-15"},
		routes: map[string]string{"DELETE /bloodpressure-service/bloodpressure/2024-01-15/4": deletedBody},
		prompt: "Delete blood pressure entry 4 on 2024-01-15?",
		json:   deletedOut,
		human:  "Deleted blood pressure entry 4\n",
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
	"workouts schedule remove": {
		args:   []string{"10"},
		routes: map[string]string{"DELETE /workout-service/schedule/10": deletedBody},
		prompt: "Remove scheduled workout 10?",
		json:   deletedOut,
		human:  "Removed scheduled workout 10\n",
	},
	"workouts delete": {
		args:   []string{"9"},
		routes: map[string]string{"DELETE /workout-service/workout/9": deletedBody},
		prompt: "Delete workout 9?",
		json:   deletedOut,
		human:  "Deleted workout 9\n",
	},
	"courses import": {
		args:  []string{"route.gpx"},
		files: map[string]string{"route.gpx": "<gpx/>"},
		routes: map[string]string{
			"POST /course-service/course/import":    `{"courseName":"Loop","geoPoints":[{"latitude":1,"longitude":2}]}`,
			"POST /course-service/course/elevation": `[[1,2,100]]`,
			"POST /course-service/course":           `{"courseId":42,"courseName":"Loop","distanceMeter":10000,"elevationGainMeter":100,"elevationLossMeter":90}`,
		},
		result: "POST /course-service/course",
		json:   "{\n  \"courseId\": 42,\n  \"courseName\": \"Loop\",\n  \"distanceMeter\": 10000,\n  \"elevationGainMeter\": 100,\n  \"elevationLossMeter\": 90\n}\n",
		notes:  "Importing route.gpx...\n",
		table:  "ID  NAME  DISTANCE  ELEV GAIN  ELEV LOSS\n42  Loop  10.00 km  100 m      90 m\n",
		plain:  "42\tLoop\t10.00 km\t100 m\t90 m\n",
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
	"courses delete": {
		args:   []string{"42"},
		routes: map[string]string{"DELETE /course-service/course/42": deletedBody},
		prompt: "Delete course 42?",
		json:   deletedOut,
		human:  "Deleted course 42\n",
	},
	"events add": {
		args:     []string{"--name", "Race", "--date", "2024-06-01", "--type", "running"},
		routes:   map[string]string{"POST /calendar-service/event": `{"id":11,"eventName":"Race","date":"2024-06-01"}`},
		json:     "{\n  \"id\": 11,\n  \"eventName\": \"Race\",\n  \"date\": \"2024-06-01\"\n}\n",
		human:    "Added event \"Race\" on 2024-06-01 (ID: 11)\n",
		fallback: "Added event\n",
	},
	"events delete": {
		args:   []string{"11"},
		routes: map[string]string{"DELETE /calendar-service/event/11": deletedBody},
		prompt: "Delete event 11?",
		json:   deletedOut,
		human:  "Deleted event 11\n",
	},
	"gear link": {
		args:   []string{"abc", "1"},
		routes: map[string]string{"PUT /gear-service/gear/link/abc/activity/1": `{"gearPk":3}`},
		json:   "{\n  \"gearPk\": 3\n}\n",
		human:  "Linked gear abc to activity 1\n",
	},
	"gear unlink": {
		args:   []string{"abc", "1"},
		routes: map[string]string{"PUT /gear-service/gear/unlink/abc/activity/1": noContent},
		json:   okOut,
		human:  "Unlinked gear abc from activity 1\n",
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

func TestWriteCommandOutput(t *testing.T) {
	for _, path := range slices.Sorted(maps.Keys(writeCases)) {
		tc := writeCases[path]
		for _, m := range writeModes {
			if m.name == "table-empty" && tc.fallback == "" {
				continue
			}
			t.Run(path+"/"+m.name, func(t *testing.T) {
				tc := tc
				if m.empty {
					tc.routes = maps.Clone(tc.routes)
					tc.routes[tc.resultRoute()] = noContent
				}
				h := newWriteHarness(t, tc)
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
	for _, path := range slices.Sorted(maps.Keys(writeCases)) {
		tc := writeCases[path]
		if tc.prompt == "" {
			continue
		}
		for _, m := range writeModes {
			if m.empty {
				continue
			}
			t.Run(path+"/"+m.name, func(t *testing.T) {
				h := newWriteHarness(t, tc)
				orig := confirmReader
				confirmReader = strings.NewReader("n\n")
				t.Cleanup(func() { confirmReader = orig })

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
	for _, path := range slices.Sorted(maps.Keys(writeCases)) {
		tc := writeCases[path]
		if tc.prompt == "" {
			continue
		}
		for _, m := range writeModes {
			if m.empty {
				continue
			}
			t.Run(path+"/"+m.name, func(t *testing.T) {
				h := newWriteHarness(t, tc)
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
	tc.routes = map[string]string{tc.resultRoute(): "<html>saved</html>"}
	h := newWriteHarness(t, tc)

	stdout, stderr, code := h.run([]string{"--json"}, append([]string{"activity", "rename"}, tc.args...))
	want := `{"error":{"code":"error","message":"garmin saved the change, but its response is not JSON: \"<html>saved</html>\""}}` + "\n"
	if code != 1 || stdout != "" || stderr != want {
		t.Errorf("exit %d, want 1\nstdout %q, want empty\nstderr %q\nwant   %q", code, stdout, stderr, want)
	}
}

func (tc writeCase) resultRoute() string {
	if tc.result != "" {
		return tc.result
	}
	for route := range tc.routes {
		return route
	}
	return ""
}

type writeHarness struct {
	dir  string
	mu   sync.Mutex
	sent []string
}

func newWriteHarness(t *testing.T, tc writeCase) *writeHarness {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	for _, env := range []string{"ACCOUNT", "COLOR", "DOMAIN", "JSON", "KEYRING_BACKEND", "PLAIN", "UNITS"} {
		t.Setenv("GCCLI_"+env, "")
		if err := os.Unsetenv("GCCLI_" + env); err != nil {
			t.Fatal(err)
		}
	}
	dir := t.TempDir()
	t.Chdir(dir)
	for name, content := range tc.files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	h := &writeHarness{dir: dir}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := r.Method + " " + r.URL.Path
		if r.Method != http.MethodGet {
			h.mu.Lock()
			h.sent = append(h.sent, route)
			h.mu.Unlock()
		}
		body, ok := tc.routes[route]
		switch {
		case !ok:
			t.Errorf("unexpected request %s", route)
			w.WriteHeader(http.StatusInternalServerError)
		case body == noContent:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(body))
		}
	}))
	t.Cleanup(server.Close)

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())
	return h
}

func (h *writeHarness) run(mode, args []string) (stdout, stderr string, code int) {
	var out, errOut bytes.Buffer
	global := append([]string{"--account", "test@example.com", "--color", "never", "--units", "metric"}, mode...)
	code = execute(append(global, args...), "test", "none", "never", &out, &errOut)
	return out.String(), errOut.String(), code
}

func (h *writeHarness) writes() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.sent)
}

func TestWriteCensus(t *testing.T) {
	fset := token.NewFileSet()
	apiWrites := map[string]bool{}
	methods := map[string]*ast.FuncDecl{}
	for _, fn := range parseFuncs(t, fset, "../garminapi") {
		if fn.Recv != nil && types.ExprString(fn.Recv.List[0].Type) == "*Client" {
			methods[fn.Name.Name] = fn
		}
	}
	for grew := true; grew; {
		grew = false
		for name, fn := range methods {
			ids := idents(fn.Body)
			if !apiWrites[name] && (ids["MethodPost"] || ids["MethodPut"] || ids["MethodPatch"] || ids["MethodDelete"] || hasAny(calls(fn.Body), apiWrites)) {
				apiWrites[name], grew = true, true
			}
		}
	}
	for name := range apiWrites {
		fn := methods[name]
		if res := fn.Type.Results; ast.IsExported(name) && (res == nil || len(res.List) != 2 ||
			types.ExprString(res.List[0].Type) != "json.RawMessage" || types.ExprString(res.List[1].Type) != "error") {
			t.Errorf("%s: Client.%s sends a write: return (json.RawMessage, error) so the command can print the response", fset.Position(fn.Pos()), name)
		}
	}

	paths := map[string][]string{}
	commandPaths(newTestParser(t).Model.Node, "", paths)
	banned := []string{"IsJSON", "IsPlain", "ModeFromContext", "WriteJSON", "WriteTable", "WritePlain", "writeHealthJSON", "Stdout", "Successf"}
	found := map[string]bool{}
	for _, fn := range parseFuncs(t, fset, ".") {
		if fn.Name.Name != "Run" || fn.Recv == nil || !hasAny(calls(fn.Body), apiWrites) {
			continue
		}
		pos := fset.Position(fn.Pos())
		recv := strings.TrimPrefix(types.ExprString(fn.Recv.List[0].Type), "*")
		if len(paths[recv]) == 0 {
			t.Errorf("%s: %s.Run sends a write but is not a command in the CLI tree", pos, recv)
		}
		ids, called := idents(fn.Body), calls(fn.Body)
		for _, path := range paths[recv] {
			found[path] = true
			if _, ok := writeCases[path]; !ok {
				t.Errorf("%s: %q sends a write to Garmin: add a row to writeCases in write_test.go", pos, path)
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
		for _, d := range f.Decls {
			if fn, ok := d.(*ast.FuncDecl); ok && fn.Body != nil {
				fns = append(fns, fn)
			}
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
			}
		}
		return true
	})
	return out
}

func hasAny(names, set map[string]bool) bool {
	for name := range names {
		if set[name] {
			return true
		}
	}
	return false
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
