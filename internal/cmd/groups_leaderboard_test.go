package cmd

import (
	"testing"
	"time"
)

func TestGroupsLeaderboardRange(t *testing.T) {
	zone := time.FixedZone("UTC-7", -7*60*60)
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
		{"end on the default Monday in the afternoon", time.Date(2026, 10, 28, 15, 4, 0, 0, zone), "", "2026-10-26", "2026-10-26", "2026-10-26", false},
		{"end today on a Monday afternoon", time.Date(2026, 10, 26, 15, 4, 0, 0, zone), "", "today", "2026-10-26", "2026-10-26", false},
		{"monday late in the evening", time.Date(2026, 10, 26, 23, 30, 0, 0, zone), "", "", "2026-10-26", "2026-11-01", false},
		{"relative start and end", time.Date(2026, 10, 28, 15, 4, 0, 0, zone), "2d", "+4d", "2026-10-26", "2026-11-01", false},
		{"start yesterday gives seven days", time.Date(2026, 10, 28, 15, 4, 0, 0, zone), "yesterday", "", "2026-10-27", "2026-11-02", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			overrideNowFn(t, tt.now)
			c := GroupsLeaderboardCmd{Metric: "steps", Start: decodeDate(t, tt.start), End: decodeDate(t, tt.end)}
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

// decodeDate resolves a flag value as the Date decoder does, and keeps an
// empty value for a flag that was left out.
func decodeDate(t *testing.T, raw Date) Date {
	t.Helper()
	if raw == "" {
		return ""
	}
	d, err := resolveDate(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	return Date(d)
}
