package cmd

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alecthomas/kong"
)

var flagTestNow = time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)

func TestDateDecode(t *testing.T) {
	overrideNowFn(t, flagTestNow)
	tests := []struct {
		args    []string
		want    Date
		wantErr string
	}{
		{args: []string{"--r", "2024-01-02"}, want: ""},
		{args: []string{"--r", "2024-01-02", "--d", "2024-03-04"}, want: "2024-03-04"},
		{args: []string{"--r", "2024-01-02", "--d", "today"}, want: "2026-10-15"},
		{args: []string{"--r", "2024-01-02", "--d", "yesterday"}, want: "2026-10-14"},
		{args: []string{"--r", "2024-01-02", "--d", "3d"}, want: "2026-10-12"},
		{args: []string{"--r", "2024-01-02", "--d=+30d"}, want: "2026-11-14"},
		{args: []string{"--r", "2024-01-02", "--d", "bad"}, wantErr: `--d: invalid date "bad"`},
		{args: []string{"--r", "2024-01-02", "--d", "20"}, wantErr: `--d: invalid date "20"`},
		{args: []string{"--r", "2024-01-02", "--d", ""}, wantErr: "--d: empty date"},
		{args: []string{"--r", "2024-01-02", "--d", " "}, wantErr: "--d: empty date"},
		{args: []string{"--r", ""}, wantErr: "--r: empty date"},
		{args: []string{"--d", "today"}, wantErr: "missing flags: --r=DATE"},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			var cli struct {
				D Date `name:"d"`
				R Date `name:"r" required:""`
			}
			parser, err := kong.New(&cli)
			if err != nil {
				t.Fatal(err)
			}
			_, err = parser.Parse(tt.args)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if cli.D != tt.want {
				t.Errorf("D = %q, want %q", cli.D, tt.want)
			}
		})
	}
}

// TestDateTimeDecode pins the format set of DateTime and the zone it parses
// in. The accepted arguments of activity create --date are stable contract,
// so an extra format is a defect, and the clock runs in UTC+2 here so that a
// parse in UTC shows as a wrong wall time or zone.
func TestDateTimeDecode(t *testing.T) {
	loc := time.FixedZone("UTC+2", 2*60*60)
	overrideNowFn(t, time.Date(2026, 10, 15, 12, 0, 0, 0, loc))
	const layout = "2006-01-02 15:04:05.000000000 -0700 MST"
	accepted := []struct {
		raw  string
		want time.Time
	}{
		{"2026-10-15T07:30:15", time.Date(2026, 10, 15, 7, 30, 15, 0, loc)},
		{"2026-10-15T07:30", time.Date(2026, 10, 15, 7, 30, 0, 0, loc)},
		{"2026-10-15 07:30:15", time.Date(2026, 10, 15, 7, 30, 15, 0, loc)},
		{"2026-10-15 07:30", time.Date(2026, 10, 15, 7, 30, 0, 0, loc)},
		{"2026-10-15", time.Date(2026, 10, 15, 0, 0, 0, 0, loc)},
		{"2026-10-15T7:30", time.Date(2026, 10, 15, 7, 30, 0, 0, loc)},
		{"2026-10-15T07:30:15.5", time.Date(2026, 10, 15, 7, 30, 15, 500000000, loc)},
		{"2026-10-15T00:00:00", time.Date(2026, 10, 15, 0, 0, 0, 0, loc)},
		{"2026-10-15T23:59:59", time.Date(2026, 10, 15, 23, 59, 59, 0, loc)},
		{"2024-02-29", time.Date(2024, 2, 29, 0, 0, 0, 0, loc)},
	}
	rejected := []string{
		"2026-10-15T07:30:15Z",
		"2026-10-15T07:30:15+02:00",
		"2026-10-15T07:30+02:00",
		"15.10.2026",
		"15.10.2026 07:30",
		"today",
		"3d",
		"",
		" ",
		" 2026-10-15",
		"2026-10-15 ",
		"2026-02-30",
		"2026-10-15T24:00",
		"2026-10-15T07:60",
		"2026-10-15T07",
		"2026-10-15T07:30:15.5Z",
	}
	decode := func(t *testing.T, raw string) (DateTime, error) {
		t.Helper()
		var cli struct {
			D DateTime `name:"d"`
		}
		parser, err := kong.New(&cli)
		if err != nil {
			t.Fatal(err)
		}
		_, err = parser.Parse([]string{"--d", raw})
		return cli.D, err
	}
	for _, tt := range accepted {
		t.Run(tt.raw, func(t *testing.T) {
			got, err := decode(t, tt.raw)
			if err != nil {
				t.Fatalf("%q: err = %v, want a time", tt.raw, err)
			}
			if got.Format(layout) != tt.want.Format(layout) {
				t.Errorf("%q = %s, want %s", tt.raw, got.Format(layout), tt.want.Format(layout))
			}
		})
	}
	for _, raw := range rejected {
		t.Run("rejected/"+raw, func(t *testing.T) {
			got, err := decode(t, raw)
			if err == nil {
				t.Fatalf("%q = %s, want a usage error", raw, got.Format(layout))
			}
			if want := "--d: invalid date and time " + strconv.Quote(raw); !strings.Contains(err.Error(), want) {
				t.Errorf("%q: err = %v, want %q", raw, err, want)
			}
		})
	}
}

