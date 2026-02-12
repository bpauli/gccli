package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bpauli/gccli/internal/outfmt"
)

func modifyTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/activity-service/activity/12345678", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			body, _ := io.ReadAll(r.Body)
			var payload map[string]any
			if err := json.Unmarshal(body, &payload); err != nil {
				http.Error(w, "invalid JSON", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(body) // Echo back payload.
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	return httptest.NewServer(mux)
}

// --- Execute-level tests ---

func TestExecute_ActivityRenameHelp(t *testing.T) {
	code := Execute([]string{"activity", "rename", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_ActivityRetypeHelp(t *testing.T) {
	code := Execute([]string{"activity", "retype", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_ActivityDeleteHelp(t *testing.T) {
	code := Execute([]string{"activity", "delete", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

// --- ActivityRenameCmd tests ---

func TestActivityRename_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &ActivityRenameCmd{ID: "12345678", Name: "New Name"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestActivityRename_Success(t *testing.T) {
	server := modifyTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &ActivityRenameCmd{ID: "12345678", Name: "New Name"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "Renamed activity") {
		t.Fatalf("expected 'Renamed activity' message, got: %q", buf.String())
	}
	if !strings.Contains(buf.String(), "New Name") {
		t.Fatalf("expected new name in message, got: %q", buf.String())
	}
}

func TestActivityRename_JSON(t *testing.T) {
	server := modifyTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &ActivityRenameCmd{ID: "12345678", Name: "New Name"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestActivityRename_InvalidID(t *testing.T) {
	server := modifyTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &ActivityRenameCmd{ID: "not-a-number", Name: "New Name"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "rename activity") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- ActivityRetypeCmd tests ---

func TestActivityRetype_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &ActivityRetypeCmd{ID: "12345678", TypeID: 1, TypeKey: "running"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestActivityRetype_Success(t *testing.T) {
	server := modifyTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &ActivityRetypeCmd{ID: "12345678", TypeID: 2, TypeKey: "cycling", ParentTypeID: 17}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "Changed activity") {
		t.Fatalf("expected 'Changed activity' message, got: %q", buf.String())
	}
	if !strings.Contains(buf.String(), "cycling") {
		t.Fatalf("expected type key in message, got: %q", buf.String())
	}
}

func TestActivityRetype_JSON(t *testing.T) {
	server := modifyTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &ActivityRetypeCmd{ID: "12345678", TypeID: 1, TypeKey: "running"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestActivityRetype_VerifyPayload(t *testing.T) {
	var receivedPayload map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("/activity-service/activity/12345678", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedPayload)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &ActivityRetypeCmd{ID: "12345678", TypeID: 2, TypeKey: "cycling", ParentTypeID: 17}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	typeDTO, ok := receivedPayload["activityTypeDTO"].(map[string]any)
	if !ok {
		t.Fatal("expected activityTypeDTO")
	}
	if typeDTO["typeKey"] != "cycling" {
		t.Errorf("expected typeKey 'cycling', got %v", typeDTO["typeKey"])
	}
	if typeDTO["typeId"] != float64(2) {
		t.Errorf("expected typeId 2, got %v", typeDTO["typeId"])
	}
	if typeDTO["parentTypeId"] != float64(17) {
		t.Errorf("expected parentTypeId 17, got %v", typeDTO["parentTypeId"])
	}
}

// --- ActivityDeleteCmd tests ---

func TestActivityDelete_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &ActivityDeleteCmd{ID: "12345678", Force: true}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestActivityDelete_Success(t *testing.T) {
	server := modifyTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &ActivityDeleteCmd{ID: "12345678", Force: true}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "Deleted activity") {
		t.Fatalf("expected 'Deleted activity' message, got: %q", buf.String())
	}
}

func TestActivityDelete_Cancelled(t *testing.T) {
	server := modifyTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	// Simulate user saying "n".
	orig := confirmReader
	confirmReader = strings.NewReader("n\n")
	t.Cleanup(func() { confirmReader = orig })

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &ActivityDeleteCmd{ID: "12345678"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "Cancelled") {
		t.Fatalf("expected 'Cancelled' message, got: %q", buf.String())
	}
}

func TestActivityDelete_ConfirmYes(t *testing.T) {
	server := modifyTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	// Simulate user saying "y".
	orig := confirmReader
	confirmReader = strings.NewReader("y\n")
	t.Cleanup(func() { confirmReader = orig })

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &ActivityDeleteCmd{ID: "12345678"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "Deleted activity") {
		t.Fatalf("expected 'Deleted activity' message, got: %q", buf.String())
	}
}

func TestActivityDelete_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/activity-service/activity/12345678", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &ActivityDeleteCmd{ID: "12345678", Force: true}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "delete activity") {
		t.Fatalf("unexpected error: %v", err)
	}
}
