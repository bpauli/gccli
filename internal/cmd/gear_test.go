package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bpauli/gccli/internal/outfmt"
)

// --- GearLinkCmd tests ---

func TestGearLink_Success(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/gear-service/gear/link/abc-123/activity/100", func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &GearLinkCmd{UUID: "abc-123", ActivityID: "100"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !called {
		t.Fatal("expected API to be called")
	}
	if !strings.Contains(buf.String(), "Linked gear abc-123 to activity 100") {
		t.Errorf("output = %q, want success message", buf.String())
	}
}

func TestGearLink_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/gear-service/gear/link/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &GearLinkCmd{UUID: "abc-123", ActivityID: "100"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- GearUnlinkCmd tests ---

func TestGearUnlink_Success(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/gear-service/gear/unlink/abc-123/activity/100", func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &GearUnlinkCmd{UUID: "abc-123", ActivityID: "100"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !called {
		t.Fatal("expected API to be called")
	}
	if !strings.Contains(buf.String(), "Unlinked gear abc-123 from activity 100") {
		t.Errorf("output = %q, want success message", buf.String())
	}
}

func TestGearUnlink_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/gear-service/gear/unlink/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &GearUnlinkCmd{UUID: "abc-123", ActivityID: "100"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
