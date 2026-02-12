package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bpauli/gccli/internal/outfmt"
)

func wellnessTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/periodichealth-service/menstrualcycle/dayview/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"cycleDayEntries":[]}`))
	})

	mux.HandleFunc("/periodichealth-service/menstrualcycle/summary/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"averageCycleLength":28}`))
	})

	mux.HandleFunc("/periodichealth-service/pregnancy/summary", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"pregnancyStatus":"active"}`))
	})

	return httptest.NewServer(mux)
}

// --- Execute-level tests ---

func TestExecute_WellnessHelp(t *testing.T) {
	code := Execute([]string{"wellness", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_WellnessMenstrualCycleHelp(t *testing.T) {
	code := Execute([]string{"wellness", "menstrual-cycle", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_WellnessMenstrualSummaryHelp(t *testing.T) {
	code := Execute([]string{"wellness", "menstrual-summary", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_WellnessPregnancySummaryHelp(t *testing.T) {
	code := Execute([]string{"wellness", "pregnancy-summary", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

// --- WellnessMenstrualCycleCmd tests ---

func TestWellnessMenstrualCycle_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &WellnessMenstrualCycleCmd{StartDate: "2024-01-01", EndDate: "2024-01-31"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWellnessMenstrualCycle_Success(t *testing.T) {
	server := wellnessTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &WellnessMenstrualCycleCmd{StartDate: "2024-01-01", EndDate: "2024-01-31"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestWellnessMenstrualCycle_PathVerification(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/periodichealth-service/menstrualcycle/dayview/", func(w http.ResponseWriter, r *http.Request) {
		called = true
		if !strings.HasSuffix(r.URL.Path, "/2024-02-01/2024-02-28") {
			t.Errorf("path = %s, want suffix /2024-02-01/2024-02-28", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"cycleDayEntries":[]}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &WellnessMenstrualCycleCmd{StartDate: "2024-02-01", EndDate: "2024-02-28"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !called {
		t.Fatal("expected API to be called")
	}
}

func TestWellnessMenstrualCycle_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/periodichealth-service/menstrualcycle/dayview/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &WellnessMenstrualCycleCmd{StartDate: "2024-01-01", EndDate: "2024-01-31"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- WellnessMenstrualSummaryCmd tests ---

func TestWellnessMenstrualSummary_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &WellnessMenstrualSummaryCmd{StartDate: "2024-01-01", EndDate: "2024-01-31"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWellnessMenstrualSummary_Success(t *testing.T) {
	server := wellnessTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &WellnessMenstrualSummaryCmd{StartDate: "2024-01-01", EndDate: "2024-01-31"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// --- WellnessPregnancySummaryCmd tests ---

func TestWellnessPregnancySummary_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &WellnessPregnancySummaryCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWellnessPregnancySummary_Success(t *testing.T) {
	server := wellnessTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &WellnessPregnancySummaryCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestWellnessPregnancySummary_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/periodichealth-service/pregnancy/summary", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &WellnessPregnancySummaryCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
