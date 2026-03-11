package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bpauli/gccli/internal/outfmt"
)

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
	cmd := &EventsListCmd{StartDate: "2026-03-11", Limit: 20, Page: 1, Sort: "eventDate_asc"}
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
	cmd := &EventsListCmd{StartDate: "2026-03-11", Limit: 20, Page: 1, Sort: "eventDate_asc"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestEventsList_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &EventsListCmd{Limit: 20, Page: 1, Sort: "eventDate_asc"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestParseEvents(t *testing.T) {
	data := []byte(`[{"id":1,"eventName":"Race"},{"id":2,"eventName":"Ride"}]`)
	events, err := parseEvents(data)
	if err != nil {
		t.Fatalf("parseEvents: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("got %d events, want 2", len(events))
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

func TestParseEvents_Invalid(t *testing.T) {
	_, err := parseEvents(json.RawMessage(`not json`))
	if err == nil {
		t.Fatal("expected error")
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
		Distance: "42.195km",
		Goal:     "3h30m",
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

func TestEventsAdd_JSON(t *testing.T) {
	server := eventsTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
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

func TestEventsAdd_InvalidDistance(t *testing.T) {
	server := eventsTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &EventsAddCmd{Name: "Race", Date: "2026-04-01", Type: "running", Distance: "abc"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestEventsAdd_InvalidGoal(t *testing.T) {
	server := eventsTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &EventsAddCmd{Name: "Race", Date: "2026-04-01", Type: "running", Goal: "notaduration"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestEventsAdd_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &EventsAddCmd{Name: "Race", Date: "2026-04-01", Type: "running"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestParseDistance(t *testing.T) {
	tests := []struct {
		input    string
		wantVal  float64
		wantUnit string
		wantErr  bool
	}{
		{"10km", 10, "kilometer", false},
		{"42.195km", 42.195, "kilometer", false},
		{"26.2mi", 26.2, "mile", false},
		{"400m", 400, "meter", false},
		{"abc", 0, "", true},
		{"10", 0, "", true},
		{"km", 0, "", true},
	}
	for _, tt := range tests {
		val, unit, err := parseDistance(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseDistance(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if val != tt.wantVal || unit != tt.wantUnit {
			t.Errorf("parseDistance(%q) = (%v, %q), want (%v, %q)", tt.input, val, unit, tt.wantVal, tt.wantUnit)
		}
	}
}

func TestParseGoalDuration(t *testing.T) {
	tests := []struct {
		input   string
		want    int
		wantErr bool
	}{
		{"50m", 3000, false},
		{"1h30m", 5400, false},
		{"2400s", 2400, false},
		{"3h30m0s", 12600, false},
		{"notaduration", 0, true},
	}
	for _, tt := range tests {
		got, err := parseGoalDuration(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseGoalDuration(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("parseGoalDuration(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

// --- EventsDeleteCmd tests ---

func TestEventsDelete_Success(t *testing.T) {
	server := eventsTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &EventsDeleteCmd{ID: "99999", Force: true}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "Deleted event") {
		t.Fatalf("expected 'Deleted event' message, got: %q", buf.String())
	}
}

func TestEventsDelete_Cancelled(t *testing.T) {
	server := eventsTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	orig := confirmReader
	confirmReader = strings.NewReader("n\n")
	t.Cleanup(func() { confirmReader = orig })

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &EventsDeleteCmd{ID: "99999"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "Cancelled") {
		t.Fatalf("expected 'Cancelled' message, got: %q", buf.String())
	}
}

func TestEventsDelete_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &EventsDeleteCmd{ID: "99999", Force: true}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestEventsDelete_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/calendar-service/event/", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &EventsDeleteCmd{ID: "99999", Force: true}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "delete event") {
		t.Fatalf("unexpected error: %v", err)
	}
}
