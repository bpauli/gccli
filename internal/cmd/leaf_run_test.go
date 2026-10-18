package cmd

import (
	"bytes"
	"maps"
	"net/http"
	"slices"
	"strings"
	"testing"
)

func runTree(t *testing.T, nodes []Node, in harnessInput, args ...string) (stdout, stderr string, code int, requests []string) {
	t.Helper()
	h := newHarness(t, in)
	cli := CLI{Plugins: attach(nodes)}
	var out, errOut bytes.Buffer
	parser := newParserFor(t, &cli, &out, &errOut)
	code = run(parser, &cli, append([]string{"--account", harnessAccount, "--color", "never", "--units", "metric"}, args...), &out, &errOut)
	for _, r := range h.requests() {
		requests = append(requests, r.route)
	}
	return out.String(), errOut.String(), code, requests
}

const (
	profileRoute    = "GET /userprofile-service/userprofile/settings"
	profileBody     = `{"displayName":"tester","profileId":3141592}`
	socialRoute     = "GET /userprofile-service/socialProfile/tester"
	settingsRoute   = "GET /userprofile-service/userprofile/user-settings"
	activitiesRoute = "GET /activitylist-service/activities/search/activities?start=0&limit=20"
	activitiesBody  = `[{"activityId":111,"activityName":"Morning Run","startTimeLocal":"2026-10-14 07:30:00","activityType":{"typeKey":"running"},"distance":10012.5,"duration":3005.2,"calories":712},{"activityId":112,"activityName":"Pool Swim","startTimeLocal":"2026-10-13 18:00:00","activityType":{"typeKey":"lap_swimming"},"distance":1500,"duration":1980,"calories":350}]`
	coursesRoute    = "GET /web-gateway/course/owner/"
	value1          = "{\n  \"value\": 1\n}\n"
)

