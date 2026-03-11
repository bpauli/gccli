package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bpauli/gccli/internal/outfmt"
)

func workoutsTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/workout-service/workouts", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"workoutId":1,"workoutName":"Morning Run","sportType":{"sportTypeKey":"running"},"owner":{"displayName":"Test User"}},{"workoutId":2,"workoutName":"Interval Cycling","sportType":{"sportTypeKey":"cycling"},"owner":{"displayName":"Test User"}}]`))
	})

	mux.HandleFunc("/workout-service/workout/42", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"workoutId":42,"workoutName":"Tempo Run","sportType":{"sportTypeKey":"running"}}`))
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/workout-service/workout/FIT/42", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write([]byte("FIT-WORKOUT-DATA"))
	})

	mux.HandleFunc("/workout-service/workout", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		// Echo back with an added workoutId.
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		payload["workoutId"] = float64(99)
		resp, _ := json.Marshal(payload)
		_, _ = w.Write(resp)
	})

	mux.HandleFunc("/workout-service/schedule/77", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, _ := io.ReadAll(r.Body)
		var payload map[string]string
		if err := json.Unmarshal(body, &payload); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		resp := fmt.Sprintf(`{"scheduledWorkoutId":123,"calendarDate":%q}`, payload["date"])
		_, _ = w.Write([]byte(resp))
	})

	mux.HandleFunc("/calendar-service/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"calendarItems":[{"itemType":"workout","id":123,"workoutId":77,"title":"Morning Run","sportTypeKey":"running","date":"2026-02-12"},{"itemType":"activity","activityId":999,"date":"2026-02-12"},{"itemType":"workout","id":456,"workoutId":88,"title":"Evening Ride","sportTypeKey":"cycling","date":"2026-02-13"}]}`))
	})

	mux.HandleFunc("/workout-service/schedule/123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	return httptest.NewServer(mux)
}

// --- Execute-level tests ---

