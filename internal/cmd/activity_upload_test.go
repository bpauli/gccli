package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bpauli/gccli/internal/outfmt"
)

func uploadTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/upload-service/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		ct := r.Header.Get("Content-Type")
		if !strings.Contains(ct, "multipart/form-data") {
			http.Error(w, "expected multipart/form-data", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"detailedImportResult":{"successes":[{"internalId":99999}]}}`))
	})

	return httptest.NewServer(mux)
}

// --- Execute-level tests ---

func TestExecute_ActivityUploadHelp(t *testing.T) {
	code := Execute([]string{"activity", "upload", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

// --- ActivityUploadCmd tests ---

func TestActivityUpload_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &ActivityUploadCmd{File: "test.fit"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestActivityUpload_FIT(t *testing.T) {
	server := uploadTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	// Create a temp FIT file.
	tmpDir := t.TempDir()
	fitPath := filepath.Join(tmpDir, "test.fit")
	if err := os.WriteFile(fitPath, []byte("FIT-DATA"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &ActivityUploadCmd{File: fitPath}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "Uploaded") {
		t.Fatalf("expected 'Uploaded' message, got: %q", buf.String())
	}
}

func TestActivityUpload_JSON(t *testing.T) {
	server := uploadTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	tmpDir := t.TempDir()
	fitPath := filepath.Join(tmpDir, "test.fit")
	if err := os.WriteFile(fitPath, []byte("FIT-DATA"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &ActivityUploadCmd{File: fitPath}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	// JSON mode should not print "Uploaded" message.
	if strings.Contains(buf.String(), "Uploaded") {
		t.Fatalf("expected JSON output (no 'Uploaded' message), got: %q", buf.String())
	}
}

func TestActivityUpload_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/upload-service/upload", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	tmpDir := t.TempDir()
	fitPath := filepath.Join(tmpDir, "test.fit")
	if err := os.WriteFile(fitPath, []byte("FIT-DATA"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &ActivityUploadCmd{File: fitPath}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "upload activity") {
		t.Fatalf("unexpected error: %v", err)
	}
}
