package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bpauli/gccli/internal/config"
	"github.com/bpauli/gccli/internal/outfmt"
)

// sampleActivityJSON returns a JSON object for a single activity.
func sampleActivityJSON() string {
	return `{
		"activityId": 12345678,
		"activityName": "Morning Run",
		"activityTypeDTO": {"typeId": 1, "typeKey": "running", "parentTypeId": 17},
		"summaryDTO": {
			"startTimeLocal": "2024-06-15 07:30:00",
			"distance": 5123.45,
			"duration": 1800.5,
			"averageSpeed": 2.847,
			"elevationGain": 85.0,
			"averageHR": 145.0,
			"maxHR": 172.0,
			"calories": 350.0
		}
	}`
}

func overrideReadConfig(t *testing.T, cfg *config.File) {
	t.Helper()
	orig := readConfigFn
	readConfigFn = func() (*config.File, error) { return cfg, nil }
	t.Cleanup(func() { readConfigFn = orig })
}

func activityTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/activity-service/activity/12345678", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/details"):
			_, _ = w.Write([]byte(`{"activityId":12345678,"metrics":[]}`))
		case strings.HasSuffix(r.URL.Path, "/splits"):
			_, _ = w.Write([]byte(`{"lapDTOs":[]}`))
		case strings.HasSuffix(r.URL.Path, "/typedsplits"):
			_, _ = w.Write([]byte(`{"typedSplits":[]}`))
		case strings.HasSuffix(r.URL.Path, "/split_summaries"):
			_, _ = w.Write([]byte(`{"splitSummaries":[]}`))
		case strings.HasSuffix(r.URL.Path, "/weather"):
			_, _ = w.Write([]byte(`{"temperature":22,"weatherTypeDTO":{"desc":"Sunny"}}`))
		case strings.HasSuffix(r.URL.Path, "/hrTimeInZones"):
			_, _ = w.Write([]byte(`[{"zoneLowBoundary":0,"secsInZone":120}]`))
		case strings.HasSuffix(r.URL.Path, "/powerTimeInZones"):
			_, _ = w.Write([]byte(`[{"zoneLowBoundary":0,"secsInZone":60}]`))
		case strings.HasSuffix(r.URL.Path, "/exerciseSets"):
			_, _ = w.Write([]byte(`{"exerciseSets":[]}`))
		default:
			_, _ = w.Write([]byte(sampleActivityJSON()))
		}
	})

	// Details endpoint has query parameters in path.
	mux.HandleFunc("/activity-service/activity/12345678/details", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"activityId":12345678,"metrics":[]}`))
	})

	mux.HandleFunc("/activity-service/activity/12345678/splits", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"lapDTOs":[]}`))
	})

	mux.HandleFunc("/activity-service/activity/12345678/typedsplits", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"typedSplits":[]}`))
	})

	mux.HandleFunc("/activity-service/activity/12345678/split_summaries", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"splitSummaries":[]}`))
	})

	mux.HandleFunc("/activity-service/activity/12345678/weather", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"temperature":22,"weatherTypeDTO":{"desc":"Sunny"}}`))
	})

	mux.HandleFunc("/activity-service/activity/12345678/hrTimeInZones", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"zoneLowBoundary":0,"secsInZone":120}]`))
	})

	mux.HandleFunc("/activity-service/activity/12345678/powerTimeInZones", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"zoneLowBoundary":0,"secsInZone":60}]`))
	})

	mux.HandleFunc("/activity-service/activity/12345678/exerciseSets", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"exerciseSets":[]}`))
	})

	mux.HandleFunc("/gear-service/gear", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("activityId") != "12345678" {
			http.Error(w, "bad activityId", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"uuid":"gear-123","displayName":"Running Shoes"}]`))
	})

	return httptest.NewServer(mux)
}

// --- Execute-level tests ---

func TestExecute_ActivityHelp(t *testing.T) {
	code := Execute([]string{"activity", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_ActivityDetailsHelp(t *testing.T) {
	code := Execute([]string{"activity", "details", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_ActivitySplitsHelp(t *testing.T) {
	code := Execute([]string{"activity", "splits", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_ActivityWeatherHelp(t *testing.T) {
	code := Execute([]string{"activity", "weather", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_ActivityHRZonesHelp(t *testing.T) {
	code := Execute([]string{"activity", "hr-zones", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_ActivityGearHelp(t *testing.T) {
	code := Execute([]string{"activity", "gear", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

// --- ActivitySummaryCmd tests ---

func TestActivitySummary_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &ActivitySummaryCmd{ID: "12345678"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestActivitySummary_NotFound(t *testing.T) {
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "nobody@example.com")
	cmd := &ActivitySummaryCmd{ID: "12345678"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no credentials stored") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestActivitySummary_Table(t *testing.T) {
	server := activityTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	overrideReadConfig(t, &config.File{})
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &ActivitySummaryCmd{ID: "12345678"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestActivitySummary_JSON(t *testing.T) {
	server := activityTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	overrideReadConfig(t, &config.File{})
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &ActivitySummaryCmd{ID: "12345678"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestActivitySummary_Plain(t *testing.T) {
	server := activityTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	overrideReadConfig(t, &config.File{})
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Plain, "test@example.com")
	cmd := &ActivitySummaryCmd{ID: "12345678"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// --- ActivityDetailsCmd tests ---

func TestActivityDetails_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &ActivityDetailsCmd{ID: "12345678", MaxChart: 2000, MaxPoly: 4000}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestActivityDetails_Success(t *testing.T) {
	server := activityTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &ActivityDetailsCmd{ID: "12345678", MaxChart: 2000, MaxPoly: 4000}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestActivityDetails_QueryParams(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/activity-service/activity/12345678/details", func(w http.ResponseWriter, r *http.Request) {
		called = true
		if got := r.URL.Query().Get("maxChartSize"); got != "500" {
			t.Errorf("expected maxChartSize=500, got %q", got)
		}
		if got := r.URL.Query().Get("maxPolylineSize"); got != "1000" {
			t.Errorf("expected maxPolylineSize=1000, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"activityId":12345678}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &ActivityDetailsCmd{ID: "12345678", MaxChart: 500, MaxPoly: 1000}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !called {
		t.Fatal("expected API to be called")
	}
}

// --- Sub-resource command tests ---

func TestActivitySplits_Success(t *testing.T) {
	server := activityTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &ActivitySplitsCmd{ID: "12345678"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestActivityTypedSplits_Success(t *testing.T) {
	server := activityTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &ActivityTypedSplitsCmd{ID: "12345678"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestActivitySplitSummaries_Success(t *testing.T) {
	server := activityTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &ActivitySplitSummariesCmd{ID: "12345678"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestActivityWeather_Success(t *testing.T) {
	server := activityTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &ActivityWeatherCmd{ID: "12345678"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestActivityHRZones_Success(t *testing.T) {
	server := activityTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &ActivityHRZonesCmd{ID: "12345678"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestActivityPowerZones_Success(t *testing.T) {
	server := activityTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &ActivityPowerZonesCmd{ID: "12345678"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestActivityExerciseSets_Success(t *testing.T) {
	server := activityTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &ActivityExerciseSetsCmd{ID: "12345678"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestActivityGear_Success(t *testing.T) {
	server := activityTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &ActivityGearCmd{ID: "12345678"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestActivityGear_VerifyActivityID(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/gear-service/gear", func(w http.ResponseWriter, r *http.Request) {
		called = true
		if got := r.URL.Query().Get("activityId"); got != "99999" {
			t.Errorf("expected activityId=99999, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &ActivityGearCmd{ID: "99999"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !called {
		t.Fatal("expected API to be called")
	}
}

// --- formatActivitySummary tests ---

func TestFormatActivitySummary(t *testing.T) {
	activity := map[string]any{
		"activityName":    "Morning Run",
		"activityTypeDTO": map[string]any{"typeKey": "running", "parentTypeId": float64(17)},
		"summaryDTO": map[string]any{
			"startTimeLocal": "2024-06-15 07:30:00",
			"distance":       float64(5123.45),
			"duration":       float64(1800),
			"averageSpeed":   float64(2.847),
			"elevationGain":  float64(85),
			"averageHR":      float64(145),
			"maxHR":          float64(172),
			"calories":       float64(350),
		},
	}

	// Running category default fields: distance, duration, avg_pace, elevation_gain, avg_hr
	rows := formatActivitySummary(activity, nil)
	if len(rows) != 8 {
		t.Fatalf("expected 8 rows (3 fixed + 5 fields), got %d", len(rows))
	}

	expected := []struct {
		label string
		value string
	}{
		{"NAME", "Morning Run"},
		{"TYPE", "running"},
		{"DATE", "2024-06-15"},
		{"DISTANCE", "5.12 km"},
		{"DURATION", "30:00"},
		{"AVG PACE", "5:51 /km"},
		{"ELEVATION", "85 m"},
		{"AVG HR", "145 bpm"},
	}

	for i, exp := range expected {
		if rows[i][0] != exp.label {
			t.Errorf("row %d: expected label %q, got %q", i, exp.label, rows[i][0])
		}
		if rows[i][1] != exp.value {
			t.Errorf("row %d: expected value %q, got %q", i, exp.value, rows[i][1])
		}
	}
}

func TestFormatActivitySummary_MissingFields(t *testing.T) {
	activity := map[string]any{
		"activityName": "Strength Training",
	}

	// Falls back to "other" category: distance, duration, avg_speed, avg_hr, calories
	rows := formatActivitySummary(activity, nil)
	if len(rows) != 8 {
		t.Fatalf("expected 8 rows (3 fixed + 5 fields), got %d", len(rows))
	}

	// Missing numeric fields should show "-".
	for _, idx := range []int{3, 4, 5, 6, 7} {
		if rows[idx][1] != "-" {
			t.Errorf("row %d (%s): expected '-', got %q", idx, rows[idx][0], rows[idx][1])
		}
	}
}

// --- formatHeartRate tests ---

func TestFormatHeartRate(t *testing.T) {
	tests := []struct {
		hr   float64
		want string
	}{
		{0, "-"},
		{145, "145 bpm"},
		{172.6, "172 bpm"},
		{60, "60 bpm"},
	}
	for _, tt := range tests {
		got := formatHeartRate(tt.hr)
		if got != tt.want {
			t.Errorf("formatHeartRate(%v) = %q, want %q", tt.hr, got, tt.want)
		}
	}
}
