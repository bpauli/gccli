package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bpauli/gccli/internal/garminapi"
	"github.com/bpauli/gccli/internal/garminauth"
	"github.com/bpauli/gccli/internal/outfmt"
)

// sampleActivitiesJSON returns a JSON array of sample activities for testing.
func sampleActivitiesJSON() string {
	return `[
		{
			"activityId": 12345678,
			"activityName": "Morning Run",
			"startTimeLocal": "2024-06-15 07:30:00",
			"activityType": {"typeId": 1, "typeKey": "running"},
			"distance": 5123.45,
			"duration": 1800.5,
			"calories": 350.0
		},
		{
			"activityId": 87654321,
			"activityName": "Evening Ride",
			"startTimeLocal": "2024-06-14 18:00:00",
			"activityType": {"typeId": 2, "typeKey": "cycling"},
			"distance": 25000.0,
			"duration": 3661.0,
			"calories": 600.0
		}
	]`
}

func activitiesTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/activitylist-service/activities/search/activities", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(sampleActivitiesJSON()))
	})

	mux.HandleFunc("/activitylist-service/activities/count", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("42"))
	})

	return httptest.NewServer(mux)
}

func overrideNewClient(t *testing.T, server *httptest.Server) {
	t.Helper()
	orig := newClientFn
	newClientFn = func(tokens *garminauth.Tokens) *garminapi.Client {
		return garminapi.NewClient(tokens, garminapi.WithBaseURL(server.URL))
	}
	t.Cleanup(func() { newClientFn = orig })
}

// --- Execute-level tests ---