func executeRecorded(t *testing.T, args ...string) (int, []*url.URL) {
	t.Helper()
	overrideNowFn(t, flagTestNow)
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	var mu sync.Mutex
	var reqs []*url.URL
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		reqs = append(reqs, r.URL)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		body := "[]"
		if strings.HasPrefix(r.URL.Path, "/calendar-service/year/") {
			body = "{}"
		}
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var code int
	captureOutput(t, func() {
		code = Execute(append([]string{"--json", "--account", "test@example.com"}, args...), "1.0.0", "abc123", "2024-01-01")
	})
	mu.Lock()
	defer mu.Unlock()
	return code, reqs
}

func TestFlagArgv(t *testing.T) {
	tests := []struct {
		args      []string
		wantPath  string
		wantQuery map[string]string
	}{
		{
			args:      []string{"activities", "list", "--limit", "5", "--offset", "10"},
			wantPath:  "/activitylist-service/activities/search/activities",
			wantQuery: map[string]string{"start": "10", "limit": "5"},
		},
		{
			args:      []string{"activities", "list", "-l", "5"},
			wantPath:  "/activitylist-service/activities/search/activities",
			wantQuery: map[string]string{"start": "0", "limit": "5"},
		},
		{
			args:      []string{"activities", "search", "--start", "2024-01-01", "--limit", "5", "--offset", "10"},
			wantPath:  "/activitylist-service/activities/search/activities",
			wantQuery: map[string]string{"start": "10", "limit": "5", "startDate": "2024-01-01"},
		},
		{
			args:      []string{"activities", "search", "--start", "3d", "--end", "today"},
			wantPath:  "/activitylist-service/activities/search/activities",
			wantQuery: map[string]string{"startDate": "2026-10-12", "endDate": "2026-10-15", "start": "0", "limit": "20"},
		},
		{
			args:     []string{"wellness", "menstrual-cycle", "--start", "3d", "--end", "today"},
			wantPath: "/periodichealth-service/menstrualcycle/dayview/2026-10-12/2026-10-15",
		},
		{
			args:     []string{"wellness", "menstrual-summary", "--start", "2024-01-01", "--end", "yesterday"},
			wantPath: "/periodichealth-service/menstrualcycle/summary/2024-01-01/2026-10-14",
		},
		{
			args:      []string{"workouts", "list", "--limit", "5", "--offset", "10"},
			wantPath:  "/workout-service/workouts",
			wantQuery: map[string]string{"start": "10", "limit": "5"},
		},
		{
			args:      []string{"events", "list", "--start", "2024-01-01", "--limit", "5", "--offset", "10"},
			wantPath:  "/calendar-service/events",
			wantQuery: map[string]string{"startDate": "2024-01-01", "pageIndex": "3", "limit": "5"},
		},
		{
			args:      []string{"events", "list", "--start", "2024-01-01"},
			wantPath:  "/calendar-service/events",
			wantQuery: map[string]string{"pageIndex": "1", "limit": "20"},
		},
		{
			args:     []string{"health", "steps", "daily", "--start", "3d", "--end", "today"},
			wantPath: "/usersummary-service/stats/steps/daily/2026-10-12/2026-10-15",
		},
		{
			args:     []string{"health", "steps", "weekly", "--end", "yesterday"},
			wantPath: "/usersummary-service/stats/steps/weekly/2026-10-14/4",
		},
		{
			args:     []string{"health", "stress", "weekly", "--end", "3d"},
			wantPath: "/usersummary-service/stats/stress/weekly/2026-10-12/4",
		},
		{
			args:      []string{"health", "body-battery", "range", "--start", "3d", "--end", "today"},
			wantPath:  "/wellness-service/wellness/bodyBattery/reports/daily",
			wantQuery: map[string]string{"startDate": "2026-10-12", "endDate": "2026-10-15"},
		},
		{
			args:     []string{"health", "intensity-minutes", "weekly", "--start", "3d", "--end", "today"},
			wantPath: "/usersummary-service/stats/im/weekly/2026-10-12/2026-10-15",
		},
		{
			args:     []string{"health", "race-predictions", "range", "--start", "3d", "--end", "today"},
			wantPath: "/metrics-service/metrics/racepredictions/range/2026-10-12/2026-10-15",
		},
		{
			args:      []string{"body", "composition", "--start", "3d", "--end", "today"},
			wantPath:  "/weight-service/weight/dateRange",
			wantQuery: map[string]string{"startDate": "2026-10-12", "endDate": "2026-10-15"},
		},
		{
			args:     []string{"body", "weigh-ins", "--start", "3d", "--end", "today"},
			wantPath: "/weight-service/weight/range/2026-10-12/2026-10-15",
		},
		{
			args:     []string{"body", "blood-pressure", "--start", "3d", "--end", "today"},
			wantPath: "/bloodpressure-service/bloodpressure/range/2026-10-12/2026-10-15",
		},
		{
			args:     []string{"devices", "solar", "123", "--start", "yesterday"},
			wantPath: "/web-gateway/solar/123/2026-10-14/2026-10-14",
		},
		{
			args:      []string{"events", "list", "--start", "+30d"},
			wantPath:  "/calendar-service/events",
			wantQuery: map[string]string{"startDate": "2026-11-14"},
		},
		{
			args:     []string{"workouts", "schedule", "list", "--start", "3d", "--end", "today"},
			wantPath: "/calendar-service/year/2026/month/9/day/12/start/0",
		},
		{args: []string{"health", "steps", "daily", "--start", "bad", "--end", "today"}},
		{args: []string{"health", "steps", "weekly", "--end", "bad"}},
		{args: []string{"health", "stress", "weekly", "--end", "bad"}},
		{args: []string{"health", "body-battery", "range", "--start", "today", "--end", "bad"}},
		{args: []string{"health", "intensity-minutes", "weekly", "--start", "bad", "--end", "today"}},
		{args: []string{"health", "race-predictions", "range", "--start", "bad", "--end", "today"}},
		{args: []string{"body", "composition", "--start", "bad", "--end", "today"}},
		{args: []string{"body", "weigh-ins", "--start", "bad", "--end", "today"}},
		{args: []string{"body", "blood-pressure", "--start", "bad", "--end", "today"}},
		{args: []string{"devices", "solar", "123", "--start", "bad"}},
		{args: []string{"events", "list", "--start", "bad"}},
		{args: []string{"workouts", "schedule", "list", "--start", "bad", "--end", "today"}},
		{args: []string{"activities", "search", "--start", ""}},
		{args: []string{"workouts", "schedule", "list", "--start", "", "--end", "today"}},
		{args: []string{"events", "list", "--start", ""}},
		{args: []string{"health", "steps", "daily", "--start", "", "--end", "today"}},
		{
			args:      []string{"events", "list", "--sort", "eventDate_desc"},
			wantPath:  "/calendar-service/events",
			wantQuery: map[string]string{"sortOrder": "eventDate_desc"},
		},
		{args: []string{"events", "list", "-s", "eventDate_desc"}},
		{args: []string{"events", "list", "--offset", "3", "--limit", "2"}},
		{args: []string{"events", "list", "--offset", "-20"}},
		{args: []string{"events", "list", "--limit", "0"}},
		{args: []string{"activities", "list", "--offset=-5"}},
		{args: []string{"activities", "list", "--limit=-1"}},
		{args: []string{"activities", "list", "--limit", "0"}},
		{args: []string{"activities", "search", "--start", "2024-01-01", "--offset=-1"}},
		{args: []string{"workouts", "list", "--offset=-5"}},
		{args: []string{"workouts", "list", "--limit", "0"}},
		{args: []string{"events", "list", "--page", "2"}},
		{args: []string{"events", "list", "-p", "2"}},
		{args: []string{"activities", "list", "--start", "10"}},
		{args: []string{"activities", "list", "-s", "10"}},
		{args: []string{"activities", "search", "--start", "2024-01-01", "-s", "10"}},
		{args: []string{"activities", "search", "--start", "20"}},
		{args: []string{"activities", "search", "--start-date", "2024-01-01"}},
		{args: []string{"activities", "search", "--start", "2024-01-01", "--end-date", "2024-01-31"}},
		{args: []string{"wellness", "menstrual-cycle", "--start-date", "2024-01-01", "--end-date", "2024-01-31"}},
		{args: []string{"wellness", "menstrual-cycle", "--start", "2024-01-01"}},
		{args: []string{"wellness", "menstrual-summary", "--start-date", "2024-01-01", "--end-date", "2024-01-31"}},
		{args: []string{"wellness", "menstrual-summary", "--start", "bad", "--end", "today"}},
		{args: []string{"workouts", "list", "--start", "10"}},
		{args: []string{"workouts", "list", "-s", "10"}},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			code, reqs := executeRecorded(t, tt.args...)
			if tt.wantPath == "" {
				if code != 2 {
					t.Errorf("exit code = %d, want 2", code)
				}
				if len(reqs) != 0 {
					t.Errorf("sent %d requests, want none: %v", len(reqs), reqs)
				}
				return
			}
			if code != 0 {
				t.Fatalf("exit code = %d, want 0", code)
			}
			if len(reqs) != 1 {
				t.Fatalf("sent %d requests, want 1: %v", len(reqs), reqs)
			}
			if reqs[0].Path != tt.wantPath {
				t.Errorf("path = %s, want %s", reqs[0].Path, tt.wantPath)
			}
			q := reqs[0].Query()
			for k, want := range tt.wantQuery {
				if got := q.Get(k); got != want {
					t.Errorf("query %s = %q, want %q (%s)", k, got, want, reqs[0].RawQuery)
				}
			}
		})
	}
}