func TestExecute_WorkoutsHelp(t *testing.T) {
	code := Execute([]string{"workouts", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_WorkoutsListHelp(t *testing.T) {
	code := Execute([]string{"workouts", "list", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_WorkoutsDetailHelp(t *testing.T) {
	code := Execute([]string{"workouts", "detail", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_WorkoutsDownloadHelp(t *testing.T) {
	code := Execute([]string{"workouts", "download", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_WorkoutsUploadHelp(t *testing.T) {
	code := Execute([]string{"workouts", "upload", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_WorkoutsScheduleHelp(t *testing.T) {
	code := Execute([]string{"workouts", "schedule", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_WorkoutsScheduleAddHelp(t *testing.T) {
	code := Execute([]string{"workouts", "schedule", "add", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_WorkoutsScheduleListHelp(t *testing.T) {
	code := Execute([]string{"workouts", "schedule", "list", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_WorkoutsScheduleRemoveHelp(t *testing.T) {
	code := Execute([]string{"workouts", "schedule", "remove", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_WorkoutsDeleteHelp(t *testing.T) {
	code := Execute([]string{"workouts", "delete", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

// --- WorkoutsListCmd tests ---

func TestWorkoutsList_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &WorkoutsListCmd{Limit: 20}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkoutsList_Table(t *testing.T) {
	server := workoutsTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &WorkoutsListCmd{Limit: 20}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestWorkoutsList_JSON(t *testing.T) {
	server := workoutsTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &WorkoutsListCmd{Limit: 20}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestWorkoutsList_Plain(t *testing.T) {
	server := workoutsTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Plain, "test@example.com")
	cmd := &WorkoutsListCmd{Limit: 20}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// --- WorkoutDetailCmd tests ---

func TestWorkoutDetail_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &WorkoutDetailCmd{ID: "42"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkoutDetail_Success(t *testing.T) {
	server := workoutsTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &WorkoutDetailCmd{ID: "42"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// --- WorkoutDownloadCmd tests ---

func TestWorkoutDownload_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &WorkoutDownloadCmd{ID: "42"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkoutDownload_Success(t *testing.T) {
	server := workoutsTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	outDir := t.TempDir()
	outPath := filepath.Join(outDir, "workout_42.fit")

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &WorkoutDownloadCmd{ID: "42", Output: outPath}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if string(data) != "FIT-WORKOUT-DATA" {
		t.Errorf("expected FIT data, got %q", data)
	}
	if !strings.Contains(buf.String(), "Downloaded") {
		t.Fatalf("expected 'Downloaded' message, got: %q", buf.String())
	}
}

func TestWorkoutDownload_DefaultFilename(t *testing.T) {
	server := workoutsTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &WorkoutDownloadCmd{ID: "42"}
	err = cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	expectedFile := filepath.Join(tmpDir, "workout_42.fit")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Fatalf("expected default output file %q to exist", expectedFile)
	}
}

func TestWorkoutDownload_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/workout-service/workout/FIT/42", func(w http.ResponseWriter, _ *http.Request) {
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
	cmd := &WorkoutDownloadCmd{ID: "42"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "download workout") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- WorkoutUploadCmd tests ---

func TestWorkoutUpload_NoAccount(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "workout.json")
	if err := os.WriteFile(tmpFile, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &WorkoutUploadCmd{File: tmpFile}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkoutUpload_Success(t *testing.T) {
	server := workoutsTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	tmpFile := filepath.Join(t.TempDir(), "workout.json")
	workoutJSON := `{"workoutName":"Test Workout","sportType":{"sportTypeKey":"running"}}`
	if err := os.WriteFile(tmpFile, []byte(workoutJSON), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &WorkoutUploadCmd{File: tmpFile}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "Uploaded workout") {
		t.Fatalf("expected 'Uploaded workout' message, got: %q", buf.String())
	}
}

func TestWorkoutUpload_JSON(t *testing.T) {
	server := workoutsTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	tmpFile := filepath.Join(t.TempDir(), "workout.json")
	workoutJSON := `{"workoutName":"Test Workout","sportType":{"sportTypeKey":"running"}}`
	if err := os.WriteFile(tmpFile, []byte(workoutJSON), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &WorkoutUploadCmd{File: tmpFile}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestWorkoutUpload_InvalidJSON(t *testing.T) {
	server := workoutsTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	tmpFile := filepath.Join(t.TempDir(), "workout.json")
	if err := os.WriteFile(tmpFile, []byte("not json at all"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &WorkoutUploadCmd{File: tmpFile}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid JSON") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- WorkoutScheduleAddCmd tests ---

func TestWorkoutScheduleAdd_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &WorkoutScheduleAddCmd{ID: "77", Date: "2026-02-12"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkoutScheduleAdd_Success(t *testing.T) {
	server := workoutsTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &WorkoutScheduleAddCmd{ID: "77", Date: "2026-02-12"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "Scheduled workout 77 on 2026-02-12") {
		t.Fatalf("expected success message, got: %q", buf.String())
	}
}

func TestWorkoutScheduleAdd_JSON(t *testing.T) {
	server := workoutsTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &WorkoutScheduleAddCmd{ID: "77", Date: "2026-02-12"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestWorkoutScheduleAdd_InvalidDate(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &WorkoutScheduleAddCmd{ID: "77", Date: "bad-date"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid date") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- WorkoutScheduleListCmd tests ---

func TestWorkoutScheduleList_Success(t *testing.T) {
	server := workoutsTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &WorkoutScheduleListCmd{Date: "2026-02-12"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestWorkoutScheduleList_JSON(t *testing.T) {
	server := workoutsTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &WorkoutScheduleListCmd{Date: "2026-02-12"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestWorkoutScheduleList_InvalidDate(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &WorkoutScheduleListCmd{Date: "bad-date"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid date") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkoutScheduleList_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &WorkoutScheduleListCmd{Date: "2026-02-12"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- WorkoutScheduleListCmd range tests ---

func TestWorkoutScheduleList_Range(t *testing.T) {
	server := workoutsTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &WorkoutScheduleListCmd{StartDate: "2026-02-10", EndDate: "2026-02-20"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestWorkoutScheduleList_RangeJSON(t *testing.T) {
	server := workoutsTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &WorkoutScheduleListCmd{StartDate: "2026-02-10", EndDate: "2026-02-20"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestWorkoutScheduleList_RangeMissingEnd(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &WorkoutScheduleListCmd{StartDate: "2026-02-10"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "both --start and --end are required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkoutScheduleList_RangeEndBeforeStart(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &WorkoutScheduleListCmd{StartDate: "2026-03-01", EndDate: "2026-02-01"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "is before --start") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- filterCalendarWorkoutsInRange tests ---

func TestFilterCalendarWorkoutsInRange(t *testing.T) {
	data := []byte(`{"calendarItems":[
		{"itemType":"workout","date":"2026-02-10","title":"W1"},
		{"itemType":"workout","date":"2026-02-15","title":"W2"},
		{"itemType":"workout","date":"2026-02-20","title":"W3"},
		{"itemType":"workout","date":"2026-02-25","title":"W4"},
		{"itemType":"activity","date":"2026-02-15","title":"A1"}
	]}`)

	items, err := filterCalendarWorkoutsInRange(data, "2026-02-10", "2026-02-20")
	if err != nil {
		t.Fatalf("filterCalendarWorkoutsInRange: %v", err)
	}
	if len(items) != 3 {
		t.Errorf("got %d items, want 3", len(items))
	}
}

func TestFilterCalendarWorkoutsInRange_Empty(t *testing.T) {
	data := []byte(`{"calendarItems":[]}`)
	items, err := filterCalendarWorkoutsInRange(data, "2026-02-10", "2026-02-20")
	if err != nil {
		t.Fatalf("filterCalendarWorkoutsInRange: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("got %d items, want 0", len(items))
	}
}

// --- WorkoutScheduleRemoveCmd tests ---

func TestWorkoutScheduleRemove_Success(t *testing.T) {
	server := workoutsTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &WorkoutScheduleRemoveCmd{ID: "123", Force: true}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "Removed scheduled workout 123") {
		t.Fatalf("expected success message, got: %q", buf.String())
	}
}

func TestWorkoutScheduleRemove_Cancelled(t *testing.T) {
	server := workoutsTestServer(t)
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
	cmd := &WorkoutScheduleRemoveCmd{ID: "123"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "Cancelled") {
		t.Fatalf("expected 'Cancelled' message, got: %q", buf.String())
	}
}

func TestWorkoutScheduleRemove_ConfirmYes(t *testing.T) {
	server := workoutsTestServer(t)
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
	cmd := &WorkoutScheduleRemoveCmd{ID: "123"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "Removed scheduled workout 123") {
		t.Fatalf("expected success message, got: %q", buf.String())
	}
}

func TestWorkoutScheduleRemove_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &WorkoutScheduleRemoveCmd{ID: "123", Force: true}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- WorkoutDeleteCmd tests ---

func TestWorkoutDelete_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &WorkoutDeleteCmd{ID: "42", Force: true}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkoutDelete_Success(t *testing.T) {
	server := workoutsTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &WorkoutDeleteCmd{ID: "42", Force: true}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "Deleted workout") {
		t.Fatalf("expected 'Deleted workout' message, got: %q", buf.String())
	}
}

func TestWorkoutDelete_Cancelled(t *testing.T) {
	server := workoutsTestServer(t)
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
	cmd := &WorkoutDeleteCmd{ID: "42"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "Cancelled") {
		t.Fatalf("expected 'Cancelled' message, got: %q", buf.String())
	}
}

func TestWorkoutDelete_ConfirmYes(t *testing.T) {
	server := workoutsTestServer(t)
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
	cmd := &WorkoutDeleteCmd{ID: "42"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "Deleted workout") {
		t.Fatalf("expected 'Deleted workout' message, got: %q", buf.String())
	}
}

func TestWorkoutDelete_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/workout-service/workout/42", func(w http.ResponseWriter, _ *http.Request) {
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
	cmd := &WorkoutDeleteCmd{ID: "42", Force: true}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "delete workout") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- parseWorkouts tests ---

func TestParseWorkouts_Array(t *testing.T) {
	data := json.RawMessage(`[{"workoutId":1,"workoutName":"Run"},{"workoutId":2,"workoutName":"Bike"}]`)
	workouts, err := parseWorkouts(data)
	if err != nil {
		t.Fatalf("parseWorkouts: %v", err)
	}
	if len(workouts) != 2 {
		t.Errorf("got %d workouts, want 2", len(workouts))
	}
}

func TestParseWorkouts_Wrapper(t *testing.T) {
	data := json.RawMessage(`{"workouts":[{"workoutId":1,"workoutName":"Run"}],"totalCount":1}`)
	workouts, err := parseWorkouts(data)
	if err != nil {
		t.Fatalf("parseWorkouts: %v", err)
	}
	if len(workouts) != 1 {
		t.Errorf("got %d workouts, want 1", len(workouts))
	}
}

func TestParseWorkouts_InvalidJSON(t *testing.T) {
	data := json.RawMessage(`not json`)
	_, err := parseWorkouts(data)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestParseWorkouts_EmptyArray(t *testing.T) {
	data := json.RawMessage(`[]`)
	workouts, err := parseWorkouts(data)
	if err != nil {
		t.Fatalf("parseWorkouts: %v", err)
	}
	if len(workouts) != 0 {
		t.Errorf("got %d workouts, want 0", len(workouts))
	}
}

// --- formatWorkoutRows tests ---

func TestFormatWorkoutRows(t *testing.T) {
	workouts := []map[string]any{
		{
			"workoutId":   float64(1),
			"workoutName": "Morning Run",
			"sportType":   map[string]any{"sportTypeKey": "running"},
			"owner":       map[string]any{"displayName": "Test User"},
		},
		{
			"workoutId":   float64(2),
			"workoutName": "Bike Session",
		},
	}

	rows := formatWorkoutRows(workouts)
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}

	// Row 1: full data.
	if rows[0][0] != "1" {
		t.Errorf("row[0][0] = %q, want 1", rows[0][0])
	}
	if rows[0][1] != "Morning Run" {
		t.Errorf("row[0][1] = %q, want Morning Run", rows[0][1])
	}
	if rows[0][2] != "running" {
		t.Errorf("row[0][2] = %q, want running", rows[0][2])
	}
	if rows[0][3] != "Test User" {
		t.Errorf("row[0][3] = %q, want Test User", rows[0][3])
	}

	// Row 2: missing sportType and owner.
	if rows[1][2] != "" {
		t.Errorf("row[1][2] = %q, want empty", rows[1][2])
	}
	if rows[1][3] != "" {
		t.Errorf("row[1][3] = %q, want empty", rows[1][3])
	}
}

// --- filterCalendarWorkouts tests ---

func TestFilterCalendarWorkouts(t *testing.T) {
	data := json.RawMessage(`{"calendarItems":[
		{"itemType":"workout","id":123,"workoutId":77,"title":"Morning Run","sportTypeKey":"running","date":"2026-02-12"},
		{"itemType":"activity","activityId":999,"date":"2026-02-12"},
		{"itemType":"workout","id":456,"workoutId":88,"title":"Evening Ride","sportTypeKey":"cycling","date":"2026-02-13"}
	]}`)

	items, err := filterCalendarWorkouts(data, "2026-02-12")
	if err != nil {
		t.Fatalf("filterCalendarWorkouts: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	if items[0]["title"] != "Morning Run" {
		t.Errorf("title = %v, want Morning Run", items[0]["title"])
	}
}

func TestFilterCalendarWorkouts_NoItems(t *testing.T) {
	data := json.RawMessage(`{"calendarItems":[]}`)
	items, err := filterCalendarWorkouts(data, "2026-02-12")
	if err != nil {
		t.Fatalf("filterCalendarWorkouts: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("got %d items, want 0", len(items))
	}
}

func TestFilterCalendarWorkouts_NoCalendarItems(t *testing.T) {
	data := json.RawMessage(`{"startDate":"2026-02-12"}`)
	items, err := filterCalendarWorkouts(data, "2026-02-12")
	if err != nil {
		t.Fatalf("filterCalendarWorkouts: %v", err)
	}
	if items != nil {
		t.Errorf("got %v, want nil", items)
	}
}

func TestFilterCalendarWorkouts_InvalidJSON(t *testing.T) {
	data := json.RawMessage(`not json`)
	_, err := filterCalendarWorkouts(data, "2026-02-12")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- formatCalendarWorkoutRows tests ---

func TestFormatCalendarWorkoutRows(t *testing.T) {
	items := []map[string]any{
		{
			"id":           float64(123),
			"workoutId":    float64(77),
			"title":        "Morning Run",
			"sportTypeKey": "running",
			"date":         "2026-02-12",
		},
	}

	rows := formatCalendarWorkoutRows(items)
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	if rows[0][0] != "123" {
		t.Errorf("scheduleId = %q, want 123", rows[0][0])
	}
	if rows[0][1] != "77" {
		t.Errorf("workoutId = %q, want 77", rows[0][1])
	}
	if rows[0][2] != "Morning Run" {
		t.Errorf("title = %q, want Morning Run", rows[0][2])
	}
	if rows[0][3] != "running" {
		t.Errorf("sport = %q, want running", rows[0][3])
	}
	if rows[0][4] != "2026-02-12" {
		t.Errorf("date = %q, want 2026-02-12", rows[0][4])
	}
}

func TestFormatCalendarWorkoutRows_Empty(t *testing.T) {
	rows := formatCalendarWorkoutRows(nil)
	if len(rows) != 0 {
		t.Errorf("got %d rows, want 0", len(rows))
	}
}

// --- workoutSportType tests ---

func TestWorkoutSportType(t *testing.T) {
	tests := []struct {
		name string
		data map[string]any
		want string
	}{
		{"with sport type", map[string]any{"sportType": map[string]any{"sportTypeKey": "running"}}, "running"},
		{"nil sport type", map[string]any{"sportType": nil}, ""},
		{"missing sport type", map[string]any{}, ""},
		{"wrong type", map[string]any{"sportType": "not a map"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := workoutSportType(tt.data)
			if got != tt.want {
				t.Errorf("workoutSportType() = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- workoutOwner tests ---

func TestWorkoutOwner(t *testing.T) {
	tests := []struct {
		name string
		data map[string]any
		want string
	}{
		{"with owner", map[string]any{"owner": map[string]any{"displayName": "John"}}, "John"},
		{"nil owner", map[string]any{"owner": nil}, ""},
		{"missing owner", map[string]any{}, ""},
		{"wrong type", map[string]any{"owner": "not a map"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := workoutOwner(tt.data)
			if got != tt.want {
				t.Errorf("workoutOwner() = %q, want %q", got, tt.want)
			}
		})
	}
}
