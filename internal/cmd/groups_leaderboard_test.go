package cmd

import (
	"testing"
	"time"
)

func TestGroupsLeaderboardRange(t *testing.T) {
	tests := []struct {
		name       string
		now        time.Time
		start, end Date
		wantStart  string
		wantEnd    string
		wantErr    bool
	}{
		{"tuesday starts on its Monday", time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC), "", "", "2026-10-26", "2026-11-01", false},
		{"monday starts on itself", time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC), "", "", "2026-10-26", "2026-11-01", false},
		{"sunday ends its week", time.Date(2026, 11, 1, 23, 0, 0, 0, time.UTC), "", "", "2026-10-26", "2026-11-01", false},
		{"start alone gives seven days", time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC), "2026-10-01", "", "2026-10-01", "2026-10-07", false},
		{"start and end give a month", time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC), "2026-10-01", "2026-10-31", "2026-10-01", "2026-10-31", false},
		{"end alone ends there", time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC), "", "2026-10-27", "2026-10-26", "2026-10-27", false},
		{"end before the default start", time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC), "", "2026-10-20", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			overrideNowFn(t, tt.now)
			c := GroupsLeaderboardCmd{Metric: "steps", Start: tt.start, End: tt.end}
			err := c.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, want error %v", err, tt.wantErr)
			}
			if c.first != tt.wantStart || c.last != tt.wantEnd {
				t.Errorf("range = %s..%s, want %s..%s", c.first, c.last, tt.wantStart, tt.wantEnd)
			}
		})
	}
}
