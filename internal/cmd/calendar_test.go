package cmd

import (
	"testing"
	"time"
)

// TestCalendarMonthDefaultsToThisMonth pins the request of calendar month
// without arguments: the month of the clock, sent 0-based. The goldens give
// the month, so only this test covers the clock.
func TestCalendarMonthDefaultsToThisMonth(t *testing.T) {
	tests := []struct {
		now  time.Time
		want string
	}{
		{time.Date(2026, time.January, 31, 23, 0, 0, 0, time.Local), "GET /calendar-service/year/2026/month/0"},
		{time.Date(2026, time.December, 1, 0, 0, 0, 0, time.Local), "GET /calendar-service/year/2026/month/11"},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			h := newHarness(t, harnessInput{routes: map[string]mockReply{
				tc.want: {body: `{"calendarItems":[]}`},
			}})
			stub(t, &nowFn, func() time.Time { return tc.now })
			_, stderr, code := h.run(nil, []string{"calendar", "month"})
			if code != 0 {
				t.Fatalf("exit %d, stderr %q", code, stderr)
			}
			got := h.requests()
			if len(got) != 1 || got[0].route != tc.want {
				t.Errorf("requests %v, want one %s", got, tc.want)
			}
		})
	}
}
