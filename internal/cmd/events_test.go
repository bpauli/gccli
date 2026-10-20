package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bpauli/gccli/internal/outfmt"
	"github.com/bpauli/gccli/internal/payload"
)

func mustDistance(t *testing.T, s string) EventDistance {
	t.Helper()
	d, err := payload.ParseDistance(s)
	if err != nil {
		t.Fatalf("ParseDistance(%q): %v", s, err)
	}
	return EventDistance(d)
}

func mustGoal(t *testing.T, s string) Goal {
	t.Helper()
	g, err := payload.ParseGoal(s)
	if err != nil {
		t.Fatalf("ParseGoal(%q): %v", s, err)
	}
	return Goal(g)
}

func eventsTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/calendar-service/events", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"id":123,"eventName":"Race Day","date":"2026-03-29","eventType":"running","location":"Berlin, DE","shareableEventUuid":"abc-123"},
			{"id":456,"eventName":"Group Ride","date":"2026-03-30","eventType":"cycling","location":"Dresden, DE"}
		]`))
	})

	mux.HandleFunc("/calendar-service/event", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":99999,"eventName":"` + body["eventName"].(string) + `","date":"` + body["date"].(string) + `"}`))
	})

	mux.HandleFunc("/calendar-service/event/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	return httptest.NewServer(mux)
}

