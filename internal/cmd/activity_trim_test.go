package cmd

import (
	"strings"
	"testing"
	"time"
)

func TestActivityClock(t *testing.T) {
	start := time.Date(2026, 1, 10, 8, 0, 0, 0, time.UTC)
	tests := []struct {
		name        string
		answer      string
		wantStart   time.Time
		wantElapsed time.Duration
		wantErr     string
	}{
		{"one fraction digit", `{"summaryDTO":{"startTimeGMT":"2026-01-10T08:00:00.0","elapsedDuration":3461.0}}`, start, 3461 * time.Second, ""},
		{"no fraction", `{"summaryDTO":{"startTimeGMT":"2026-01-10T08:00:00","elapsedDuration":60}}`, start, time.Minute, ""},
		{"milliseconds", `{"summaryDTO":{"startTimeGMT":"2026-01-10T08:00:00.250","elapsedDuration":1.5}}`, start.Add(250 * time.Millisecond), 1500 * time.Millisecond, ""},
		{"no start", `{"summaryDTO":{"elapsedDuration":60}}`, time.Time{}, 0, "summaryDTO.startTimeGMT"},
		{"bad start", `{"summaryDTO":{"startTimeGMT":"10.01.2026 08:00","elapsedDuration":60}}`, time.Time{}, 0, `summaryDTO.startTimeGMT "10.01.2026 08:00"`},
		{"no elapsed", `{"summaryDTO":{"startTimeGMT":"2026-01-10T08:00:00.0"}}`, time.Time{}, 0, "summaryDTO.elapsedDuration"},
		{"empty body", ``, time.Time{}, 0, "summaryDTO.startTimeGMT"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStart, gotElapsed, err := activityClock([]byte(tt.answer))
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want one that names %s", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !gotStart.Equal(tt.wantStart) || gotStart.Location() != time.UTC || gotElapsed != tt.wantElapsed {
				t.Errorf("got %s (%s), %s; want %s (UTC), %s", gotStart, gotStart.Location(), gotElapsed, tt.wantStart, tt.wantElapsed)
			}
		})
	}
}

func TestActivityLegsStartIsUTC(t *testing.T) {
	local := time.Local
	time.Local = time.FixedZone("UTC-5", -5*60*60)
	t.Cleanup(func() { time.Local = local })
	h := newHarness(t, harnessInput{routes: map[string]mockReply{
		"GET /activity-service/activity/v2/1/reconstruct": {body: `{"activityReconstructDTO":{"reconstructRequest":[{"activityId":2,"activityTypeDTO":{"typeKey":"running"},"startTime":1768032000000,"endTime":1768035600000}]}}`},
	}})

	stdout, stderr, code := h.run([]string{"--plain"}, []string{"activity", "legs", "1"})
	if want := "2\trunning\t2026-01-10T08:00:00Z\t1:00:00\n"; code != 0 || stdout != want {
		t.Errorf("exit %d, stdout %q, stderr %q; want exit 0, stdout %q", code, stdout, stderr, want)
	}
}
