package conformance

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
)

type shape interface {
	kind() string
	check(got any, at string) []string
}

type scalar string

const (
	text   scalar = "string"
	number scalar = "number"
)

type arrayOf struct{ elem shape }

type object map[string]shape

func (s scalar) kind() string { return string(s) }
func (arrayOf) kind() string  { return "array" }
func (object) kind() string   { return "object" }

func jsonKind(v any) string {
	switch v.(type) {
	case string:
		return "string"
	case json.Number:
		return "number"
	case bool:
		return "bool"
	case nil:
		return "null"
	case map[string]any:
		return "object"
	}
	return "array"
}

func joinPath(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

func kindProblem(want shape, got any, at string) []string {
	if gk := jsonKind(got); gk != want.kind() {
		return []string{fmt.Sprintf("%s: want %s, the recording has %s", cmp.Or(at, "(body)"), want.kind(), gk)}
	}
	return nil
}

func (s scalar) check(got any, at string) []string { return kindProblem(s, got, at) }

func (a arrayOf) check(got any, at string) []string {
	if p := kindProblem(a, got, at); p != nil || a.elem == nil {
		return p
	}
	var problems []string
	for _, x := range got.([]any) {
		if x != nil {
			problems = append(problems, a.elem.check(x, at+"[]")...)
		}
	}
	slices.Sort(problems)
	return slices.Compact(problems)
}

func (o object) check(got any, at string) []string {
	if p := kindProblem(o, got, at); p != nil {
		return p
	}
	g := got.(map[string]any)
	var problems []string
	for _, key := range slices.Sorted(maps.Keys(o)) {
		x, ok := g[key]
		switch {
		case !ok:
			problems = append(problems, joinPath(at, key)+": missing")
		case x != nil:
			problems = append(problems, o[key].check(x, joinPath(at, key))...)
		}
	}
	return problems
}

var readShapes = []struct {
	route string
	shape shape
}{
	{"GET /activitylist-service/activities/count", object{"totalCount": number}},
	{"GET /activitylist-service/activities/search/activities", arrayOf{object{
		"activityId": number, "activityName": text, "startTimeLocal": text,
		"activityType": object{"typeKey": text}, "distance": number, "duration": number,
	}}},
	{"GET /activity-service/activity/{n}", object{
		"activityId": number, "activityName": text, "activityTypeDTO": object{"typeKey": text},
		"summaryDTO": object{"startTimeLocal": text, "distance": number, "duration": number},
	}},
	{"GET /activity-service/activity/{n}/exerciseSets", object{"exerciseSets": arrayOf{object{"setType": text, "exercises": arrayOf{}}}}},
	{"GET /activity-service/activity/{n}/hrTimeInZones", arrayOf{}},
	{"GET /activity-service/activity/{n}/splits", object{"lapDTOs": arrayOf{}}},
	{"GET /calendar-service/events", arrayOf{object{"id": number, "eventName": text, "date": text}}},
	{"GET /calendar-service/year/{n}/month/{n}/day/{n}/start/{n}", object{"calendarItems": arrayOf{object{"itemType": text, "date": text}}}},
	{"GET /course-service/course/favorites", arrayOf{object{"courseId": number, "courseName": text}}},
	{"GET /course-service/course/{n}", object{"courseId": number, "courseName": text}},
	{"GET /device-service/deviceregistration/devices", arrayOf{object{"deviceId": number}}},
	{"GET /goal-service/goal/goals", arrayOf{}},
	{"GET /userprofile-service/userprofile/settings", object{"displayName": text}},
	{"GET /userprofile-service/userprofile/user-settings", object{"userData": object{"measurementSystem": text}}},
	{"GET /userstats-service/wellness/daily/tester", object{}},
	{"GET /usersummary-service/usersummary/daily/tester", object{}},
	{"GET /web-gateway/course/owner/", object{"coursesForUser": arrayOf{object{"courseId": number, "courseName": text}}}},
	{"GET /wellness-service/wellness/dailyHeartRate/tester", object{}},
	{"GET /wellness-service/wellness/dailySleepData/tester", object{}},
	{"GET /workout-service/workout/{n}", object{"workoutId": number, "workoutName": text}},
	{"GET /workout-service/workouts", arrayOf{object{"workoutId": number, "workoutName": text}}},
	{"POST /activity-service/activity", object{"activityId": number}},
	{"POST /calendar-service/event", object{"id": number}},
	{"POST /course-service/course", object{"courseId": number}},
	{"POST /course-service/course/elevation", arrayOf{arrayOf{}}},
	{"POST /course-service/course/import", object{"geoPoints": arrayOf{object{}}}},
	{"POST /workout-service/workout", object{"workoutId": number}},
}

func replayKey(method, host, uri string) string {
	path, _, _ := strings.Cut(uri, "?")
	segs := strings.Split(path, "/")
	for i, s := range segs {
		switch {
		case digitsOnly.MatchString(s):
			segs[i] = "{n}"
		case dateOnly.MatchString(s):
			segs[i] = "{date}"
		}
	}
	return method + " " + host + strings.Join(segs, "/")
}

func routeKey(route string) string {
	method, path, _ := strings.Cut(route, " ")
	return replayKey(method, routeHost(path), path)
}

func goldenRouteKey(route string) string {
	method, target, _ := strings.Cut(route, " ")
	host, uri, _ := strings.Cut(target, "/")
	return replayKey(method, host, "/"+uri)
}

func succeeded(files []recordedFile) []recordedFile {
	var ok []recordedFile
	for _, f := range files {
		if f.Status >= 200 && f.Status < 300 {
			ok = append(ok, f)
		}
	}
	return ok
}

func checkShape(want shape, body json.RawMessage) ([]string, error) {
	if body == nil {
		return []string{"(body): want " + want.kind() + ", the recording has no JSON body"}, nil
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var got any
	if err := dec.Decode(&got); err != nil {
		return nil, err
	}
	return want.check(got, ""), nil
}

func TestCheckShape(t *testing.T) {
	activities := arrayOf{object{"activityId": number, "activityType": object{"typeKey": text}}}
	for _, tc := range []struct {
		name, body string
		want       []string
	}{
		{"match", `[{"activityId": 1, "activityType": {"typeKey": "running"}, "extra": "x"}]`, nil},
		{"null values pass", `[{"activityId": null, "activityType": null}, null]`, nil},
		{"empty array", `[]`, nil},
		{"object for array", `{"activities": []}`, []string{"(body): want array, the recording has object"}},
		{"null body", `null`, []string{"(body): want array, the recording has null"}},
		{"missing key", `[{"activityType": {"typeKey": "running"}}]`, []string{"[].activityId: missing"}},
		{"nested kind", `[{"activityId": 1, "activityType": {"typeKey": 3}}, {"activityId": "1", "activityType": {"typeKey": "x"}}]`, []string{
			"[].activityId: want number, the recording has string",
			"[].activityType.typeKey: want string, the recording has number",
		}},
	} {
		got, err := checkShape(activities, json.RawMessage(tc.body))
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(got, tc.want) {
			t.Errorf("%s: checkShape = %q, want %q", tc.name, got, tc.want)
		}
	}
	if got, _ := checkShape(object{}, nil); len(got) != 1 {
		t.Errorf("checkShape of a record without a JSON body = %q, want one problem", got)
	}
}

func TestReplayKey(t *testing.T) {
	same := [][2]string{
		{"/activity-service/activity/111/splits", "/activity-service/activity/1/splits"},
		{"/calendar-service/events?startDate=2026-10-15&limit=20", "/calendar-service/events?startDate=2026-01-01&limit=1"},
		{"/usersummary-service/stats/steps/daily/2026-10-01/2026-10-01", "/usersummary-service/stats/steps/daily/2026-01-01/2026-01-01"},
	}
	for _, p := range same {
		if a, b := replayKey("GET", "connectapi.garmin.com", p[0]), replayKey("GET", "connectapi.garmin.com", p[1]); a != b {
			t.Errorf("replayKey(%s) = %s, replayKey(%s) = %s, want the same key", p[0], a, p[1], b)
		}
	}
	for _, p := range [][2]string{
		{"/course-service/course/111", "/course-service/course/favorites"},
		{"/userprofile-service/socialProfile/tester", "/userprofile-service/socialProfile/other"},
	} {
		if replayKey("GET", "connectapi.garmin.com", p[0]) == replayKey("GET", "connectapi.garmin.com", p[1]) {
			t.Errorf("replayKey gives %s and %s the same key", p[0], p[1])
		}
	}
	if replayKey("GET", "connectapi.garmin.com", "/x") == replayKey("POST", "connectapi.garmin.com", "/x") ||
		replayKey("GET", "connectapi.garmin.com", "/x") == replayKey("GET", "connect.garmin.com", "/x") {
		t.Error("replayKey ignores the method or the host")
	}
}

func recordedByKey(t *testing.T, dir string) map[string][]recordedFile {
	t.Helper()
	m := map[string][]recordedFile{}
	for _, f := range recordedFiles(t, dir) {
		k := replayKey(f.Method, f.Host, f.URI)
		m[k] = append(m[k], f)
	}
	return m
}

func goldenReaders(t *testing.T) map[string][]string {
	t.Helper()
	readers := map[string][]string{}
	for _, f := range goldenFiles(t) {
		for route := range f.routes {
			k := goldenRouteKey(route)
			if !slices.Contains(readers[k], f.name) {
				readers[k] = append(readers[k], f.name)
			}
		}
	}
	for _, names := range readers {
		slices.Sort(names)
	}
	return readers
}

func shapeProblems(recorded map[string][]recordedFile, readers map[string][]string) ([]string, error) {
	var problems []string
	for _, s := range readShapes {
		key := routeKey(s.route)
		if len(readers[key]) == 0 {
			problems = append(problems, fmt.Sprintf("readShapes names %s, which no golden case in %s calls", s.route, goldenDir))
			continue
		}
		cmds := strings.Join(readers[key], ", ")
		files := recorded[key]
		ok := succeeded(files)
		switch {
		case len(files) == 0:
			problems = append(problems, fmt.Sprintf("%s: %s has no %s", cmds, recordedDir(), s.route))
		case len(ok) == 0:
			var statuses []string
			for _, f := range files {
				statuses = append(statuses, fmt.Sprintf("%d in %s", f.Status, f.name))
			}
			problems = append(problems, fmt.Sprintf("%s: %s answered %s, want 2xx", cmds, s.route, strings.Join(statuses, ", ")))
		}
		for _, f := range ok {
			found, err := checkShape(s.shape, f.Body)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", f.name, err)
			}
			for _, p := range found {
				problems = append(problems, fmt.Sprintf("%s: %s in %s: %s", cmds, s.route, f.name, p))
			}
		}
	}
	return problems, nil
}

func TestRecordedShapes(t *testing.T) {
	problems, err := shapeProblems(recordedByKey(t, recordedDir()), goldenReaders(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range problems {
		t.Error(p)
	}
}

func TestShapeProblemsNameTheCommand(t *testing.T) {
	goals := routeKey("GET /goal-service/goal/goals")
	for _, tc := range []struct {
		name string
		rec  Record
		want string
	}{
		{"object for array", Record{Status: 200, Body: json.RawMessage(`{"goals": []}`)}, "goals list: GET /goal-service/goal/goals in goals.json: (body): want array, the recording has object"},
		{"bad request", Record{Status: 400, Body: json.RawMessage(`{"message": "status cant be null"}`)}, "goals list: GET /goal-service/goal/goals answered 400 in goals.json, want 2xx"},
	} {
		recorded := recordedByKey(t, "testdata/recorded")
		recorded[goals] = []recordedFile{{"goals.json", tc.rec}}
		problems, err := shapeProblems(recorded, goldenReaders(t))
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(problems, []string{tc.want}) {
			t.Errorf("%s: shapeProblems = %q, want %q", tc.name, problems, tc.want)
		}
	}
}

type recordedReplay struct {
	served map[string]Record

	mu         sync.Mutex
	unexpected []string
}

func (r *recordedReplay) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	key := replayKey(req.Method, req.Host, req.URL.RequestURI())
	rec, ok := r.served[key]
	if !ok {
		r.mu.Lock()
		r.unexpected = append(r.unexpected, key)
		r.mu.Unlock()
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if rec.Body != nil {
		w.Header().Set("Content-Type", "application/json")
	}
	w.WriteHeader(rec.Status)
	_, _ = w.Write(rec.Body)
}

func TestRecordedReplay(t *testing.T) {
	bin := binary(t)
	served := map[string]Record{}
	for key, files := range recordedByKey(t, recordedDir()) {
		for _, f := range succeeded(files) {
			if f.NonJSONBytes == 0 {
				served[key] = f.Record
				break
			}
		}
	}
	token, err := os.ReadFile("testdata/state/token_golden.json")
	if err != nil {
		t.Fatal(err)
	}
	seed := newSandbox(t, bin, http.NotFoundHandler())
	seed.importToken(t, token)

	ran := 0
	for _, f := range goldenFiles(t) {
		if _, skip := skipped[f.name]; skip || len(f.routes) == 0 || f.outputs["table exit"] != "0\n" {
			continue
		}
		covered := true
		for route := range f.routes {
			if _, ok := served[goldenRouteKey(route)]; !ok {
				covered = false
			}
		}
		if !covered {
			continue
		}
		ran++
		t.Run(f.name, func(t *testing.T) {
			t.Parallel()
			r := &recordedReplay{served: served}
			s := newSandbox(t, bin, r)
			if err := os.CopyFS(s.home, os.DirFS(seed.home)); err != nil {
				t.Fatal(err)
			}
			for name, content := range f.files {
				if err := os.WriteFile(filepath.Join(s.work, name), []byte(content), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			res := s.run(t, f.stdin, slices.Concat(goldenGlobals, f.args)...)
			if res.code != 0 {
				t.Errorf("%q exits %d on the responses in %s, want 0\nstderr: %s", f.name, res.code, recordedDir(), res.stderr)
			}
			for _, u := range r.unexpectedRequests() {
				t.Errorf("%q sent %s, which %s has no 2xx recording for", f.name, u, recordedDir())
			}
		})
	}
	if ran == 0 {
		t.Fatalf("no golden case has a 2xx recording in %s for each of its routes", recordedDir())
	}
}

func (r *recordedReplay) unexpectedRequests() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.unexpected)
}
