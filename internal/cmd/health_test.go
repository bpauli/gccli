package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bpauli/gccli/internal/outfmt"
)

func healthTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	serveDisplayName(mux)

	mux.HandleFunc("/usersummary-service/usersummary/daily/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"totalSteps":8500,"totalDistanceMeters":6200}`))
	})

	mux.HandleFunc("/wellness-service/wellness/dailySummaryChart/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"startGMT":"2024-06-15","steps":8500}]`))
	})

	mux.HandleFunc("/usersummary-service/stats/steps/daily/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"calendarDate":"2024-06-15","totalSteps":8500}]`))
	})

	mux.HandleFunc("/usersummary-service/stats/steps/weekly/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"calendarDate":"2024-06-15","totalSteps":59500}]`))
	})

	mux.HandleFunc("/wellness-service/wellness/dailyHeartRate/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"restingHeartRate":62,"maxHeartRate":165}`))
	})

	mux.HandleFunc("/userstats-service/wellness/daily/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"restingHeartRate":62}]`))
	})

	mux.HandleFunc("/wellness-service/wellness/floorsChartData/daily/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"floorsTaken":12}]`))
	})

	mux.HandleFunc("/wellness-service/wellness/dailySleepData/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sleepTimeSeconds":28800,"sleepScores":{"overall":{"value":82}}}`))
	})

	mux.HandleFunc("/wellness-service/wellness/dailyStress/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"overallStressLevel":35}`))
	})

	mux.HandleFunc("/usersummary-service/stats/stress/weekly/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"calendarDate":"2024-06-15","averageStressLevel":30}]`))
	})

	mux.HandleFunc("/wellness-service/wellness/daily/respiration/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"avgWakingRespirationValue":16.5}`))
	})

	return httptest.NewServer(mux)
}

func overrideNowFn(t *testing.T, fixed time.Time) {
	t.Helper()
	orig := nowFn
	nowFn = func() time.Time { return fixed }
	t.Cleanup(func() { nowFn = orig })
}

// --- Execute-level tests ---

// --- HealthSummaryCmd tests ---

