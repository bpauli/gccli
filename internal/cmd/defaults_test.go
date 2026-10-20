package cmd

import (
	"strings"
	"testing"
)

// TestOmittedDateDefaults pins the Custom commands that fill in an omitted
// date or time from the clock in Run, through orToday, orNow or a
// default:"today" tag. These are tests and not goldens, because the
// conformance suite replays every golden on the real clock.
func TestOmittedDateDefaults(t *testing.T) {
	cases := []struct {
		name  string
		args  []string
		route string
		body  string
	}{
		{
			"hydration add sends today",
			[]string{"hydration", "add", "250"},
			"PUT /usersummary-service/usersummary/hydration/log",
			`"calendarDate": "2026-10-15"`,
		},
		{
			"workouts schedule list asks for today",
			[]string{"workouts", "schedule", "list"},
			"GET /calendar-service/year/2026/month/9/day/15/start/0",
			"",
		},
		{
			"body composition asks for today",
			[]string{"body", "composition"},
			"GET /weight-service/weight/dateRange?startDate=2026-10-15&endDate=2026-10-15",
			"",
		},
		{
			"activity create sends now",
			[]string{"activity", "create", "--name", "Swim", "--type", "swimming", "--duration", "30m"},
			"POST /activity-service/activity",
			`"startTimeLocal": "2026-10-15T09:30:00.000"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t, harnessInput{routes: map[string]mockReply{tc.route: {body: `{"calendarItems":[]}`}}})
			_, stderr, code := h.run([]string{"--plain"}, tc.args)
			if code != 0 {
				t.Fatalf("exit %d, want 0; stderr:\n%s", code, stderr)
			}
			requests := h.requests()
			if len(requests) != 1 {
				t.Fatalf("sent %d requests, want 1: %v", len(requests), requests)
			}
			if requests[0].route != tc.route {
				t.Errorf("sent %s, want %s", requests[0].route, tc.route)
			}
			if !strings.Contains(requests[0].entry, tc.body) {
				t.Errorf("request body lacks %s:\n%s", tc.body, requests[0].entry)
			}
		})
	}
}
