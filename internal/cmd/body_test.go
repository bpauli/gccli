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

func bodyTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/weight-service/weight/dateRange", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"totalAverage":{"weight":75000.0},"dateWeightList":[{"date":"2024-01-15"}]}`))
	})

	mux.HandleFunc("/weight-service/weight/range/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"dailyWeightSummaries":[{"date":"2024-01-15","weight":75.0}]}`))
	})

	mux.HandleFunc("/weight-service/weight/dayview/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"dateWeightList":[{"weight":75.0}]}`))
	})

	mux.HandleFunc("/weight-service/user-weight", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"weight":75.5}`))
	})

	mux.HandleFunc("/weight-service/weight/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/bloodpressure-service/bloodpressure/range/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"measurementSummaries":[{"systolic":120,"diastolic":80}]}`))
	})

	mux.HandleFunc("/bloodpressure-service/bloodpressure", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"systolic":120,"diastolic":80,"pulse":72}`))
			return
		}
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})

	// Handle blood pressure delete with date/version path.
	mux.HandleFunc("/bloodpressure-service/bloodpressure/", func(w http.ResponseWriter, r *http.Request) {
		// Check if it's a range request or a delete request.
		if strings.Contains(r.URL.Path, "/range/") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"measurementSummaries":[]}`))
			return
		}
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/upload-service/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"detailedImportResult":{"successes":[{"internalId":12345}]}}`))
	})

	return httptest.NewServer(mux)
}

// --- Execute-level tests ---

func TestExecute_BodyHelp(t *testing.T) {
	code := Execute([]string{"body", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_BodyCompositionHelp(t *testing.T) {
	code := Execute([]string{"body", "composition", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_BodyWeighInsHelp(t *testing.T) {
	code := Execute([]string{"body", "weigh-ins", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_BodyDailyWeighInsHelp(t *testing.T) {
	code := Execute([]string{"body", "daily-weigh-ins", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_BodyAddWeightHelp(t *testing.T) {
	code := Execute([]string{"body", "add-weight", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_BodyAddCompositionHelp(t *testing.T) {
	code := Execute([]string{"body", "add-composition", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_BodyDeleteWeightHelp(t *testing.T) {
	code := Execute([]string{"body", "delete-weight", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_BodyBloodPressureHelp(t *testing.T) {
	code := Execute([]string{"body", "blood-pressure", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_BodyAddBloodPressureHelp(t *testing.T) {
	code := Execute([]string{"body", "add-blood-pressure", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_BodyDeleteBloodPressureHelp(t *testing.T) {
	code := Execute([]string{"body", "delete-blood-pressure", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

// --- BodyCompositionCmd tests ---

func TestBodyComposition_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &BodyCompositionCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBodyComposition_Success(t *testing.T) {
	server := bodyTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &BodyCompositionCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestBodyComposition_WithDateRange(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/weight-service/weight/dateRange", func(w http.ResponseWriter, r *http.Request) {
		called = true
		if got := r.URL.Query().Get("startDate"); got != "2024-01-01" {
			t.Errorf("startDate = %s, want 2024-01-01", got)
		}
		if got := r.URL.Query().Get("endDate"); got != "2024-01-31" {
			t.Errorf("endDate = %s, want 2024-01-31", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"totalAverage":{}}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &BodyCompositionCmd{StartDate: "2024-01-01", EndDate: "2024-01-31"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !called {
		t.Fatal("expected API to be called")
	}
}

func TestBodyComposition_InvalidDate(t *testing.T) {
	server := bodyTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &BodyCompositionCmd{Date: "not-a-date"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid date") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- BodyWeighInsCmd tests ---

func TestBodyWeighIns_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &BodyWeighInsCmd{StartDate: "2024-01-01", EndDate: "2024-01-31"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBodyWeighIns_Success(t *testing.T) {
	server := bodyTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &BodyWeighInsCmd{StartDate: "2024-01-01", EndDate: "2024-01-31"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// --- BodyDailyWeighInsCmd tests ---

func TestBodyDailyWeighIns_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &BodyDailyWeighInsCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBodyDailyWeighIns_Success(t *testing.T) {
	server := bodyTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &BodyDailyWeighInsCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// --- BodyAddWeightCmd tests ---

func TestBodyAddWeight_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &BodyAddWeightCmd{Value: 75.5, Unit: "kg"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBodyAddWeight_Success(t *testing.T) {
	server := bodyTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &BodyAddWeightCmd{Value: 75.5, Unit: "kg"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "Added weight") {
		t.Fatalf("expected 'Added weight' message, got: %q", buf.String())
	}
	if !strings.Contains(buf.String(), "75.5") {
		t.Fatalf("expected weight value in message, got: %q", buf.String())
	}
}

func TestBodyAddWeight_JSON(t *testing.T) {
	server := bodyTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &BodyAddWeightCmd{Value: 75.5, Unit: "kg"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestBodyAddWeight_VerifyPayload(t *testing.T) {
	var receivedPayload map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("/weight-service/user-weight", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedPayload)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"weight":80.0}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &BodyAddWeightCmd{Value: 80.0, Unit: "lbs"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if receivedPayload["value"] != float64(80) {
		t.Errorf("value = %v, want 80", receivedPayload["value"])
	}
	if receivedPayload["unitKey"] != "lbs" {
		t.Errorf("unitKey = %v, want lbs", receivedPayload["unitKey"])
	}
	if receivedPayload["sourceType"] != "MANUAL" {
		t.Errorf("sourceType = %v, want MANUAL", receivedPayload["sourceType"])
	}
	if receivedPayload["dateTimestamp"] == nil {
		t.Error("expected dateTimestamp to be set")
	}
	if receivedPayload["gmtTimestamp"] == nil {
		t.Error("expected gmtTimestamp to be set")
	}
}

// --- BodyAddCompositionCmd tests ---

func TestBodyAddComposition_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &BodyAddCompositionCmd{Weight: 75.5}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBodyAddComposition_Success(t *testing.T) {
	server := bodyTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &BodyAddCompositionCmd{Weight: 75.5}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "Added body composition") {
		t.Fatalf("expected 'Added body composition' message, got: %q", buf.String())
	}
	if !strings.Contains(buf.String(), "75.5") {
		t.Fatalf("expected weight value in message, got: %q", buf.String())
	}
}

func TestBodyAddComposition_JSON(t *testing.T) {
	server := bodyTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &BodyAddCompositionCmd{Weight: 75.5}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestBodyAddComposition_WithAllFields(t *testing.T) {
	var receivedContentType string
	mux := http.NewServeMux()
	mux.HandleFunc("/upload-service/upload", func(w http.ResponseWriter, r *http.Request) {
		receivedContentType = r.Header.Get("Content-Type")

		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("get form file: %v", err)
		}
		defer func() { _ = file.Close() }()

		if header.Filename != "body_composition.fit" {
			t.Errorf("filename = %s, want body_composition.fit", header.Filename)
		}

		data, _ := io.ReadAll(file)
		if len(data) == 0 {
			t.Error("expected non-empty FIT data")
		}

		// Verify it starts with FIT header
		if len(data) >= 12 && string(data[8:12]) != ".FIT" {
			t.Error("uploaded data does not appear to be a valid FIT file")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"detailedImportResult":{"successes":[]}}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	bodyFat := 18.3
	muscleMass := 35.0
	boneMass := 3.1
	bmi := 24.5

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &BodyAddCompositionCmd{
		Weight:     75.5,
		BodyFat:    &bodyFat,
		MuscleMass: &muscleMass,
		BoneMass:   &boneMass,
		BMI:        &bmi,
	}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.HasPrefix(receivedContentType, "multipart/form-data") {
		t.Errorf("Content-Type = %s, want multipart/form-data", receivedContentType)
	}
}

func TestBodyAddComposition_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/upload-service/upload", func(w http.ResponseWriter, _ *http.Request) {
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
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &BodyAddCompositionCmd{Weight: 75.5}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "upload body composition") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- BodyDeleteWeightCmd tests ---

func TestBodyDeleteWeight_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &BodyDeleteWeightCmd{PK: "12345", Date: "2024-01-15", Force: true}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBodyDeleteWeight_Success(t *testing.T) {
	server := bodyTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &BodyDeleteWeightCmd{PK: "12345", Date: "2024-01-15", Force: true}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "Deleted weigh-in") {
		t.Fatalf("expected 'Deleted weigh-in' message, got: %q", buf.String())
	}
}

func TestBodyDeleteWeight_Cancelled(t *testing.T) {
	server := bodyTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	orig := confirmReader
	confirmReader = strings.NewReader("n\n")
	t.Cleanup(func() { confirmReader = orig })

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &BodyDeleteWeightCmd{PK: "12345", Date: "2024-01-15"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "Cancelled") {
		t.Fatalf("expected 'Cancelled' message, got: %q", buf.String())
	}
}

func TestBodyDeleteWeight_ConfirmYes(t *testing.T) {
	server := bodyTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	orig := confirmReader
	confirmReader = strings.NewReader("y\n")
	t.Cleanup(func() { confirmReader = orig })

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &BodyDeleteWeightCmd{PK: "12345", Date: "2024-01-15"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "Deleted weigh-in") {
		t.Fatalf("expected 'Deleted weigh-in' message, got: %q", buf.String())
	}
}

// --- BodyBloodPressureCmd tests ---

func TestBodyBloodPressure_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &BodyBloodPressureCmd{StartDate: "2024-01-01", EndDate: "2024-01-31"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBodyBloodPressure_Success(t *testing.T) {
	server := bodyTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &BodyBloodPressureCmd{StartDate: "2024-01-01", EndDate: "2024-01-31"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// --- BodyAddBPCmd tests ---

func TestBodyAddBP_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &BodyAddBPCmd{Systolic: 120, Diastolic: 80, Pulse: 72}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBodyAddBP_Success(t *testing.T) {
	server := bodyTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &BodyAddBPCmd{Systolic: 120, Diastolic: 80, Pulse: 72}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "Added blood pressure") {
		t.Fatalf("expected 'Added blood pressure' message, got: %q", buf.String())
	}
	if !strings.Contains(buf.String(), "120/80") {
		t.Fatalf("expected bp values in message, got: %q", buf.String())
	}
}

func TestBodyAddBP_JSON(t *testing.T) {
	server := bodyTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &BodyAddBPCmd{Systolic: 120, Diastolic: 80, Pulse: 72}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestBodyAddBP_VerifyPayload(t *testing.T) {
	var receivedPayload map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("/bloodpressure-service/bloodpressure", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedPayload)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"systolic":130,"diastolic":85,"pulse":68}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &BodyAddBPCmd{Systolic: 130, Diastolic: 85, Pulse: 68, Notes: "evening reading"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if receivedPayload["systolic"] != float64(130) {
		t.Errorf("systolic = %v, want 130", receivedPayload["systolic"])
	}
	if receivedPayload["diastolic"] != float64(85) {
		t.Errorf("diastolic = %v, want 85", receivedPayload["diastolic"])
	}
	if receivedPayload["pulse"] != float64(68) {
		t.Errorf("pulse = %v, want 68", receivedPayload["pulse"])
	}
	if receivedPayload["notes"] != "evening reading" {
		t.Errorf("notes = %v, want 'evening reading'", receivedPayload["notes"])
	}
	if receivedPayload["sourceType"] != "MANUAL" {
		t.Errorf("sourceType = %v, want MANUAL", receivedPayload["sourceType"])
	}
}

// --- BodyDeleteBPCmd tests ---

func TestBodyDeleteBP_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &BodyDeleteBPCmd{Version: "67890", Date: "2024-01-15", Force: true}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBodyDeleteBP_Success(t *testing.T) {
	server := bodyTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &BodyDeleteBPCmd{Version: "67890", Date: "2024-01-15", Force: true}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "Deleted blood pressure entry") {
		t.Fatalf("expected 'Deleted blood pressure entry' message, got: %q", buf.String())
	}
}

func TestBodyDeleteBP_Cancelled(t *testing.T) {
	server := bodyTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	orig := confirmReader
	confirmReader = strings.NewReader("n\n")
	t.Cleanup(func() { confirmReader = orig })

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &BodyDeleteBPCmd{Version: "67890", Date: "2024-01-15"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "Cancelled") {
		t.Fatalf("expected 'Cancelled' message, got: %q", buf.String())
	}
}

func TestBodyDeleteBP_ConfirmYes(t *testing.T) {
	server := bodyTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	orig := confirmReader
	confirmReader = strings.NewReader("y\n")
	t.Cleanup(func() { confirmReader = orig })

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &BodyDeleteBPCmd{Version: "67890", Date: "2024-01-15"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "Deleted blood pressure entry") {
		t.Fatalf("expected 'Deleted blood pressure entry' message, got: %q", buf.String())
	}
}

func TestBodyDeleteBP_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/bloodpressure-service/bloodpressure/", func(w http.ResponseWriter, _ *http.Request) {
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
	cmd := &BodyDeleteBPCmd{Version: "67890", Date: "2024-01-15", Force: true}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "delete blood pressure") {
		t.Fatalf("unexpected error: %v", err)
	}
}
