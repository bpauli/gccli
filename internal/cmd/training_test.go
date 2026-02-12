package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bpauli/gccli/internal/outfmt"
)

func trainingTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/trainingplan-service/trainingplan/plans", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"planId":"plan-1","planName":"5K Beginner"}]`))
	})

	mux.HandleFunc("/trainingplan-service/trainingplan/plan/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"planId":"plan-1","planName":"5K Beginner","description":"A beginner plan"}`))
	})

	return httptest.NewServer(mux)
}

// --- Execute-level tests ---

func TestExecute_TrainingHelp(t *testing.T) {
	code := Execute([]string{"training", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_TrainingPlansHelp(t *testing.T) {
	code := Execute([]string{"training", "plans", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_TrainingPlanHelp(t *testing.T) {
	code := Execute([]string{"training", "plan", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

// --- TrainingPlansCmd tests ---

func TestTrainingPlans_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &TrainingPlansCmd{Locale: "en"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTrainingPlans_Success(t *testing.T) {
	server := trainingTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &TrainingPlansCmd{Locale: "en"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestTrainingPlans_WithLocale(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/trainingplan-service/trainingplan/plans", func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.URL.Query().Get("locale") != "de" {
			t.Errorf("locale = %s, want de", r.URL.Query().Get("locale"))
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
	cmd := &TrainingPlansCmd{Locale: "de"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !called {
		t.Fatal("expected API to be called")
	}
}

func TestTrainingPlans_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/trainingplan-service/trainingplan/plans", func(w http.ResponseWriter, _ *http.Request) {
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
	cmd := &TrainingPlansCmd{Locale: "en"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- TrainingPlanCmd tests ---

func TestTrainingPlan_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &TrainingPlanCmd{ID: "plan-1"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTrainingPlan_Success(t *testing.T) {
	server := trainingTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &TrainingPlanCmd{ID: "plan-1"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestTrainingPlan_PathVerification(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/trainingplan-service/trainingplan/plan/", func(w http.ResponseWriter, r *http.Request) {
		called = true
		wantSuffix := "/plan-xyz"
		if !strings.HasSuffix(r.URL.Path, wantSuffix) {
			t.Errorf("path = %s, want suffix %s", r.URL.Path, wantSuffix)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"planId":"plan-xyz"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &TrainingPlanCmd{ID: "plan-xyz"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !called {
		t.Fatal("expected API to be called")
	}
}