func walkFlags(t *testing.T, fn func(path string, f *kong.Flag)) {
	t.Helper()
	var walk func(path string, node *kong.Node)
	walk = func(path string, node *kong.Node) {
		for _, f := range node.Flags {
			fn(path, f)
		}
		for _, child := range node.Children {
			if child.Type == kong.CommandNode {
				walk(strings.TrimSpace(path+" "+child.Name), child)
			}
		}
	}
	walk("", newTestParser(t).Model.Node)
}

func TestNoLegacyFlagNames(t *testing.T) {
	dateType := reflect.TypeOf(Date(""))
	walkFlags(t, func(path string, f *kong.Flag) {
		for _, name := range append([]string{f.Name}, f.Aliases...) {
			switch name {
			case "start-date", "end-date", "page":
				t.Errorf("%s: legacy flag --%s", path, name)
			case "start", "end":
				if f.Target.Type() != dateType {
					t.Errorf("%s: --%s has type %s, want Date", path, name, f.Target.Type())
				}
			}
		}
	})
}

func TestShortFlagsHaveOneMeaning(t *testing.T) {
	type use struct{ long, path string }
	first := map[rune]use{}
	walkFlags(t, func(path string, f *kong.Flag) {
		if f.Short == 0 {
			return
		}
		prev, ok := first[f.Short]
		if !ok {
			first[f.Short] = use{long: f.Name, path: path}
			return
		}
		if prev.long != f.Name {
			t.Errorf("-%c means --%s in %q and --%s in %q", f.Short, prev.long, prev.path, f.Name, path)
		}
	})
}

