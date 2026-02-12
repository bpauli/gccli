package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bpauli/gccli/internal/outfmt"
)

func healthAdvancedTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/wellness-service/wellness/daily/spo2/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"averageSPO2":96}`))
	})

	mux.HandleFunc("/hrv-service/hrv/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"weeklyAvg":42,"lastNightAvg":38}`))
	})

	mux.HandleFunc("/wellness-service/wellness/bodyBattery/reports/daily", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"date":"2024-06-15","charged":65,"drained":40}]`))
	})

	mux.HandleFunc("/wellness-service/wellness/daily/im/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"moderateIntensityMinutes":30,"vigorousIntensityMinutes":15}`))
	})

	mux.HandleFunc("/usersummary-service/stats/im/weekly/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"weeklyModerate":150,"weeklyVigorous":75}]`))
	})

	mux.HandleFunc("/metrics-service/metrics/trainingreadiness/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"score":72,"level":"MODERATE"}`))
	})

	mux.HandleFunc("/metrics-service/metrics/trainingstatus/aggregated/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"trainingStatus":"PRODUCTIVE"}`))
	})

	mux.HandleFunc("/fitnessage-service/fitnessage/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"fitnessAge":32}`))
	})

	mux.HandleFunc("/metrics-service/metrics/maxmet/daily/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"vo2MaxValue":48.5}]`))
	})

	mux.HandleFunc("/biometric-service/biometric/latestLactateThreshold", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"lactateThreshold":165}`))
	})

	mux.HandleFunc("/biometric-service/biometric/latestFunctionalThresholdPower/CYCLING", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"functionalThresholdPower":250}`))
	})

	mux.HandleFunc("/metrics-service/metrics/racepredictions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"5k":1200,"10k":2500,"halfMarathon":5400,"marathon":11200}`))
	})

	mux.HandleFunc("/metrics-service/metrics/endurancescore", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"enduranceScore":65}`))
	})

	mux.HandleFunc("/metrics-service/metrics/hillscore", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hillScore":58}`))
	})

	mux.HandleFunc("/wellness-service/wellness/dailyEvents", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"eventType":"SLEEP","startTimestamp":1718409600000}]`))
	})

	mux.HandleFunc("/lifestylelogging-service/dailyLog/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"totalCalories":2100}`))
	})

	return httptest.NewServer(mux)
}

// --- Execute-level help tests ---

