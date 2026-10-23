package cmd

import (
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/bpauli/gccli/internal/garminapi"
)

// thursday is 2026-10-15 09:30 local, a Thursday.
var thursday = time.Date(2026, time.October, 15, 9, 30, 0, 0, time.Local)

// TestChallengeDays pins the days of each --duration, with and without
// --start and --end, as the web app sets them.
func TestChallengeDays(t *testing.T) {
	at := func(month time.Month, day int) time.Time { return time.Date(2026, month, day, 9, 30, 0, 0, time.Local) }
	tests := []struct {
		name        string
		duration    string
		start, end  Date
		now         time.Time
		first, last string
	}{
		{"day starts and ends today", "day", "", "", thursday, "2026-10-15", "2026-10-15"},
		{"day with --start", "day", "2026-10-20", "", thursday, "2026-10-20", "2026-10-20"},
		{"week is seven days from today", "week", "", "", thursday, "2026-10-15", "2026-10-21"},
		{"week across a month end", "week", "2026-10-28", "", thursday, "2026-10-28", "2026-11-03"},
		{"weekend on a Thursday is this Saturday", "weekend", "", "", thursday, "2026-10-17", "2026-10-18"},
		{"weekend on a Saturday is today", "weekend", "", "", at(time.October, 17), "2026-10-17", "2026-10-18"},
		{"weekend on a Sunday is yesterday", "weekend", "", "", at(time.October, 18), "2026-10-17", "2026-10-18"},
		{"weekend on a Monday is the next Saturday", "weekend", "", "", at(time.October, 19), "2026-10-24", "2026-10-25"},
		{"weekend with a Saturday --start", "weekend", "2026-10-31", "", thursday, "2026-10-31", "2026-11-01"},
		{"month is this month", "month", "", "", thursday, "2026-10-01", "2026-10-31"},
		{"month ends at the end of the start month", "month", "2026-02-10", "", thursday, "2026-02-10", "2026-02-28"},
		{"month of a leap February", "month", "2028-02-03", "", thursday, "2028-02-03", "2028-02-29"},
		{"custom from today", "custom", "", "2026-10-20", thursday, "2026-10-15", "2026-10-20"},
		{"custom of one day", "custom", "2026-10-20", "2026-10-20", thursday, "2026-10-20", "2026-10-20"},
		{"custom of 60 days", "custom", "", "2026-12-14", thursday, "2026-10-15", "2026-12-14"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			first, last, err := challengeDays(tc.duration, tc.start, tc.end, tc.now)
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			if got := first.Format(time.DateOnly) + " " + last.Format(time.DateOnly); got != tc.first+" "+tc.last {
				t.Errorf("days = %s, want %s %s", got, tc.first, tc.last)
			}
		})
	}
}

func TestChallengeDaysRejects(t *testing.T) {
	tests := []struct {
		duration   string
		start, end Date
		want       string
	}{
		{"weekend", "2026-10-15", "", "a weekend challenge starts on a Saturday, and 2026-10-15 is a Thursday"},
		{"weekend", "2026-10-18", "", "a weekend challenge starts on a Saturday, and 2026-10-18 is a Sunday"},
		{"custom", "", "", "--duration custom needs --end"},
		{"custom", "", "2026-10-14", "--end 2026-10-14 is before the first day 2026-10-15"},
		{"custom", "", "2026-12-15", "--end 2026-12-15 is more than 60 days after the first day 2026-10-15"},
		{"custom", "2026-11-01", "2027-01-01", "--end 2027-01-01 is more than 60 days after the first day 2026-11-01"},
		{"day", "", "2026-10-20", "--end works only with --duration custom"},
		{"weekend", "", "2026-10-18", "--end works only with --duration custom"},
		{"week", "", "2026-10-20", "--end works only with --duration custom"},
		{"month", "", "2026-10-31", "--end works only with --duration custom"},
	}
	for _, tc := range tests {
		t.Run(tc.duration+" "+string(tc.start)+" "+string(tc.end), func(t *testing.T) {
			_, _, err := challengeDays(tc.duration, tc.start, tc.end, thursday)
			if err == nil || err.Error() != tc.want {
				t.Errorf("err = %v, want %q", err, tc.want)
			}
		})
	}
}

// TestChallengeIDs pins the Garmin ids that each --type and --duration value
// sends, and that the flag enums offer exactly the values of the tables.
func TestChallengeIDs(t *testing.T) {
	wantTypes := map[string]garminapi.ChallengeType{
		"walking": 0, "running": 1, "cycling": 2, "swimming": 3, "steps": 4, "pushes": 10, "pushing-distance": 11,
	}
	if !maps.Equal(challengeTypes, wantTypes) {
		t.Errorf("challengeTypes = %v, want %v", challengeTypes, wantTypes)
	}
	wantDurations := map[string]garminapi.ChallengeDuration{"week": 0, "month": 1, "custom": 2, "weekend": 3, "day": 4}
	gotDurations := map[string]garminapi.ChallengeDuration{}
	for name, d := range challengeDurations {
		gotDurations[name] = d.id
	}
	if !maps.Equal(gotDurations, wantDurations) {
		t.Errorf("challengeDurations ids = %v, want %v", gotDurations, wantDurations)
	}
	cmd := reflect.TypeFor[ChallengesAdhocCreateCmd]()
	for field, keys := range map[string][]string{
		"Type":     slices.Sorted(maps.Keys(challengeTypes)),
		"Duration": slices.Sorted(maps.Keys(challengeDurations)),
	} {
		f, _ := cmd.FieldByName(field)
		enum := strings.Split(f.Tag.Get("enum"), ",")
		slices.Sort(enum)
		if !slices.Equal(enum, keys) {
			t.Errorf("--%s enum = %q, want the table keys %q", strings.ToLower(field), enum, keys)
		}
	}
}

// TestChallengesAdhocCreateDefaultsToToday pins the days that the command
// sends without --start: the goldens give --start, so only this test covers
// the clock.
func TestChallengesAdhocCreateDefaultsToToday(t *testing.T) {
	h := newHarness(t, harnessInput{routes: map[string]mockReply{
		"GET /userprofile-service/userprofile/settings": {body: `{"displayName":"tester","profileId":3141592}`},
		"GET /userprofile-service/socialProfile/tester": {body: `{"displayName":"tester","profileId":3141592}`},
		"POST /adhocchallenge-service/adHocChallenge":   {body: `{"uuid":"FEDCBA9876543210FEDCBA9876543210"}`},
	}})
	_, stderr, code := h.run(nil, []string{"challenges", "adhoc", "create", "Weekend walk", "--type", "walking", "--duration", "weekend"})
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	got := h.requests()
	if len(got) != 3 {
		t.Fatalf("requests %v, want the profile reads and the create", got)
	}
	for _, want := range []string{`"startDate": "2026-10-17T00:00:00.00"`, `"endDate": "2026-10-18T23:59:59.999"`, `"durationTypeId": 3`, `"socialChallengeActivityTypeId": 0`} {
		if !strings.Contains(got[2].entry, want) {
			t.Errorf("create request\n%s\ndoes not hold %s", got[2].entry, want)
		}
	}
}
