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

// A shape names what the commands that call a route read from its
// response. The check fails when a field they read is missing or has
// another JSON type. A null passes anywhere, because Garmin sends null for
// a value it does not have.
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

// A passthrough route is one whose commands print the body as it is in
// every mode and read nothing beneath the top level. It names the top
// level kind, object or array.
type passthrough string

// The unread shape marks a route whose response the commands never read,
// such as a DELETE that answers 204. The download shape marks a response
// that is not JSON: the commands write the bytes to a file.
type special string

const (
	unread   special = "unread"
	download special = "download"
)

func (s scalar) kind() string      { return string(s) }
func (arrayOf) kind() string       { return "array" }
func (object) kind() string        { return "object" }
func (p passthrough) kind() string { return string(p) }
func (s special) kind() string     { return string(s) }

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

func (s scalar) check(got any, at string) []string      { return kindProblem(s, got, at) }
func (p passthrough) check(got any, at string) []string { return kindProblem(p, got, at) }
func (special) check(any, string) []string              { return nil }

func (a arrayOf) check(got any, at string) []string {
	if p := kindProblem(a, got, at); p != nil || a.elem == nil {
		return p
	}
	if len(got.([]any)) == 0 {
		return []string{fmt.Sprintf("%s: want at least one element to check its shape, the recording has none", cmp.Or(at, "(body)"))}
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

type routeShape struct {
	route string
	shape shape
}

var (
	settingsName = routeShape{"GET /userprofile-service/userprofile/settings", object{"displayName": text}}
	activityRows = arrayOf{object{
		"activityId": number, "activityName": text, "startTimeLocal": text, "activityType": object{"typeKey": text},
		"distance": number, "duration": number, "calories": number,
	}}
	courseRows = arrayOf{object{"courseId": number, "courseName": text, "courseType": object{"typeKey": text}, "distanceInMeters": number, "createdDate": text}}
)

// caseReads names, for each golden case the replay runs, what its command
// reads from each route in table and plain mode: the fields, or
// passthrough, unread or download. A field read through a helper, such as
// displayName through resolveDisplayName, is listed too. The fields come
// from the command code in internal/cmd. A field that a command reads only
// when present, and that the golden mock does not send, is not listed:
// activity summary reads parentTypeId, trainingEffect,
// anaerobicTrainingEffect, directWorkoutFeel and directWorkoutRpe when
// present, workouts list reads owner.displayName, and workouts schedule
// list reads workoutId and sportTypeKey of workout items only.
var caseReads = map[string][]routeShape{
	"activities count":            {{"GET /activitylist-service/activities/count", object{"totalCount": number}}},
	"activities list":             {{"GET /activitylist-service/activities/search/activities", activityRows}},
	"activities search":           {{"GET /activitylist-service/activities/search/activities", activityRows}},
	"activity create":             {{"POST /activity-service/activity", object{"activityId": number}}},
	"activity delete":             {{"DELETE /activity-service/activity/{n}", unread}},
	"activity describe":           {{"PUT /activity-service/activity/{n}", unread}},
	"activity exercise-sets set":  {{"PUT /activity-service/activity/{n}/exerciseSets", unread}},
	"activity exercise-sets show": {{"GET /activity-service/activity/{n}/exerciseSets", passthrough("object")}},
	"activity favorite":           {{"PUT /activity-service/activity/{n}", unread}},
	"activity hr-zones":           {{"GET /activity-service/activity/{n}/hrTimeInZones", passthrough("array")}},
	"activity privacy":            {{"PUT /activity-service/activity/{n}", unread}},
	"activity rename":             {{"PUT /activity-service/activity/{n}", unread}},
	"activity retype":             {{"PUT /activity-service/activity/{n}", unread}},
	"activity splits":             {{"GET /activity-service/activity/{n}/splits", passthrough("object")}},
	"activity summary": {{"GET /activity-service/activity/{n}", object{
		"activityName": text, "activityTypeDTO": object{"typeKey": text},
		"summaryDTO": object{"startTimeLocal": text, "distance": number, "duration": number, "averageSpeed": number, "averageHR": number, "calories": number},
	}}},
	"activity unfavorite":   {{"PUT /activity-service/activity/{n}", unread}},
	"activity weather":      {{"GET /activity-service/activity/{n}/weather", passthrough("object")}},
	"calendar notes add":    {{"POST /calendar-service/note", object{"id": number}}},
	"calendar notes delete": {{"DELETE /calendar-service/note/{n}", unread}},
	"calendar notes move":   {{"PUT /calendar-service/calendarItem/note/{n}/date/{date}", unread}},
	"calendar notes update": {{"GET /calendar-service/note/{n}", passthrough("object")}, {"PUT /calendar-service/note/{n}", unread}},
	"courses delete":        {{"DELETE /course-service/course/{n}", unread}},
	"courses detail":        {{"GET /course-service/course/{n}", passthrough("object")}},
	"courses favorites":     {{"GET /course-service/course/favorites", courseRows}},
	"courses import": {
		{"POST /course-service/course/import", object{"courseName": text, "geoPoints": arrayOf{object{"latitude": number, "longitude": number}}}},
		{"POST /course-service/course/elevation", arrayOf{arrayOf{number}}},
		{"POST /course-service/course", object{"courseId": number, "courseName": text, "distanceMeter": number, "elevationGainMeter": number, "elevationLossMeter": number}},
	},
	"courses list": {
		{"GET /web-gateway/course/owner/", object{"coursesForUser": courseRows}},
		{"GET /userprofile-service/userprofile/user-settings", object{"userData": object{"measurementSystem": text}}},
	},
	"devices list":             {{"GET /device-service/deviceregistration/devices", passthrough("array")}},
	"events add":               {{"POST /calendar-service/event", object{"id": number, "eventName": text, "date": text}}},
	"events delete":            {{"DELETE /calendar-service/event/{n}", unread}},
	"goals list":               {{"GET /goal-service/goal/goals", passthrough("array")}},
	"health hr view":           {settingsName, {"GET /wellness-service/wellness/dailyHeartRate/tester", passthrough("object")}},
	"health rhr":               {settingsName, {"GET /userstats-service/wellness/daily/tester", passthrough("object")}},
	"health sleep view":        {settingsName, {"GET /wellness-service/wellness/dailySleepData/tester", passthrough("object")}},
	"health steps view":        {settingsName, {"GET /wellness-service/wellness/dailySummaryChart/tester", passthrough("array")}},
	"health summary":           {settingsName, {"GET /usersummary-service/usersummary/daily/tester", passthrough("object")}},
	"profile settings":         {{"GET /userprofile-service/userprofile/user-settings", passthrough("object")}},
	"profile view":             {{"GET /userprofile-service/userprofile/settings", passthrough("object")}},
	"records list":             {settingsName, {"GET /personalrecord-service/personalrecord/prs/tester", passthrough("array")}},
	"workouts copy":            {{"POST /workout-service/workout/copy/{n}", object{"workoutId": number, "workoutName": text}}},
	"workouts create":          {{"POST /workout-service/workout", object{"workoutId": number}}},
	"workouts delete":          {{"DELETE /workout-service/workout/{n}", unread}},
	"workouts detail":          {{"GET /workout-service/workout/{n}", passthrough("object")}},
	"workouts download":        {{"GET /workout-service/workout/FIT/{n}", download}},
	"workouts list":            {{"GET /workout-service/workouts", arrayOf{object{"workoutId": number, "workoutName": text, "sportType": object{"sportTypeKey": text}}}}},
	"workouts schedule add":    {{"POST /workout-service/schedule/{n}", unread}},
	"workouts schedule list":   {{"GET /calendar-service/year/{n}/month/{n}/day/{n}/start/{n}", object{"calendarItems": arrayOf{object{"itemType": text, "date": text, "id": number, "title": text}}}}},
	"workouts schedule remove": {{"DELETE /workout-service/schedule/{n}", unread}},
	"workouts update":          {{"PUT /workout-service/workout/{n}", unread}},
	"workouts upload":          {{"POST /workout-service/workout", unread}},
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

func checkRecord(want shape, r Record) ([]string, error) {
	switch {
	case want == unread:
		return nil, nil
	case want == download:
		if r.NonJSONBytes == 0 {
			return []string{"(body): want bytes that are not JSON, the recording has none"}, nil
		}
		return nil, nil
	case r.Body == nil:
		return []string{"(body): want " + want.kind() + ", the recording has no JSON body"}, nil
	}
	dec := json.NewDecoder(bytes.NewReader(r.Body))
	dec.UseNumber()
	var got any
	if err := dec.Decode(&got); err != nil {
		return nil, err
	}
	return want.check(got, ""), nil
}

func checkShape(want shape, body json.RawMessage) ([]string, error) {
	return checkRecord(want, Record{Body: body})
}

func TestCheckShape(t *testing.T) {
	activities := arrayOf{object{"activityId": number, "activityType": object{"typeKey": text}}}
	for _, tc := range []struct {
		name, body string
		want       []string
	}{
		{"match", `[{"activityId": 1, "activityType": {"typeKey": "running"}, "extra": "x"}]`, nil},
		{"null values pass", `[{"activityId": null, "activityType": null}, null]`, nil},
		{"empty array", `[]`, []string{"(body): want at least one element to check its shape, the recording has none"}},
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
	for _, tc := range []struct {
		name string
		want shape
		rec  Record
		n    int
	}{
		{"an object shape on a record without a body", object{"a": text}, Record{}, 1},
		{"an empty array where no element shape is named", arrayOf{}, Record{Body: json.RawMessage(`[]`)}, 0},
		{"a passthrough on an array body", passthrough("object"), Record{Body: json.RawMessage(`[]`)}, 1},
		{"a passthrough on an object body", passthrough("object"), Record{Body: json.RawMessage(`{}`)}, 0},
		{"unread on a record without a body", unread, Record{}, 0},
		{"download on a JSON body", download, Record{Body: json.RawMessage(`{}`)}, 1},
		{"download on bytes", download, Record{NonJSONBytes: 12}, 0},
	} {
		if got, _ := checkRecord(tc.want, tc.rec); len(got) != tc.n {
			t.Errorf("%s: checkRecord = %q, want %d problems", tc.name, got, tc.n)
		}
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

// servedRecords picks, for each route key, the first 2xx recording with
// a JSON body. The replay serves these, and a case is replayable when
// every route it calls has one.
func servedRecords(recorded map[string][]recordedFile) map[string]Record {
	served := map[string]Record{}
	for key, files := range recorded {
		for _, f := range succeeded(files) {
			if f.NonJSONBytes == 0 {
				served[key] = f.Record
				break
			}
		}
	}
	return served
}

// replayable says whether the recorded replay runs a golden file: a leaf
// case that is not skipped, calls at least one route, exits 0 in table
// mode and has a served recording for every route. A case file
// <leaf>@<case>.golden runs the leaf's command on other mock data, and
// the recording replaces that data, so it is not replayed.
func replayable(f goldenFile, served map[string]Record) bool {
	if _, skip := skipped[f.leaf]; skip || f.name != f.leaf || len(f.routes) == 0 || f.outputs["table exit"] != "0\n" {
		return false
	}
	for route := range f.routes {
		if _, ok := served[goldenRouteKey(route)]; !ok {
			return false
		}
	}
	return true
}

// caseProblems checks each route of a golden case that has a recording: it
// wants an entry in reads and a 2xx recording with the shapes of that
// entry. A route without a recording is not checked, because the e2e
// suite does not call every route. An entry for a route the case does not
// call is a problem too.
func caseProblems(reads []routeShape, c goldenCase, recorded map[string][]recordedFile) ([]string, error) {
	byKey := map[string][]routeShape{}
	for _, r := range reads {
		byKey[routeKey(r.route)] = append(byKey[routeKey(r.route)], r)
	}
	var problems []string
	for _, route := range slices.Sorted(maps.Keys(c.routes)) {
		key := goldenRouteKey(route)
		entries := byKey[key]
		delete(byKey, key)
		files := recorded[key]
		if len(files) == 0 {
			continue
		}
		if len(entries) == 0 {
			problems = append(problems, fmt.Sprintf("%s: caseReads has no entry for this route: name the fields the command reads, or passthrough, unread or download", route))
			continue
		}
		ok := succeeded(files)
		if len(ok) == 0 {
			var statuses []string
			for _, f := range files {
				statuses = append(statuses, fmt.Sprintf("%d in %s", f.Status, f.name))
			}
			problems = append(problems, fmt.Sprintf("%s answered %s, want 2xx", route, strings.Join(statuses, ", ")))
			continue
		}
		for _, f := range ok {
			for _, e := range entries {
				found, err := checkRecord(e.shape, f.Record)
				if err != nil {
					return nil, fmt.Errorf("%s: %w", f.name, err)
				}
				for _, p := range found {
					problems = append(problems, fmt.Sprintf("%s in %s: %s", route, f.name, p))
				}
			}
		}
	}
	for _, key := range slices.Sorted(maps.Keys(byKey)) {
		problems = append(problems, fmt.Sprintf("caseReads names %s, which this case does not call", byKey[key][0].route))
	}
	return problems, nil
}

func TestRecordedShapes(t *testing.T) {
	recorded := recordedByKey(t, recordedDir())
	served := servedRecords(recorded)
	names := map[string]bool{}
	for _, f := range goldenFiles(t) {
		names[f.name] = true
		reads, listed := caseReads[f.name]
		if !listed {
			if replayable(f, served) {
				t.Errorf("caseReads has no entry for %q, which the replay runs: name what the command reads from each route", f.name)
			}
			continue
		}
		t.Run(f.name, func(t *testing.T) {
			problems, err := caseProblems(reads, f.goldenCase, recorded)
			if err != nil {
				t.Fatal(err)
			}
			for _, p := range problems {
				t.Error(p)
			}
		})
	}
	for name := range caseReads {
		if !names[name] {
			t.Errorf("caseReads names %q, which has no golden file in %s", name, goldenDir)
		}
	}
}

func TestCaseProblems(t *testing.T) {
	var goals goldenFile
	for _, f := range goldenFiles(t) {
		if f.name == "goals list" {
			goals = f
		}
	}
	key := routeKey("GET /goal-service/goal/goals")
	route := "GET connectapi.garmin.com/goal-service/goal/goals?status=active"
	for _, tc := range []struct {
		name  string
		reads []routeShape
		rec   Record
		want  []string
	}{
		{"array", caseReads["goals list"], Record{Status: 200, Body: json.RawMessage(`[]`)}, nil},
		{"object for array", caseReads["goals list"], Record{Status: 200, Body: json.RawMessage(`{"goals": []}`)}, []string{route + " in goals.json: (body): want array, the recording has object"}},
		{"bad request", caseReads["goals list"], Record{Status: 400, Body: json.RawMessage(`{"message": "status cant be null"}`)}, []string{route + " answered 400 in goals.json, want 2xx"}},
		{"missing field", []routeShape{{"GET /goal-service/goal/goals", arrayOf{object{"goalType": text}}}}, Record{Status: 200, Body: json.RawMessage(`[{"goalId": 1}]`)}, []string{route + " in goals.json: [].goalType: missing"}},
		{"no entry", nil, Record{Status: 200, Body: json.RawMessage(`[]`)}, []string{route + ": caseReads has no entry for this route: name the fields the command reads, or passthrough, unread or download"}},
		{"entry for a route the case does not call", []routeShape{{"GET /goal-service/goal/goals", passthrough("array")}, {"GET /badge-service/badge/earned", passthrough("array")}}, Record{Status: 200, Body: json.RawMessage(`[]`)}, []string{"caseReads names GET /badge-service/badge/earned, which this case does not call"}},
	} {
		recorded := map[string][]recordedFile{key: {{"goals.json", tc.rec}}}
		problems, err := caseProblems(tc.reads, goals.goldenCase, recorded)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(problems, tc.want) {
			t.Errorf("%s: caseProblems = %q, want %q", tc.name, problems, tc.want)
		}
	}
	if problems, _ := caseProblems(nil, goals.goldenCase, map[string][]recordedFile{}); len(problems) != 0 {
		t.Errorf("caseProblems of a case without a recording = %q, want none", problems)
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
	recorded := recordedByKey(t, recordedDir())
	served := servedRecords(recorded)
	token, err := os.ReadFile("testdata/state/token_golden.json")
	if err != nil {
		t.Fatal(err)
	}
	seed := newSandbox(t, bin, http.NotFoundHandler())
	seed.importToken(t, token)

	ran := 0
	for _, f := range goldenFiles(t) {
		if !replayable(f, served) {
			continue
		}
		ran++
		t.Run(f.name, func(t *testing.T) {
			t.Parallel()
			reads, listed := caseReads[f.name]
			if !listed {
				t.Fatalf("caseReads has no entry for %q: name what the command reads from each route", f.name)
			}
			problems, err := caseProblems(reads, f.goldenCase, recorded)
			if err != nil {
				t.Fatal(err)
			}
			for _, p := range problems {
				t.Error(p)
			}
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
				t.Errorf("%q exits %d on the responses in %s, want 0 (its stderr is not shown, because it can echo a response)", f.name, res.code, recordedDir())
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
