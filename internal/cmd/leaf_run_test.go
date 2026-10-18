package cmd

import (
	"bytes"
	"net/http"
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
	profileRoute = "GET /userprofile-service/userprofile/settings"
	profileBody  = `{"displayName":"tester","profileId":3141592}`
	socialRoute  = "GET /userprofile-service/socialProfile/tester"
	value1       = "{\n  \"value\": 1\n}\n"
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
			name:   "an output that is not wired fails before any request",
			args:   []string{"activities"},
			stderr: "cmd.Rows output is not wired yet: keep the command Custom\n",
			code:   1,
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
