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

// --- ActivityRenameCmd tests ---

func TestActivityRename_Success(t *testing.T) {
	server := modifyTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &ActivityRenameCmd{ID: 12345678, Name: "New Name"}
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

// --- ActivityRetypeCmd tests ---

func TestActivityRetype_Success(t *testing.T) {
	server := modifyTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &ActivityRetypeCmd{ID: 12345678, TypeID: 2, TypeKey: "cycling", ParentTypeID: 17}
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
	cmd := &ActivityRetypeCmd{ID: 12345678, TypeID: 2, TypeKey: "cycling", ParentTypeID: 17}
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