// walkValues calls fn on every flag and positional of the whole Kong tree,
// Custom commands included.
func walkValues(t *testing.T, fn func(path string, v *kong.Value)) {
	t.Helper()
	var walk func(path string, node *kong.Node)
	walk = func(path string, node *kong.Node) {
		for _, f := range node.Flags {
			fn(path, f.Value)
		}
		for _, p := range node.Positional {
			fn(path, p)
		}
		for _, child := range node.Children {
			if child.Type == kong.CommandNode {
				walk(strings.TrimSpace(path+" "+child.Name), child)
			}
		}
	}
	walk("", newTestParser(t).Model.Node)
}

// TestDateFieldsHaveDateType walks the whole Kong tree, Custom commands
// included: a flag or a positional that is a date by its name or by its
// help must be a Date or a DateTime, so that a bad date fails at parse
// time, before the login. checkLeaf has the same rule for Leaf rows.
func TestDateFieldsHaveDateType(t *testing.T) {
	walkValues(t, func(path string, v *kong.Value) {
		if msg := dateTypeError(v.Name, v.Help, v.Target.Type()); msg != "" {
			t.Errorf("%s: %s", path, msg)
		}
	})
}

// TestIDFieldsHaveIDType walks the whole Kong tree, Custom commands
// included: a flag or a positional that is an ID by its name must not be a
// plain string, so that an empty ID fails at parse time, before the login.
// checkLeaf has the same rule for Leaf rows.
func TestIDFieldsHaveIDType(t *testing.T) {
	walkValues(t, func(path string, v *kong.Value) {
		if msg := idTypeError(v.Name, v.Target.Type()); msg != "" {
			t.Errorf("%s: %s", path, msg)
		}
	})
}