func TestHealthSummary_Success(t *testing.T) {
	server := healthTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &HealthSummaryCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestHealthSummary_WithDate(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	serveDisplayName(mux)
	mux.HandleFunc("/usersummary-service/usersummary/daily/", func(w http.ResponseWriter, r *http.Request) {
		called = true
		if !strings.Contains(r.URL.String(), "calendarDate=2024-06-15") {
			t.Errorf("expected calendarDate=2024-06-15, got %q", r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"totalSteps":8500}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &HealthSummaryCmd{Date: "2024-06-15"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !called {
		t.Fatal("expected API to be called")
	}
}

// --- HealthStepsCmd tests ---

func TestHealthSteps_Success(t *testing.T) {
	server := healthTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &HealthStepsViewCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestHealthStepsDaily_Success(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/usersummary-service/stats/steps/daily/", func(w http.ResponseWriter, r *http.Request) {
		called = true
		path := r.URL.Path
		if !strings.Contains(path, "2024-06-01") || !strings.Contains(path, "2024-06-30") {
			t.Errorf("expected path with date range, got %q", path)
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
	cmd := &HealthStepsDailyCmd{StartDate: "2024-06-01", EndDate: "2024-06-30"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !called {
		t.Fatal("expected API to be called")
	}
}

func TestHealthStepsWeekly_Success(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/usersummary-service/stats/steps/weekly/", func(w http.ResponseWriter, r *http.Request) {
		called = true
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
	cmd := &HealthStepsWeeklyCmd{Weeks: 4}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !called {
		t.Fatal("expected API to be called")
	}
}

func TestHealthWeeklySendsWeeksAsDecimal(t *testing.T) {
	tests := []struct {
		cmd  interface{ Run(*Globals) error }
		want string
	}{
		{&HealthStepsWeeklyCmd{EndDate: "2024-01-31", Weeks: 12}, "/usersummary-service/stats/steps/weekly/2024-01-31/12"},
		{&HealthStepsWeeklyCmd{EndDate: "2024-01-31", Weeks: 0}, "/usersummary-service/stats/steps/weekly/2024-01-31/0"},
		{&HealthStressWeeklyCmd{EndDate: "2024-01-31", Weeks: -1}, "/usersummary-service/stats/stress/weekly/2024-01-31/-1"},
	}
	for _, tc := range tests {
		var got string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got = r.URL.Path
			_, _ = w.Write([]byte(`[]`))
		}))
		store := newTestSecretsStore(t)
		overrideLoadSecrets(t, store)
		overrideNewClient(t, server)
		storeTestTokens(t, store, "test@example.com", testTokens())

		var buf bytes.Buffer
		if err := tc.cmd.Run(testGlobals(t, &buf, outfmt.JSON, "test@example.com")); err != nil {
			t.Fatalf("Run: %v", err)
		}
		server.Close()
		if got != tc.want {
			t.Errorf("path = %q, want %q", got, tc.want)
		}
	}
}

func TestHealthStepsWeekly_DefaultEndDate(t *testing.T) {
	fixed := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	overrideNowFn(t, fixed)

	var capturedPath string
	mux := http.NewServeMux()
	mux.HandleFunc("/usersummary-service/stats/steps/weekly/", func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
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
	cmd := &HealthStepsWeeklyCmd{Weeks: 4}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(capturedPath, "2024-06-15") {
		t.Errorf("expected path to contain today's date 2024-06-15, got %q", capturedPath)
	}
}

// --- HealthHRCmd tests ---

func TestHealthHR_Success(t *testing.T) {
	server := healthTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &HealthHRCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// --- HealthRHRCmd tests ---

func TestHealthRHR_Success(t *testing.T) {
	server := healthTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &HealthRHRCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// --- HealthFloorsCmd tests ---

func TestHealthFloors_Success(t *testing.T) {
	server := healthTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &HealthFloorsCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// --- HealthSleepCmd tests ---

func TestHealthSleep_Success(t *testing.T) {
	server := healthTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &HealthSleepCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// --- HealthStressCmd tests ---

func TestHealthStress_Success(t *testing.T) {
	server := healthTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &HealthStressViewCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestHealthStressWeekly_Success(t *testing.T) {
	server := healthTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &HealthStressWeeklyCmd{Weeks: 4}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestHealthStressWeekly_DefaultEndDate(t *testing.T) {
	fixed := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	overrideNowFn(t, fixed)

	var capturedPath string
	mux := http.NewServeMux()
	mux.HandleFunc("/usersummary-service/stats/stress/weekly/", func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
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
	cmd := &HealthStressWeeklyCmd{Weeks: 4}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(capturedPath, "2024-06-15") {
		t.Errorf("expected path to contain today's date 2024-06-15, got %q", capturedPath)
	}
}

// --- HealthRespirationCmd tests ---

func TestHealthRespiration_Success(t *testing.T) {
	server := healthTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &HealthRespirationCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// --- resolveDate tests ---

func TestResolveDate(t *testing.T) {
	fixed := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	overrideNowFn(t, fixed)

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"empty defaults to today", "", "2024-06-15", false},
		{"today", "today", "2024-06-15", false},
		{"Today (case insensitive)", "Today", "2024-06-15", false},
		{"yesterday", "yesterday", "2024-06-14", false},
		{"Yesterday (case insensitive)", "Yesterday", "2024-06-14", false},
		{"3d relative", "3d", "2024-06-12", false},
		{"7d relative", "7d", "2024-06-08", false},
		{"1d relative", "1d", "2024-06-14", false},
		{"0d relative", "0d", "2024-06-15", false},
		{"+7d future", "+7d", "2024-06-22", false},
		{"+365d future", "+365d", "2025-06-15", false},
		{"+0d future", "+0d", "2024-06-15", false},
		{"explicit date", "2024-01-01", "2024-01-01", false},
		{"whitespace trimmed", "  2024-06-15  ", "2024-06-15", false},
		{"invalid date", "not-a-date", "", true},
		{"invalid format", "15/06/2024", "", true},
		{"partial date", "2024-06", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveDate(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if got != tt.want {
				t.Errorf("resolveDate(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveDate_ErrorMessage(t *testing.T) {
	_, err := resolveDate("bad-date")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "YYYY-MM-DD") {
		t.Errorf("expected error to mention YYYY-MM-DD format, got: %v", err)
	}
	if !strings.Contains(err.Error(), "3d") {
		t.Errorf("expected error to mention relative date example, got: %v", err)
	}
}

// --- writeHealthJSON tests ---

func TestWriteHealthJSON_NilData(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	err := writeHealthJSON(g, nil)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// --- Server error test ---

func TestHealthSummary_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	serveDisplayName(mux)
	mux.HandleFunc("/usersummary-service/usersummary/daily/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal server error"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &HealthSummaryCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "get daily summary") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Invalid date test ---

func TestHealthSummary_InvalidDate(t *testing.T) {
	server := healthTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &HealthSummaryCmd{Date: "not-a-date"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid date") {
		t.Fatalf("unexpected error: %v", err)
	}
}
