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
	mux.HandleFunc("/metrics-service/metrics/racepredictions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"5k":1200,"10k":2500,"halfMarathon":5400,"marathon":11200}`))
	})
	return httptest.NewServer(mux)
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

// --- HealthEnduranceScore date param verification ---

// --- HealthHillScore date param verification ---