// requiredStringExempt names the required plain-string values of the Kong
// tree that are not a FreeText on purpose, with the reason. A key is the
// command path and the value name.
var requiredStringExempt = map[string]string{
	"activity describe text":       "an empty text clears the description by design",
	"calendar notes add name":      "Validate rejects an empty or blank name with its own message",
	"gear collections rename name": "Validate rejects an empty or blank name with its own message",
}

// TestRequiredStringsHaveTextType walks the whole Kong tree, Custom
// commands included: a required flag or positional of type string lets an
// empty value through to the login, so its type must be FreeText, or ID when it
// fills a request path. Kong itself rejects an empty enum value and an empty
// existingfile value, so those pass. An exempt value must still be a
// required plain string, or the entry is stale.
func TestRequiredStringsHaveTextType(t *testing.T) {
	seen := map[string]bool{}
	walkValues(t, func(path string, v *kong.Value) {
		key := path + " " + v.Name
		plain := v.Required && v.Target.Type() == reflect.TypeFor[string]() && v.Enum == "" && v.Tag.Type == ""
		if reason, ok := requiredStringExempt[key]; ok {
			seen[key] = true
			if !plain {
				t.Errorf("%s: requiredStringExempt names it (%s), but it is not a required plain string any more: drop the entry", key, reason)
			}
			return
		}
		if plain {
			t.Errorf("%s: a required string, which Kong lets be empty: make the field a FreeText, or an ID when it fills a request path, so Kong rejects an empty value at parse time", key)
		}
	})
	for key := range requiredStringExempt {
		if !seen[key] {
			t.Errorf("requiredStringExempt names %q, but the Kong tree has no such value: drop the entry", key)
		}
	}
}