func TestExecute_HealthSPO2Help(t *testing.T) {
	code := Execute([]string{"health", "spo2", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_HealthHRVHelp(t *testing.T) {
	code := Execute([]string{"health", "hrv", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_HealthBodyBatteryHelp(t *testing.T) {
	code := Execute([]string{"health", "body-battery", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_HealthBodyBatteryRangeHelp(t *testing.T) {
	code := Execute([]string{"health", "body-battery", "range", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_HealthIntensityMinutesHelp(t *testing.T) {
	code := Execute([]string{"health", "intensity-minutes", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_HealthIntensityMinutesWeeklyHelp(t *testing.T) {
	code := Execute([]string{"health", "intensity-minutes", "weekly", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_HealthTrainingReadinessHelp(t *testing.T) {
	code := Execute([]string{"health", "training-readiness", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_HealthTrainingStatusHelp(t *testing.T) {
	code := Execute([]string{"health", "training-status", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_HealthFitnessAgeHelp(t *testing.T) {
	code := Execute([]string{"health", "fitness-age", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_HealthMaxMetricsHelp(t *testing.T) {
	code := Execute([]string{"health", "max-metrics", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_HealthLactateThresholdHelp(t *testing.T) {
	code := Execute([]string{"health", "lactate-threshold", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_HealthCyclingFTPHelp(t *testing.T) {
	code := Execute([]string{"health", "cycling-ftp", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_HealthRacePredictionsHelp(t *testing.T) {
	code := Execute([]string{"health", "race-predictions", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_HealthRacePredictionsRangeHelp(t *testing.T) {
	code := Execute([]string{"health", "race-predictions", "range", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_HealthEnduranceScoreHelp(t *testing.T) {
	code := Execute([]string{"health", "endurance-score", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_HealthHillScoreHelp(t *testing.T) {
	code := Execute([]string{"health", "hill-score", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_HealthEventsHelp(t *testing.T) {
	code := Execute([]string{"health", "events", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_HealthLifestyleHelp(t *testing.T) {
	code := Execute([]string{"health", "lifestyle", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

// --- HealthSPO2Cmd tests ---

func TestHealthSPO2_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &HealthSPO2Cmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHealthSPO2_Success(t *testing.T) {
	server := healthAdvancedTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &HealthSPO2Cmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// --- HealthHRVCmd tests ---

func TestHealthHRV_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &HealthHRVCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestHealthHRV_Success(t *testing.T) {
	server := healthAdvancedTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &HealthHRVCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// --- HealthBodyBatteryCmd tests ---

func TestHealthBodyBattery_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &HealthBodyBatteryViewCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestHealthBodyBattery_Success(t *testing.T) {
	server := healthAdvancedTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &HealthBodyBatteryViewCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestHealthBodyBatteryRange_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &HealthBodyBatteryRangeCmd{StartDate: "2024-06-01", EndDate: "2024-06-15"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestHealthBodyBatteryRange_Success(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/wellness-service/wellness/bodyBattery/reports/daily", func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.URL.Query().Get("startDate") != "2024-06-01" {
			t.Errorf("expected startDate=2024-06-01, got %q", r.URL.Query().Get("startDate"))
		}
		if r.URL.Query().Get("endDate") != "2024-06-15" {
			t.Errorf("expected endDate=2024-06-15, got %q", r.URL.Query().Get("endDate"))
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
	cmd := &HealthBodyBatteryRangeCmd{StartDate: "2024-06-01", EndDate: "2024-06-15"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !called {
		t.Fatal("expected API to be called")
	}
}

// --- HealthIntensityMinutesCmd tests ---

func TestHealthIntensityMinutes_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &HealthIntensityMinutesViewCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestHealthIntensityMinutes_Success(t *testing.T) {
	server := healthAdvancedTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &HealthIntensityMinutesViewCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestHealthIntensityMinutesWeekly_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &HealthIntensityMinutesWeeklyCmd{StartDate: "2024-06-01", EndDate: "2024-06-15"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestHealthIntensityMinutesWeekly_Success(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/usersummary-service/stats/im/weekly/", func(w http.ResponseWriter, r *http.Request) {
		called = true
		path := r.URL.Path
		if !strings.Contains(path, "2024-06-01") || !strings.Contains(path, "2024-06-15") {
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
	cmd := &HealthIntensityMinutesWeeklyCmd{StartDate: "2024-06-01", EndDate: "2024-06-15"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !called {
		t.Fatal("expected API to be called")
	}
}

// --- HealthTrainingReadinessCmd tests ---

func TestHealthTrainingReadiness_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &HealthTrainingReadinessCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestHealthTrainingReadiness_Success(t *testing.T) {
	server := healthAdvancedTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &HealthTrainingReadinessCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// --- HealthTrainingStatusCmd tests ---

func TestHealthTrainingStatus_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &HealthTrainingStatusCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestHealthTrainingStatus_Success(t *testing.T) {
	server := healthAdvancedTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &HealthTrainingStatusCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// --- HealthFitnessAgeCmd tests ---

func TestHealthFitnessAge_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &HealthFitnessAgeCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestHealthFitnessAge_Success(t *testing.T) {
	server := healthAdvancedTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &HealthFitnessAgeCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// --- HealthMaxMetricsCmd tests ---

func TestHealthMaxMetrics_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &HealthMaxMetricsCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestHealthMaxMetrics_Success(t *testing.T) {
	server := healthAdvancedTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &HealthMaxMetricsCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// --- HealthLactateThresholdCmd tests ---

func TestHealthLactateThreshold_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &HealthLactateThresholdCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestHealthLactateThreshold_Success(t *testing.T) {
	server := healthAdvancedTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &HealthLactateThresholdCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// --- HealthCyclingFTPCmd tests ---

func TestHealthCyclingFTP_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &HealthCyclingFTPCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestHealthCyclingFTP_Success(t *testing.T) {
	server := healthAdvancedTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &HealthCyclingFTPCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// --- HealthRacePredictionsCmd tests ---

func TestHealthRacePredictions_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &HealthRacePredictionsViewCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestHealthRacePredictions_Success(t *testing.T) {
	server := healthAdvancedTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &HealthRacePredictionsViewCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestHealthRacePredictionsRange_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &HealthRacePredictionsRangeCmd{StartDate: "2024-06-01", EndDate: "2024-06-15"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestHealthRacePredictionsRange_Success(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics-service/metrics/racepredictions/range/", func(w http.ResponseWriter, r *http.Request) {
		called = true
		path := r.URL.Path
		if !strings.Contains(path, "2024-06-01") || !strings.Contains(path, "2024-06-15") {
			t.Errorf("expected path with date range, got %q", path)
		}
		if r.URL.Query().Get("type") != "daily" {
			t.Errorf("expected type=daily, got %q", r.URL.Query().Get("type"))
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
	cmd := &HealthRacePredictionsRangeCmd{StartDate: "2024-06-01", EndDate: "2024-06-15"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !called {
		t.Fatal("expected API to be called")
	}
}

// --- HealthEnduranceScoreCmd tests ---

func TestHealthEnduranceScore_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &HealthEnduranceScoreCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestHealthEnduranceScore_Success(t *testing.T) {
	server := healthAdvancedTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &HealthEnduranceScoreCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// --- HealthHillScoreCmd tests ---

func TestHealthHillScore_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &HealthHillScoreCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestHealthHillScore_Success(t *testing.T) {
	server := healthAdvancedTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &HealthHillScoreCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// --- HealthEventsCmd tests ---

func TestHealthEvents_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &HealthEventsCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestHealthEvents_Success(t *testing.T) {
	server := healthAdvancedTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &HealthEventsCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestHealthEvents_WithDate(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/wellness-service/wellness/dailyEvents", func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.URL.Query().Get("calendarDate") != "2024-06-15" {
			t.Errorf("expected calendarDate=2024-06-15, got %q", r.URL.Query().Get("calendarDate"))
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
	cmd := &HealthEventsCmd{Date: "2024-06-15"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !called {
		t.Fatal("expected API to be called")
	}
}

// --- HealthLifestyleCmd tests ---

func TestHealthLifestyle_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &HealthLifestyleCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestHealthLifestyle_Success(t *testing.T) {
	server := healthAdvancedTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &HealthLifestyleCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestHealthLifestyle_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/lifestylelogging-service/dailyLog/", func(w http.ResponseWriter, r *http.Request) {
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
	cmd := &HealthLifestyleCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "get lifestyle") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- HealthEnduranceScore date param verification ---

func TestHealthEnduranceScore_DateParam(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics-service/metrics/endurancescore", func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.URL.Query().Get("calendarDate") != "2024-06-15" {
			t.Errorf("expected calendarDate=2024-06-15, got %q", r.URL.Query().Get("calendarDate"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"enduranceScore":65}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &HealthEnduranceScoreCmd{Date: "2024-06-15"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !called {
		t.Fatal("expected API to be called")
	}
}

// --- HealthHillScore date param verification ---

func TestHealthHillScore_DateParam(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics-service/metrics/hillscore", func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.URL.Query().Get("calendarDate") != "2024-06-15" {
			t.Errorf("expected calendarDate=2024-06-15, got %q", r.URL.Query().Get("calendarDate"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hillScore":58}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &HealthHillScoreCmd{Date: "2024-06-15"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !called {
		t.Fatal("expected API to be called")
	}
}
