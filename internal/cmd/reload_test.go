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

func reloadTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/wellness-service/wellness/epoch/request/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"OK"}`))
	})

	return httptest.NewServer(mux)
}

// --- Execute-level tests ---

func TestExecute_ReloadHelp(t *testing.T) {
	code := Execute([]string{"reload", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

// --- ReloadCmd tests ---

func TestReload_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &ReloadCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReload_Success(t *testing.T) {
	server := reloadTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	overrideNowFn(t, time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC))

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &ReloadCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestReload_WithDate(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/wellness-service/wellness/epoch/request/", func(w http.ResponseWriter, r *http.Request) {
		called = true
		if !strings.HasSuffix(r.URL.Path, "/2024-03-20") {
			t.Errorf("path = %s, want suffix /2024-03-20", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"OK"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &ReloadCmd{Date: "2024-03-20"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !called {
		t.Fatal("expected API to be called")
	}
}

func TestReload_InvalidDate(t *testing.T) {
	server := reloadTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &ReloadCmd{Date: "not-a-date"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestReload_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/wellness-service/wellness/epoch/request/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	overrideNowFn(t, time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC))

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &ReloadCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