func TestLeafRun(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		stdin    string
		routes   map[string]mockReply
		stdout   string
		stderr   string
		code     int
		requests []string
	}{
		{
			name:     "sends the input and prints the body",
			args:     []string{"goals", "list", "--status", "past"},
			routes:   map[string]mockReply{"GET /goal-service/goal/goals?status=past": {body: `{"value":1}`}},
			stdout:   value1,
			requests: []string{"GET /goal-service/goal/goals?status=past"},
		},
		{
			name:     "prints {} on an empty body",
			args:     []string{"goals", "list"},
			routes:   map[string]mockReply{"GET /goal-service/goal/goals?status=active": {status: http.StatusNoContent}},
			stdout:   "{}\n",
			requests: []string{"GET /goal-service/goal/goals?status=active"},
		},
		{
			name:     "wraps a request error with the verb",
			args:     []string{"--json", "goals", "list"},
			routes:   map[string]mockReply{"GET /goal-service/goal/goals?status=active": {status: http.StatusInternalServerError, body: "down"}},
			stderr:   `{"error":{"code":"error","status":500,"message":"get goals: garmin api: 500 Internal Server Error: down"}}` + "\n",
			code:     1,
			requests: []string{"GET /goal-service/goal/goals?status=active"},
		},
		{
			name:   "checks a late date after the login and before any request",
			args:   []string{"health", "sleep", "2026-02-30"},
			stderr: "invalid date \"2026-02-30\": expected YYYY-MM-DD, today, yesterday, Nd (e.g. 3d), or +Nd (e.g. +30d)\n",
			code:   1,
		},
		{
			name: "resolves a late date and looks the display name up once",
			args: []string{"health", "sleep", "yesterday"},
			routes: map[string]mockReply{
				profileRoute: {body: profileBody},
				"GET /wellness-service/wellness/dailySleepData/tester?date=2026-10-14&nonSleepBufferMinutes=60": {body: `{"value":1}`},
			},
			stdout:   value1,
			requests: []string{profileRoute, "GET /wellness-service/wellness/dailySleepData/tester?date=2026-10-14&nonSleepBufferMinutes=60"},
		},
		{
			name:     "a failed lookup stops the command",
			args:     []string{"health", "sleep"},
			routes:   map[string]mockReply{profileRoute: {status: http.StatusInternalServerError, body: "down"}},
			stderr:   "resolve display name: garmin api: 500 Internal Server Error: down\n",
			code:     1,
			requests: []string{profileRoute},
		},
		{
			name: "resolves the profile PK",
			args: []string{"gear"},
			routes: map[string]mockReply{
				profileRoute: {body: profileBody},
				socialRoute:  {body: profileBody},
				"GET /gear-service/gear/filterGear?userProfilePk=3141592": {body: `{"value":1}`},
			},
			stdout:   value1,
			requests: []string{profileRoute, socialRoute, "GET /gear-service/gear/filterGear?userProfilePk=3141592"},
		},
		{
			name:     "Or falls back to Today",
			args:     []string{"devices", "solar", "7"},
			routes:   map[string]mockReply{"GET /web-gateway/solar/7/2026-10-15/2026-10-15": {body: `{"value":1}`}},
			stdout:   value1,
			requests: []string{"GET /web-gateway/solar/7/2026-10-15/2026-10-15"},
		},
		{
			name:     "Or falls back to another input",
			args:     []string{"devices", "solar", "7", "--start", "2026-01-02"},
			routes:   map[string]mockReply{"GET /web-gateway/solar/7/2026-01-02/2026-01-02": {body: `{"value":1}`}},
			stdout:   value1,
			requests: []string{"GET /web-gateway/solar/7/2026-01-02/2026-01-02"},
		},
		{
			name:     "Or keeps a given input",
			args:     []string{"devices", "solar", "7", "--start", "2026-01-02", "--end", "2026-01-05"},
			routes:   map[string]mockReply{"GET /web-gateway/solar/7/2026-01-02/2026-01-05": {body: `{"value":1}`}},
			stdout:   value1,
			requests: []string{"GET /web-gateway/solar/7/2026-01-02/2026-01-05"},
		},
		{
			name:     "a write asks, sends and reports Done",
			args:     []string{"courses", "delete", "42"},
			stdin:    "y\n",
			routes:   map[string]mockReply{"DELETE /course-service/course/42": {body: `{"deleted":true}`}},
			stderr:   "Delete course 42? [y/N]: Deleted course 42\n",
			requests: []string{"DELETE /course-service/course/42"},
		},
		{
			name:     "a write prints the body in json mode",
			args:     []string{"--json", "courses", "delete", "42", "--force"},
			routes:   map[string]mockReply{"DELETE /course-service/course/42": {body: `{"deleted":true}`}},
			stdout:   "{\n  \"deleted\": true\n}\n",
			requests: []string{"DELETE /course-service/course/42"},
		},
		{
			name:     "a value with braces is not expanded again",
			args:     []string{"courses", "delete", "{ID}", "--force"},
			routes:   map[string]mockReply{"DELETE /course-service/course/%7BID%7D": {body: `{"deleted":true}`}},
			stderr:   "Deleted course {ID}\n",
			requests: []string{"DELETE /course-service/course/%7BID%7D"},
		},
		{
			name:   "a declined write sends nothing",
			args:   []string{"courses", "delete", "42"},
			stdin:  "n\n",
			stderr: "Delete course 42? [y/N]: Cancelled\n",
		},
		{
			name:     "prints columns in table mode",
			args:     []string{"activities"},
			routes:   map[string]mockReply{activitiesRoute: {body: activitiesBody}},
			stdout:   "ID   DATE        TYPE          NAME         DISTANCE  DURATION  CALORIES\n111  2026-10-14  running       Morning Run  10.01 km  50:05     712\n112  2026-10-13  lap_swimming  Pool Swim    1.50 km   33:00     350\n",
			requests: []string{activitiesRoute},
		},
		{
			name:     "prints the header first with --plain --header",
			args:     []string{"--plain", "--header", "activities", "list"},
			routes:   map[string]mockReply{activitiesRoute: {body: activitiesBody}},
			stdout:   "ID\tDATE\tTYPE\tNAME\tDISTANCE\tDURATION\tCALORIES\n111\t2026-10-14\trunning\tMorning Run\t10.01 km\t50:05\t712\n112\t2026-10-13\tlap_swimming\tPool Swim\t1.50 km\t33:00\t350\n",
			requests: []string{activitiesRoute},
		},
		{
			name:     "prints null on an empty body in json mode",
			args:     []string{"--json", "activities"},
			routes:   map[string]mockReply{activitiesRoute: {status: http.StatusNoContent}},
			stdout:   "null\n",
			requests: []string{activitiesRoute},
		},
		{
			name:     "an empty body fails in table mode before the units lookup",
			args:     []string{"--units", "auto", "activities"},
			routes:   map[string]mockReply{activitiesRoute: {status: http.StatusNoContent}},
			stderr:   "parse activities: unexpected end of JSON input\n",
			code:     1,
			requests: []string{activitiesRoute},
		},
		{
			name:     "looks the units up after the list",
			args:     []string{"--units", "auto", "activities"},
			routes:   map[string]mockReply{activitiesRoute: {body: activitiesBody}, settingsRoute: {body: `{"userData":{"measurementSystem":"statute_us"}}`}},
			stdout:   "ID   DATE        TYPE          NAME         DISTANCE  DURATION  CALORIES\n111  2026-10-14  running       Morning Run  6.22 mi   50:05     712\n112  2026-10-13  lap_swimming  Pool Swim    0.93 mi   33:00     350\n",
			requests: []string{activitiesRoute, settingsRoute},
		},
		{
			name:     "skips the units lookup when no column needs them",
			args:     []string{"--units", "auto", "courses"},
			routes:   map[string]mockReply{coursesRoute: {body: `{"coursesForUser":[{"courseId":111,"courseName":"Morning Loop","createdDate":"2026-09-01 10:00:00"}]}`}},
			stdout:   "ID   NAME          CREATED\n111  Morning Loop  2026-09-01\n",
			requests: []string{coursesRoute},
		},
		{
			name:     "a missing wrapper key gives no rows",
			args:     []string{"courses"},
			routes:   map[string]mockReply{coursesRoute: {body: `{"courses":[{"courseId":111}]}`}},
			stdout:   "ID  NAME  CREATED\n",
			requests: []string{coursesRoute},
		},
		{
			name:     "a wrapper element that is not an object is skipped",
			args:     []string{"--plain", "courses"},
			routes:   map[string]mockReply{coursesRoute: {body: `{"coursesForUser":[5,{"courseId":111,"courseName":"Morning Loop"}]}`}},
			stdout:   "111\tMorning Loop\t\n",
			requests: []string{coursesRoute},
		},
		{
			name:     "a root element that is not an object is an error",
			args:     []string{"activities"},
			routes:   map[string]mockReply{activitiesRoute: {body: `[5]`}},
			stderr:   "parse activities: json: cannot unmarshal number into Go value of type map[string]interface {}\n",
			code:     1,
			requests: []string{activitiesRoute},
		},
		{
			name:   "a declined write with a lookup sends nothing",
			args:   []string{"gear", "unlink", "abc"},
			stdin:  "n\n",
			stderr: "Unlink gear abc? [y/N]: Cancelled\n",
		},
		{
			name:     "a write looks up only after the prompt",
			args:     []string{"gear", "unlink", "abc"},
			stdin:    "y\n",
			routes:   map[string]mockReply{profileRoute: {body: profileBody}, "PUT /gear-service/gear/unlink/abc/activity/tester": {body: `{"ok":1}`}},
			stderr:   "Unlink gear abc? [y/N]: Unlinked gear abc\n",
			requests: []string{profileRoute, "PUT /gear-service/gear/unlink/abc/activity/tester"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, code, requests := runTree(t, fixtureTree(), harnessInput{routes: tc.routes, stdin: tc.stdin}, tc.args...)
			if code != tc.code || stdout != tc.stdout || stderr != tc.stderr {
				t.Errorf("exit %d, want %d\nstdout %q\nwant   %q\nstderr %q\nwant   %q", code, tc.code, stdout, tc.stdout, stderr, tc.stderr)
			}
			if got, want := strings.Join(requests, "\n"), strings.Join(tc.requests, "\n"); got != want {
				t.Errorf("requests:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

var lookupRoutes = map[lookup]struct{ route, errPrefix string }{
	lookupDisplayName: {profileRoute, "resolve display name: "},
	lookupProfilePK:   {profileRoute, "get user profile: "},
}

func TestLeafRowSadPaths(t *testing.T) {
	walkNodes(commands, "", func(path string, n Node) {
		l, ok := n.(Leaf)
		if !ok {
			return
		}
		args, in, err := goldenInput(goldenFile(path))
		if err != nil {
			t.Errorf("%s: %v", path, err)
			return
		}
		var lookups []lookup
		for _, s := range l.Params {
			for _, a := range s.alts {
				if _, ok := lookupRoutes[a.lookup]; ok && !slices.Contains(lookups, a.lookup) {
					lookups = append(lookups, a.lookup)
				}
			}
		}
		var dataRoutes []string
		for route := range in.routes {
			if route != profileRoute && route != socialRoute && route != settingsRoute {
				dataRoutes = append(dataRoutes, route)
			}
		}
		if len(dataRoutes) != 1 {
			t.Errorf("%s: golden has %d data routes, want 1", path, len(dataRoutes))
			return
		}
		data := dataRoutes[0]
		exec := func(t *testing.T, routes map[string]mockReply, flags, extra []string) (stdout, stderr string, code int, requests []string) {
			h := newHarness(t, harnessInput{files: in.files, routes: routes, stdin: in.stdin})
			stdout, stderr, code = h.run(flags, append(slices.Clone(args), extra...))
			for _, r := range h.requests() {
				requests = append(requests, r.route)
			}
			return stdout, stderr, code, requests
		}
		run := func(name string, routes map[string]mockReply, flag string, check func(t *testing.T, stdout, stderr string, code int, requests []string)) {
			t.Run(path+"/"+name+"/"+strings.TrimPrefix(flag, "--"), func(t *testing.T) {
				stdout, stderr, code, requests := exec(t, routes, []string{flag}, nil)
				check(t, stdout, stderr, code, requests)
			})
		}
		for _, flag := range []string{"--plain", "--json"} {
			failed := maps.Clone(in.routes)
			failed[data] = mockReply{status: http.StatusInternalServerError, body: "down"}
			run("request fails", failed, flag, func(t *testing.T, stdout, stderr string, code int, requests []string) {
				want := l.Verb + ": garmin api: 500 Internal Server Error: down\n"
				if flag == "--json" {
					want = `{"error":{"code":"error","status":500,"message":"` + l.Verb + `: garmin api: 500 Internal Server Error: down"}}` + "\n"
				}
				if code != 1 || stdout != "" || stderr != want {
					t.Errorf("exit %d, want 1\nstdout %q, want empty\nstderr %q\nwant   %q", code, stdout, stderr, want)
				}
				if len(requests) != len(lookups)+1 || requests[len(requests)-1] != data {
					t.Errorf("requests %q, want the lookups then %q", requests, data)
				}
			})
			for _, k := range lookups {
				failed := maps.Clone(in.routes)
				failed[lookupRoutes[k].route] = mockReply{status: http.StatusInternalServerError, body: "down"}
				run("lookup fails", failed, flag, func(t *testing.T, stdout, stderr string, code int, requests []string) {
					want := lookupRoutes[k].errPrefix + "garmin api: 500 Internal Server Error: down\n"
					if flag == "--json" {
						want = `{"error":{"code":"error","status":500,"message":"` + strings.TrimSuffix(want, "\n") + `"}}` + "\n"
					}
					if code != 1 || stdout != "" || stderr != want {
						t.Errorf("exit %d, want 1\nstdout %q, want empty\nstderr %q\nwant   %q", code, stdout, stderr, want)
					}
					if len(requests) != 1 || requests[0] != lookupRoutes[k].route {
						t.Errorf("requests %q, want only %q", requests, lookupRoutes[k].route)
					}
				})
			}
			empty := maps.Clone(in.routes)
			empty[data] = mockReply{status: http.StatusNoContent}
			switch out := l.Output.(type) {
			case JSON:
				run("empty body", empty, flag, func(t *testing.T, stdout, stderr string, code int, _ []string) {
					if code != 0 || stdout != "{}\n" || stderr != "" {
						t.Errorf("exit %d, want 0\nstdout %q, want {}\nstderr %q, want empty", code, stdout, stderr)
					}
				})
			case Rows:
				run("empty body", empty, flag, func(t *testing.T, stdout, stderr string, code int, _ []string) {
					wantOut, wantErr, wantCode := "", "parse "+out.Noun+": unexpected end of JSON input\n", 1
					if flag == "--json" {
						wantOut, wantErr, wantCode = "null\n", "", 0
					}
					if code != wantCode || stdout != wantOut || stderr != wantErr {
						t.Errorf("exit %d, want %d\nstdout %q\nwant   %q\nstderr %q\nwant   %q", code, wantCode, stdout, wantOut, stderr, wantErr)
					}
				})
				if flag == "--json" || !slices.ContainsFunc(out.Columns, func(c Column) bool { return c.Format == Distance }) {
					continue
				}
				t.Run(path+"/units lookup fails/"+strings.TrimPrefix(flag, "--"), func(t *testing.T) {
					metric, _, _, _ := exec(t, in.routes, []string{flag}, []string{"--units", "metric"})
					failed := maps.Clone(in.routes)
					failed[settingsRoute] = mockReply{status: http.StatusInternalServerError, body: "down"}
					stdout, stderr, code, requests := exec(t, failed, []string{flag}, []string{"--units", "auto"})
					want := "could not read Garmin unit preference (garmin api: 500 Internal Server Error: down); using metric. Set --units to skip this lookup.\n"
					if code != 0 || stdout != metric || stderr != want {
						t.Errorf("exit %d, want 0\nstdout %q\nwant   %q\nstderr %q\nwant   %q", code, stdout, metric, stderr, want)
					}
					if len(requests) != len(lookups)+2 || requests[len(requests)-2] != data || requests[len(requests)-1] != settingsRoute {
						t.Errorf("requests %q, want the lookups, then %q, then %q", requests, data, settingsRoute)
					}
				})
			}
		}
	})
}
