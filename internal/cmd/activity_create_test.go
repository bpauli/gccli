package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bpauli/gccli/internal/outfmt"
)

func createTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/activity-service/activity", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read body", http.StatusBadRequest)
			return
		}

		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		// Verify expected fields.
		if _, ok := payload["activityName"]; !ok {
			http.Error(w, "missing activityName", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"activityId":99999,"activityName":"Test Run"}`))
	})

	return httptest.NewServer(mux)
}

// --- Execute-level tests ---

func TestExecute_ActivityCreateHelp(t *testing.T) {
	code := Execute([]string{"activity", "create", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

// --- ActivityCreateCmd tests ---

func TestActivityCreate_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &ActivityCreateCmd{Name: "Test", Type: "running", Duration: 30 * time.Minute}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestActivityCreate_Success(t *testing.T) {
	server := createTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &ActivityCreateCmd{
		Name:     "Test Run",
		Type:     "running",
		Distance: 5000,
		Duration: 30 * time.Minute,
	}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "Created activity") {
		t.Fatalf("expected 'Created activity' message, got: %q", buf.String())
	}
}

func TestActivityCreate_JSON(t *testing.T) {
	server := createTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &ActivityCreateCmd{
		Name:     "Test Run",
		Type:     "running",
		Distance: 5000,
		Duration: 30 * time.Minute,
	}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	// JSON mode should not print "Created activity" message.
	if strings.Contains(buf.String(), "Created activity") {
		t.Fatalf("expected JSON output, got: %q", buf.String())
	}
}

func TestActivityCreate_VerifyPayload(t *testing.T) {
	var receivedPayload map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("/activity-service/activity", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedPayload)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"activityId":99999}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &ActivityCreateCmd{
		Name:     "Test Ride",
		Type:     "cycling",
		Distance: 25000,
		Duration: 1*time.Hour + 15*time.Minute,
	}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if receivedPayload == nil {
		t.Fatal("expected payload to be received")
	}

	if name, ok := receivedPayload["activityName"].(string); !ok || name != "Test Ride" {
		t.Errorf("expected activityName 'Test Ride', got %v", receivedPayload["activityName"])
	}

	typeDTO, ok := receivedPayload["activityTypeDTO"].(map[string]any)
	if !ok {
		t.Fatal("expected activityTypeDTO to be a map")
	}
	if typeKey, ok := typeDTO["typeKey"].(string); !ok || typeKey != "cycling" {
		t.Errorf("expected typeKey 'cycling', got %v", typeDTO["typeKey"])
	}

	summaryDTO, ok := receivedPayload["summaryDTO"].(map[string]any)
	if !ok {
		t.Fatal("expected summaryDTO to be a map")
	}
	if dist, ok := summaryDTO["distance"].(float64); !ok || dist != 25000 {
		t.Errorf("expected distance 25000, got %v", summaryDTO["distance"])
	}
	if dur, ok := summaryDTO["duration"].(float64); !ok || dur != 4500 {
		t.Errorf("expected duration 4500 (1h15m in seconds), got %v", summaryDTO["duration"])
	}
}

func TestActivityCreate_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/activity-service/activity", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &ActivityCreateCmd{
		Name:     "Test Run",
		Type:     "running",
		Distance: 5000,
		Duration: 30 * time.Minute,
	}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "create activity") {
		t.Fatalf("unexpected error: %v", err)
	}
}
