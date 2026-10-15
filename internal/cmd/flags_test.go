package cmd

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
)

func executeRecorded(t *testing.T, args ...string) (int, []*url.URL) {
	t.Helper()
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
		_, _ = w.Write([]byte("[]"))
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
			args:      []string{"activities", "search", "--start-date", "2024-01-01", "--limit", "5", "--offset", "10"},
			wantPath:  "/activitylist-service/activities/search/activities",
			wantQuery: map[string]string{"start": "10", "limit": "5", "startDate": "2024-01-01"},
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
		{args: []string{"events", "list", "--offset", "3", "--limit", "2"}},
		{args: []string{"events", "list", "--offset", "-20"}},
		{args: []string{"events", "list", "--page", "2"}},
		{args: []string{"events", "list", "-p", "2"}},
		{args: []string{"activities", "list", "--start", "10"}},
		{args: []string{"activities", "list", "-s", "10"}},
		{args: []string{"activities", "search", "--start-date", "2024-01-01", "-s", "10"}},
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
