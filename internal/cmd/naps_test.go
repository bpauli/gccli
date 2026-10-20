package cmd

import (
	"strings"
	"testing"
	"time"
)

// TestNapsAddDST runs add on days with a daylight saving change. The
// harness clock is on a day without a change, so each case sets its own.
func TestNapsAddDST(t *testing.T) {
	cases := []struct {
		name               string
		zone               string
		now                time.Time
		start, end         string
		wantErr            string
		wantStart, wantEnd string
	}{
		{
			name: "gap end", zone: "America/New_York",
			now: time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC), start: "01:50", end: "02:10",
			wantErr: "the time 02:10 does not exist on 2026-03-08 in your Garmin time zone (America/New_York)",
		},
		{
			name: "gap start", zone: "America/New_York",
			now: time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC), start: "02:30", end: "03:10",
			wantErr: "the time 02:30 does not exist on 2026-03-08 in your Garmin time zone (America/New_York)",
		},
		{
			name: "gap start in Berlin", zone: "Europe/Berlin",
			now: time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC), start: "02:30", end: "03:30",
			wantErr: "the time 02:30 does not exist on 2026-03-29 in your Garmin time zone (Europe/Berlin)",
		},
		{
			name: "ambiguous start", zone: "America/New_York",
			now: time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC), start: "01:30", end: "01:50",
			wantErr: "the time 01:30 is ambiguous on 2026-11-01 in your Garmin time zone (America/New_York)",
		},
		{
			name: "ambiguous end in Berlin", zone: "Europe/Berlin",
			now: time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC), start: "01:30", end: "02:15",
			wantErr: "the time 02:15 is ambiguous on 2026-10-25 in your Garmin time zone (Europe/Berlin)",
		},
		{
			name: "across the gap", zone: "America/New_York",
			now: time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC), start: "01:30", end: "03:30",
			wantStart: "2026-03-08T06:30:00.000", wantEnd: "2026-03-08T07:30:00.000",
		},
		{
			name: "across the repeated hour in Berlin", zone: "Europe/Berlin",
			now: time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC), start: "01:30", end: "03:30",
			wantStart: "2026-10-24T23:30:00.000", wantEnd: "2026-10-25T02:30:00.000",
		},
		{
			name: "day after the change", zone: "America/New_York",
			now: time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC), start: "01:30", end: "03:30",
			wantStart: "2026-03-09T05:30:00.000", wantEnd: "2026-03-09T07:30:00.000",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			date := tc.now.Format("2006-01-02")
			h := newHarness(t, harnessInput{routes: map[string]mockReply{
				profileRoute:                    {body: `{"displayName":"tester","timeZone":"` + tc.zone + `"}`},
				socialRoute:                     {body: `{"profileId":3141592}`},
				"POST /sleep-service/sleep/nap": {body: `{"calendarDate":"` + date + `"}`},
			}})
			stub(t, &nowFn, func() time.Time { return tc.now })
			_, stderr, code := h.run(nil, []string{"health", "sleep", "naps", "add", tc.start, tc.end})
			if tc.wantErr != "" {
				if code != 1 || !strings.Contains(stderr, tc.wantErr) {
					t.Errorf("exit %d, stderr %q, want exit 1 with %q", code, stderr, tc.wantErr)
				}
				if w := h.writes(); len(w) != 0 {
					t.Errorf("sent %v, want no write", w)
				}
				return
			}
			if code != 0 {
				t.Fatalf("exit %d, stderr %q", code, stderr)
			}
			var body string
			for _, r := range h.requests() {
				if r.method == "POST" {
					body = r.entry
				}
			}
			for _, want := range []string{`"napStartTimestampGMT": "` + tc.wantStart + `"`, `"napEndTimestampGMT": "` + tc.wantEnd + `"`} {
				if !strings.Contains(body, want) {
					t.Errorf("POST body %q lacks %s", body, want)
				}
			}
		})
	}
}