// TestNonEmptyDecode pins the one rule of ID and FreeText: an empty or blank
// value is a usage error, and any other value is kept as given.
// TestNumericIDDecode pins the IDs that NumericID accepts: a base 10 int64
// with an optional sign and leading zeros, and nothing else.
func TestNumericIDDecode(t *testing.T) {
	accepted := []struct {
		raw  string
		want NumericID
	}{
		{"1", 1},
		{"007", 7},
		{"010", 10},
		{"+7", 7},
		{"21504387319", 21504387319},
		{"9223372036854775807", 9223372036854775807},
	}
	rejected := []string{
		"",
		" ",
		" 7",
		"7 ",
		"\t7",
		"0x10",
		"0X10",
		"0o10",
		"0b10",
		"1_000",
		"1.5",
		"1e3",
		"abc",
		"7a",
		"9223372036854775808",
	}
	decode := func(t *testing.T, raw string) (NumericID, error) {
		t.Helper()
		var cli struct {
			ID NumericID `name:"id"`
		}
		parser, err := kong.New(&cli)
		if err != nil {
			t.Fatal(err)
		}
		_, err = parser.Parse([]string{"--id", raw})
		return cli.ID, err
	}
	for _, tt := range accepted {
		t.Run(tt.raw, func(t *testing.T) {
			got, err := decode(t, tt.raw)
			if err != nil {
				t.Fatalf("%q: err = %v, want %d", tt.raw, err, tt.want)
			}
			if got != tt.want {
				t.Errorf("%q = %d, want %d", tt.raw, got, tt.want)
			}
		})
	}
	for _, raw := range rejected {
		t.Run("rejected/"+raw, func(t *testing.T) {
			got, err := decode(t, raw)
			if err == nil {
				t.Fatalf("%q = %d, want a usage error", raw, got)
			}
			if want := "--id: invalid ID " + strconv.Quote(raw) + ": expected a number"; !strings.Contains(err.Error(), want) {
				t.Errorf("%q: err = %v, want %q", raw, err, want)
			}
		})
	}
}

func TestNonEmptyDecode(t *testing.T) {
	tests := []struct {
		args    []string
		wantID  ID
		want    FreeText
		wantErr string
	}{
		{args: []string{"7", "Morning Run"}, wantID: "7", want: "Morning Run"},
		{args: []string{" 7 ", " x "}, wantID: " 7 ", want: " x "},
		{args: []string{"", "x"}, wantErr: "<id>: must not be empty"},
		{args: []string{" ", "x"}, wantErr: "<id>: must not be empty"},
		{args: []string{"7", ""}, wantErr: "<name>: must not be empty"},
		{args: []string{"7", " \t"}, wantErr: "<name>: must not be empty"},
		{args: []string{"7"}, wantErr: `expected "<name>"`},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			var cli struct {
				ID   ID       `arg:""`
				Name FreeText `arg:""`
			}
			parser, err := kong.New(&cli)
			if err != nil {
				t.Fatal(err)
			}
			_, err = parser.Parse(tt.args)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if cli.ID != tt.wantID || cli.Name != tt.want {
				t.Errorf("got ID %q, Name %q, want %q, %q", cli.ID, cli.Name, tt.wantID, tt.want)
			}
		})
	}
}
