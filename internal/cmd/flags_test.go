package cmd

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
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

// TestDateFieldsHaveDateType walks the whole Kong tree, Custom commands
// included: a flag or a positional that is a date by its name or by its
// help must be a Date or a DateTime, so that a bad date fails at parse
// time, before the login. checkLeaf has the same rule for Leaf rows.
func TestDateFieldsHaveDateType(t *testing.T) {
	check := func(path string, v *kong.Value) {
		if msg := dateTypeError(v.Name, v.Help, v.Target.Type()); msg != "" {
			t.Errorf("%s: %s", path, msg)
		}
	}
	var walk func(path string, node *kong.Node)
	walk = func(path string, node *kong.Node) {
		for _, f := range node.Flags {
			check(path, f.Value)
		}
		for _, p := range node.Positional {
			check(path, p)
		}
		for _, child := range node.Children {
			if child.Type == kong.CommandNode {
				walk(strings.TrimSpace(path+" "+child.Name), child)
			}
		}
	}
	walk("", newTestParser(t).Model.Node)
}