func TestEventsList_Success(t *testing.T) {
	server := eventsTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &EventsListCmd{StartDate: "2026-03-11", Pagination: Pagination{Limit: 20}, Sort: "eventDate_asc"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestEventsList_JSON(t *testing.T) {
	server := eventsTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &EventsListCmd{StartDate: "2026-03-11", Pagination: Pagination{Limit: 20}, Sort: "eventDate_asc"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestFormatEventListRows(t *testing.T) {
	events := []map[string]any{
		{
			"id":                 float64(123),
			"date":               "2026-03-29",
			"eventName":          "Race Day",
			"eventType":          "running",
			"location":           "Berlin, DE",
			"shareableEventUuid": "abc-123",
		},
		{
			"id":        float64(456),
			"date":      "2026-03-30",
			"eventName": "Group Ride",
			"eventType": "cycling",
			"location":  "Dresden, DE",
		},
	}

	rows := formatEventListRows(events)
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}

	// First event uses shareableEventUuid as ID.
	if rows[0][0] != "abc-123" {
		t.Errorf("id = %s, want abc-123", rows[0][0])
	}
	if rows[0][1] != "2026-03-29" {
		t.Errorf("date = %s, want 2026-03-29", rows[0][1])
	}

	// Second event falls back to numeric id.
	if rows[1][0] != "456" {
		t.Errorf("id = %s, want 456", rows[1][0])
	}
}

func TestEventsAdd_Success(t *testing.T) {
	server := eventsTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &EventsAddCmd{
		Name: "Spring Race",
		Date: "2026-04-01",
		Type: "running",
	}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestEventsAdd_AllFlags(t *testing.T) {
	server := eventsTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &EventsAddCmd{
		Name:     "Berlin Marathon",
		Date:     "2026-09-27",
		Type:     "running",
		Race:     true,
		Location: "Berlin, Germany",
		Time:     "09:15",
		Timezone: "Europe/Berlin",
		Distance: mustDistance(t, "42.195km"),
		Goal:     mustGoal(t, "3h30m"),
		Training: true,
		Private:  true,
		Note:     "My first marathon",
		URL:      "https://example.com",
	}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestEventsAdd_BadValueFailsBeforeCredentials(t *testing.T) {
	digits400 := "1" + strings.Repeat("0", 400)
	tests := []struct {
		name    string
		flags   []string
		wantErr string
	}{
		{"400-digit distance", []string{"--distance", digits400 + "km"}, "--distance: invalid distance \"" + digits400 + "km\": out of range"},
		{"zero goal", []string{"--goal", "0s"}, "--goal: goal duration must be positive"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runBadInput(t, append([]string{"events", "add", "--name", "x", "--date", "2026-11-01", "--type", "running"}, tt.flags...), tt.wantErr)
		})
	}
}

func TestEventsListPageIndex(t *testing.T) {
	tests := []struct {
		offset, limit int
		want          int
		wantErr       string
	}{
		{offset: 0, limit: 20, want: 1},
		{offset: 4, limit: 2, want: 3},
		{offset: 3, limit: 1, want: 4},
		{offset: 3, limit: 3, want: 2},
		{offset: 3, limit: 2, wantErr: "--offset 3 is not a multiple of --limit 2"},
		{offset: -20, limit: 20, wantErr: "--offset must not be negative, got -20"},
		{offset: 0, limit: 0, wantErr: "--limit must be at least 1, got 0"},
		{offset: 5, limit: 0, wantErr: "--limit must be at least 1, got 0"},
	}
	for _, tt := range tests {
		cmd := &EventsListCmd{Pagination: Pagination{Limit: tt.limit, Offset: tt.offset}}
		var got int
		err := cmd.Validate()
		if err == nil {
			got, err = cmd.pageIndex()
		}
		if tt.wantErr != "" {
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("offset %d limit %d: err = %v, want %q", tt.offset, tt.limit, err, tt.wantErr)
			}
			continue
		}
		if err != nil || got != tt.want {
			t.Errorf("offset %d limit %d: pageIndex() = %d, %v, want %d", tt.offset, tt.limit, got, err, tt.want)
		}
	}
}

func TestEventsListPageIndexRejectsLimitBelowOne(t *testing.T) {
	for _, limit := range []int{0, -1} {
		cmd := &EventsListCmd{Pagination: Pagination{Limit: limit, Offset: 4}}
		if _, err := cmd.pageIndex(); err == nil || !strings.Contains(err.Error(), "--limit must be at least 1") {
			t.Errorf("limit %d: pageIndex() err = %v, want a --limit error", limit, err)
		}
	}
}

func TestEventsList_LimitZeroIsUsageError(t *testing.T) {
	h := newHarness(t, harnessInput{})
	stdout, stderr, code := h.run([]string{"--json"}, []string{"events", "list", "--limit", "0"})
	if code != 2 || stdout != "" || !strings.Contains(stderr, "--limit must be at least 1, got 0") {
		t.Errorf("exit %d, want 2\nstdout %q, want empty\nstderr %q, want a --limit error", code, stdout, stderr)
	}
}

func TestEventsAdd_SentBody(t *testing.T) {
	tests := []struct {
		name string
		cmd  EventsAddCmd
		want string
	}{
		{
			name: "required flags only",
			cmd:  EventsAddCmd{Name: "Spring Race", Date: "2026-04-01", Type: "running"},
			want: `{
				"date": "2026-04-01",
				"eventName": "Spring Race",
				"eventType": "running",
				"race": false
			}`,
		},
		{
			name: "every flag",
			cmd: EventsAddCmd{
				Name:     "Berlin Marathon",
				Date:     "2026-09-27",
				Type:     "running",
				Race:     true,
				Location: "Berlin, Germany",
				Time:     "09:15",
				Timezone: "Europe/Berlin",
				Distance: mustDistance(t, "42.195km"),
				Goal:     mustGoal(t, "3h30m"),
				Training: true,
				Private:  true,
				Note:     "My first marathon",
				URL:      "https://example.com",
			},
			want: `{
				"completionTarget": {
					"unit": "kilometer",
					"unitType": "distance",
					"value": 42.195
				},
				"date": "2026-09-27",
				"eventCustomization": {
					"customGoal": {
						"unit": "second",
						"unitType": "time",
						"value": 12600
					},
					"isPrimaryEvent": false,
					"isTrainingEvent": true
				},
				"eventName": "Berlin Marathon",
				"eventPrivacy": {
					"label": "PRIVATE"
				},
				"eventTimeLocal": {
					"startTimeHhMm": "09:15",
					"timeZoneId": "Europe/Berlin"
				},
				"eventType": "running",
				"location": "Berlin, Germany",
				"note": "My first marathon",
				"race": true,
				"url": "https://example.com"
			}`,
		},
		{
			name: "primary without goal, time without zone, miles",
			cmd:  EventsAddCmd{Name: "Ten", Date: "2026-05-02", Type: "trail_running", Time: "07:00", Distance: mustDistance(t, "10mi"), Primary: true},
			want: `{
				"completionTarget": {
					"unit": "mile",
					"unitType": "distance",
					"value": 10
				},
				"date": "2026-05-02",
				"eventCustomization": {
					"isPrimaryEvent": true,
					"isTrainingEvent": false
				},
				"eventName": "Ten",
				"eventTimeLocal": {
					"startTimeHhMm": "07:00"
				},
				"eventType": "trail_running",
				"race": false
			}`,
		},
		{
			name: "goal without priority, meters",
			cmd:  EventsAddCmd{Name: "Track", Date: "2026-06-03", Type: "running", Distance: mustDistance(t, "400 m"), Goal: mustGoal(t, "75s")},
			want: `{
				"completionTarget": {
					"unit": "meter",
					"unitType": "distance",
					"value": 400
				},
				"date": "2026-06-03",
				"eventCustomization": {
					"customGoal": {
						"unit": "second",
						"unitType": "time",
						"value": 75
					},
					"isPrimaryEvent": false,
					"isTrainingEvent": false
				},
				"eventName": "Track",
				"eventType": "running",
				"race": false
			}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sentBody(t, "POST /calendar-service/event", &tt.cmd); got != compactJSON(t, tt.want) {
				t.Errorf("body\n got %s\nwant %s", got, tt.want)
			}
		})
	}
}