func TestExecute_ActivitiesHelp(t *testing.T) {
	code := Execute([]string{"activities", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_ActivitiesCountHelp(t *testing.T) {
	code := Execute([]string{"activities", "count", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_ActivitiesSearchHelp(t *testing.T) {
	code := Execute([]string{"activities", "search", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

// --- ActivitiesListCmd tests ---

func TestActivitiesList_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &ActivitiesListCmd{Limit: 20}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestActivitiesList_NotFound(t *testing.T) {
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "nobody@example.com")
	cmd := &ActivitiesListCmd{Limit: 20}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no credentials stored") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestActivitiesList_Table(t *testing.T) {
	server := activitiesTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &ActivitiesListCmd{Limit: 20, Start: 0}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	// Table output goes to stdout, not our buffer; just verify no error.
}

func TestActivitiesList_JSON(t *testing.T) {
	server := activitiesTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &ActivitiesListCmd{Limit: 20, Start: 0}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	// JSON output goes to stdout; just verify no error.
}

func TestActivitiesList_WithType(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/activitylist-service/activities/search/activities", func(w http.ResponseWriter, r *http.Request) {
		called = true
		if got := r.URL.Query().Get("activityType"); got != "running" {
			t.Errorf("expected activityType=running, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &ActivitiesListCmd{Limit: 20, Start: 0, Type: "running"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !called {
		t.Fatal("expected API to be called")
	}
}

// --- ActivitiesCountCmd tests ---

func TestActivitiesCount_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &ActivitiesCountCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestActivitiesCount_Success(t *testing.T) {
	server := activitiesTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &ActivitiesCountCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	// Count is printed to stdout.
}

func TestActivitiesCount_JSON(t *testing.T) {
	server := activitiesTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &ActivitiesCountCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// --- ActivitiesSearchCmd tests ---

func TestActivitiesSearch_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &ActivitiesSearchCmd{StartDate: "2024-01-01"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestActivitiesSearch_Success(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/activitylist-service/activities/search/activities", func(w http.ResponseWriter, r *http.Request) {
		called = true
		if got := r.URL.Query().Get("startDate"); got != "2024-06-01" {
			t.Errorf("expected startDate=2024-06-01, got %q", got)
		}
		if got := r.URL.Query().Get("endDate"); got != "2024-06-30" {
			t.Errorf("expected endDate=2024-06-30, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(sampleActivitiesJSON()))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &ActivitiesSearchCmd{
		Limit:     20,
		Start:     0,
		StartDate: "2024-06-01",
		EndDate:   "2024-06-30",
	}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !called {
		t.Fatal("expected API to be called")
	}
}

func TestActivitiesSearch_JSON(t *testing.T) {
	server := activitiesTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &ActivitiesSearchCmd{
		Limit:     20,
		Start:     0,
		StartDate: "2024-06-01",
		EndDate:   "2024-06-30",
	}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// --- resolveClient tests ---

func TestResolveClient_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	_, err := resolveClient(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveClient_NotFound(t *testing.T) {
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "nobody@example.com")
	_, err := resolveClient(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no credentials stored") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveClient_Success(t *testing.T) {
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	client, err := resolveClient(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.Tokens().Email != "test@example.com" {
		t.Fatalf("expected email test@example.com, got %q", client.Tokens().Email)
	}
}

// --- Formatting helper tests ---

func TestParseActivities(t *testing.T) {
	data := json.RawMessage(sampleActivitiesJSON())
	activities, err := parseActivities(data)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(activities) != 2 {
		t.Fatalf("expected 2 activities, got %d", len(activities))
	}
}

func TestParseActivities_Invalid(t *testing.T) {
	data := json.RawMessage(`{"not": "an array"}`)
	_, err := parseActivities(data)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestParseActivities_Empty(t *testing.T) {
	data := json.RawMessage(`[]`)
	activities, err := parseActivities(data)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(activities) != 0 {
		t.Fatalf("expected 0 activities, got %d", len(activities))
	}
}

func TestFormatActivityRows(t *testing.T) {
	data := json.RawMessage(sampleActivitiesJSON())
	activities, err := parseActivities(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	rows := formatActivityRows(activities)
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	// First row: Morning Run
	row := rows[0]
	if row[0] != "12345678" {
		t.Errorf("expected ID 12345678, got %q", row[0])
	}
	if row[1] != "2024-06-15" {
		t.Errorf("expected date 2024-06-15, got %q", row[1])
	}
	if row[2] != "running" {
		t.Errorf("expected type running, got %q", row[2])
	}
	if row[3] != "Morning Run" {
		t.Errorf("expected name 'Morning Run', got %q", row[3])
	}
	if row[4] != "5.12 km" {
		t.Errorf("expected distance '5.12 km', got %q", row[4])
	}
	if row[5] != "30:00" {
		t.Errorf("expected duration '30:00', got %q", row[5])
	}
	if row[6] != "350" {
		t.Errorf("expected calories '350', got %q", row[6])
	}

	// Second row: Evening Ride
	row = rows[1]
	if row[0] != "87654321" {
		t.Errorf("expected ID 87654321, got %q", row[0])
	}
	if row[5] != "1:01:01" {
		t.Errorf("expected duration '1:01:01', got %q", row[5])
	}
}

func TestFormatDistance(t *testing.T) {
	tests := []struct {
		meters float64
		want   string
	}{
		{0, "-"},
		{1000, "1.00 km"},
		{5123.45, "5.12 km"},
		{42195.0, "42.20 km"},
	}
	for _, tt := range tests {
		got := formatDistance(tt.meters)
		if got != tt.want {
			t.Errorf("formatDistance(%v) = %q, want %q", tt.meters, got, tt.want)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		seconds float64
		want    string
	}{
		{0, "-"},
		{90, "1:30"},
		{3600, "1:00:00"},
		{3661, "1:01:01"},
		{7200, "2:00:00"},
	}
	for _, tt := range tests {
		got := formatDuration(tt.seconds)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.seconds, got, tt.want)
		}
	}
}

func TestFormatCalories(t *testing.T) {
	tests := []struct {
		cal  float64
		want string
	}{
		{0, "-"},
		{350, "350"},
		{600.7, "600"},
	}
	for _, tt := range tests {
		got := formatCalories(tt.cal)
		if got != tt.want {
			t.Errorf("formatCalories(%v) = %q, want %q", tt.cal, got, tt.want)
		}
	}
}

func TestFormatDate(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"2024-06-15 07:30:00", "2024-06-15"},
		{"2024-06-15", "2024-06-15"},
	}
	for _, tt := range tests {
		got := formatDate(tt.input)
		if got != tt.want {
			t.Errorf("formatDate(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestActivityTypeKey(t *testing.T) {
	tests := []struct {
		name string
		data map[string]any
		want string
	}{
		{"with type", map[string]any{"activityType": map[string]any{"typeKey": "running"}}, "running"},
		{"nil type", map[string]any{}, ""},
		{"missing typeKey", map[string]any{"activityType": map[string]any{}}, ""},
		{"non-map type", map[string]any{"activityType": "invalid"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := activityTypeKey(tt.data)
			if got != tt.want {
				t.Errorf("activityTypeKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestJsonString(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]any
		key  string
		want string
	}{
		{"string value", map[string]any{"k": "hello"}, "k", "hello"},
		{"integer float", map[string]any{"k": float64(42)}, "k", "42"},
		{"decimal float", map[string]any{"k": float64(3.14)}, "k", "3.14"},
		{"missing key", map[string]any{}, "k", ""},
		{"nil value", map[string]any{"k": nil}, "k", ""},
		{"bool value", map[string]any{"k": true}, "k", "true"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jsonString(tt.m, tt.key)
			if got != tt.want {
				t.Errorf("jsonString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestJsonFloat(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]any
		key  string
		want float64
	}{
		{"float value", map[string]any{"k": float64(42.5)}, "k", 42.5},
		{"missing key", map[string]any{}, "k", 0},
		{"nil value", map[string]any{"k": nil}, "k", 0},
		{"string value", map[string]any{"k": "not a number"}, "k", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jsonFloat(tt.m, tt.key)
			if got != tt.want {
				t.Errorf("jsonFloat() = %v, want %v", got, tt.want)
			}
		})
	}
}
