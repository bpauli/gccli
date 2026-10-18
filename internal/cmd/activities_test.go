package cmd

import (
	"bytes"
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
		_, _ = w.Write([]byte(`{"totalCount":42}`))
	})

	mux.HandleFunc("/activity-service/activity/activityTypes", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"typeId":1,"typeKey":"running","parentTypeId":17,"isHidden":false,"restricted":false,"trimmable":true},
			{"typeId":2,"typeKey":"cycling","parentTypeId":17,"isHidden":false,"restricted":false,"trimmable":true},
			{"typeId":181,"typeKey":"ultra_run","parentTypeId":1,"isHidden":false,"restricted":false,"trimmable":true}
		]`))
	})

	return httptest.NewServer(mux)
}

func overrideNewClient(t *testing.T, server *httptest.Server) {
	t.Helper()
	orig := newClientFn
	newClientFn = func(tokens *garminauth.Tokens) *garminapi.Client {
		return garminapi.NewClient(tokens, garminapi.WithHTTPClient(server.Client()), garminapi.WithBaseURL(server.URL))
	}
	t.Cleanup(func() { newClientFn = orig })
}

// --- Execute-level tests ---

// --- ActivitiesCountCmd tests ---

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

func TestActivitiesCount_WritesToGlobalsStdout(t *testing.T) {
	server := activitiesTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	for _, tc := range []struct {
		mode outfmt.Mode
		want string
	}{
		{outfmt.Table, "42\n"},
		{outfmt.JSON, "{\n  \"count\": 42\n}\n"},
	} {
		var uiBuf, out bytes.Buffer
		g := testGlobals(t, &uiBuf, tc.mode, "test@example.com")
		g.Stdout = &out
		var err error
		leaked := captureStdout(t, func() { err = (&ActivitiesCountCmd{}).Run(g) })
		if err != nil {
			t.Fatalf("mode %v: unexpected error: %v", tc.mode, err)
		}
		if out.String() != tc.want {
			t.Errorf("mode %v: g.Stdout = %q, want %q", tc.mode, out.String(), tc.want)
		}
		if leaked != "" {
			t.Errorf("mode %v: os.Stdout got %q, want nothing", tc.mode, leaked)
		}
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

func TestValueAt(t *testing.T) {
	tests := []struct {
		name string
		data map[string]any
		want string
	}{
		{"with type", map[string]any{"activityType": map[string]any{"typeKey": "running"}}, "running"},
		{"nil type", map[string]any{}, ""},
		{"missing typeKey", map[string]any{"activityType": map[string]any{}}, ""},
		{"non-map type", map[string]any{"activityType": "invalid"}, ""},
		{"null type", map[string]any{"activityType": nil}, ""},
		{"number key", map[string]any{"activityType": map[string]any{"typeKey": 7.0}}, "7"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := textOf(valueAt(tt.data, "activityType.typeKey"))
			if got != tt.want {
				t.Errorf("valueAt() = %q, want %q", got, tt.want)
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

// --- ActivitiesTypesCmd tests ---

func TestActivitiesTypes_Table(t *testing.T) {
	server := activitiesTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &ActivitiesTypesCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestActivitiesTypes_JSON(t *testing.T) {
	server := activitiesTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &ActivitiesTypesCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	// JSON output goes to stdout; just verify no error.
}

func TestParseActivityTypes_SortsByTypeID(t *testing.T) {
	data := []byte(`[
		{"typeId":181,"typeKey":"ultra_run"},
		{"typeId":1,"typeKey":"running"},
		{"typeId":17,"typeKey":"other"}
	]`)
	types, err := parseActivityTypes(data)
	if err != nil {
		t.Fatalf("parseActivityTypes: %v", err)
	}
	want := []string{"running", "other", "ultra_run"}
	for i, w := range want {
		if got := jsonString(types[i], "typeKey"); got != w {
			t.Errorf("types[%d].typeKey = %q, want %q", i, got, w)
		}
	}
}

func TestParseActivityTypes_InvalidJSON(t *testing.T) {
	_, err := parseActivityTypes([]byte("not-json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestFormatActivityTypeRows(t *testing.T) {
	types := []map[string]any{
		{
			"typeId":       float64(1),
			"typeKey":      "running",
			"parentTypeId": float64(17),
			"trimmable":    true,
			"isHidden":     false,
			"restricted":   false,
		},
		{
			// parentTypeId null, trimmable missing, hidden true.
			"typeId":       float64(17),
			"typeKey":      "other",
			"parentTypeId": nil,
			"isHidden":     true,
			"restricted":   false,
		},
	}
	rows := formatActivityTypeRows(types)
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(rows))
	}
	want := [][]string{
		{"1", "running", "17", "yes", "no", "no"},
		{"17", "other", "-", "-", "yes", "no"},
	}
	for i, row := range rows {
		if len(row) != len(want[i]) {
			t.Fatalf("row[%d] has %d cols, want %d", i, len(row), len(want[i]))
		}
		for j, cell := range row {
			if cell != want[i][j] {
				t.Errorf("row[%d][%d] = %q, want %q", i, j, cell, want[i][j])
			}
		}
	}
}

func TestJsonBool(t *testing.T) {
	m := map[string]any{
		"yes":     true,
		"no":      false,
		"null":    nil,
		"notbool": "true",
	}
	cases := []struct {
		key     string
		wantVal bool
		wantOK  bool
	}{
		{"yes", true, true},
		{"no", false, true},
		{"null", false, false},
		{"missing", false, false},
		{"notbool", false, false},
	}
	for _, tc := range cases {
		v, ok := jsonBool(m, tc.key)
		if v != tc.wantVal || ok != tc.wantOK {
			t.Errorf("jsonBool(%q) = (%v, %v), want (%v, %v)", tc.key, v, ok, tc.wantVal, tc.wantOK)
		}
	}
}
