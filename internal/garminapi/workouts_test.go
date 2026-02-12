package garminapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bpauli/gccli/internal/garminauth"
)

// --- GetWorkouts tests ---

func TestGetWorkouts_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/workout-service/workouts" {
			t.Errorf("path = %s, want /workout-service/workouts", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if got := r.URL.Query().Get("start"); got != "0" {
			t.Errorf("start = %s, want 0", got)
		}
		if got := r.URL.Query().Get("limit"); got != "10" {
			t.Errorf("limit = %s, want 10", got)
		}
		_, _ = w.Write([]byte(`[{"workoutId":1,"workoutName":"Morning Run"}]`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetWorkouts(context.Background(), 0, 10)
	if err != nil {
		t.Fatalf("GetWorkouts: %v", err)
	}

	var workouts []map[string]any
	if err := json.Unmarshal(data, &workouts); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(workouts) != 1 {
		t.Errorf("got %d workouts, want 1", len(workouts))
	}
}

func TestGetWorkouts_ServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})

	_, client := testServer(t, handler)
	_, err := client.GetWorkouts(context.Background(), 0, 10)
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *GarminAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected GarminAPIError, got %T: %v", err, err)
	}
}

// --- GetWorkout tests ---

func TestGetWorkout_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/workout-service/workout/42" {
			t.Errorf("path = %s, want /workout-service/workout/42", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"workoutId":42,"workoutName":"Tempo Run"}`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetWorkout(context.Background(), "42")
	if err != nil {
		t.Fatalf("GetWorkout: %v", err)
	}

	var workout map[string]any
	if err := json.Unmarshal(data, &workout); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if workout["workoutName"] != "Tempo Run" {
		t.Errorf("workoutName = %v, want Tempo Run", workout["workoutName"])
	}
}

func TestGetWorkout_NotFound(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	})

	_, client := testServer(t, handler)
	_, err := client.GetWorkout(context.Background(), "999")
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *GarminAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected GarminAPIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("status = %d, want 404", apiErr.StatusCode)
	}
}

// --- DownloadWorkout tests ---

func TestDownloadWorkout_Success(t *testing.T) {
	fitData := []byte("FIT-BINARY-WORKOUT-DATA")
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/workout-service/workout/FIT/42" {
			t.Errorf("path = %s, want /workout-service/workout/FIT/42", r.URL.Path)
		}
		_, _ = w.Write(fitData)
	})

	_, client := testServer(t, handler)
	data, err := client.DownloadWorkout(context.Background(), "42")
	if err != nil {
		t.Fatalf("DownloadWorkout: %v", err)
	}
	if string(data) != string(fitData) {
		t.Errorf("got %q, want %q", data, fitData)
	}
}

func TestDownloadWorkout_ServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})

	_, client := testServer(t, handler)
	_, err := client.DownloadWorkout(context.Background(), "42")
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- UploadWorkout tests ---

func TestUploadWorkout_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/workout-service/workout" {
			t.Errorf("path = %s, want /workout-service/workout", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}

		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		if payload["workoutName"] != "Test Workout" {
			t.Errorf("workoutName = %v, want Test Workout", payload["workoutName"])
		}

		_, _ = w.Write([]byte(`{"workoutId":99,"workoutName":"Test Workout"}`))
	})

	_, client := testServer(t, handler)
	workoutJSON := json.RawMessage(`{"workoutName":"Test Workout","sportType":{"sportTypeKey":"running"}}`)
	data, err := client.UploadWorkout(context.Background(), workoutJSON)
	if err != nil {
		t.Fatalf("UploadWorkout: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if result["workoutId"] != float64(99) {
		t.Errorf("workoutId = %v, want 99", result["workoutId"])
	}
}

func TestUploadWorkout_ServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("invalid workout"))
	})

	_, client := testServer(t, handler)
	_, err := client.UploadWorkout(context.Background(), json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *GarminAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected GarminAPIError, got %T: %v", err, err)
	}
}

// --- ScheduleWorkout tests ---

func TestScheduleWorkout_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/workout-service/schedule/77" {
			t.Errorf("path = %s, want /workout-service/schedule/77", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}

		var payload map[string]string
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		if payload["date"] != "2026-02-12" {
			t.Errorf("date = %s, want 2026-02-12", payload["date"])
		}

		_, _ = w.Write([]byte(`{"scheduledWorkoutId":123,"calendarDate":"2026-02-12"}`))
	})

	_, client := testServer(t, handler)
	data, err := client.ScheduleWorkout(context.Background(), "77", "2026-02-12")
	if err != nil {
		t.Fatalf("ScheduleWorkout: %v", err)
	}

	var scheduled map[string]any
	if err := json.Unmarshal(data, &scheduled); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if scheduled["scheduledWorkoutId"] != float64(123) {
		t.Errorf("scheduledWorkoutId = %v, want 123", scheduled["scheduledWorkoutId"])
	}
}

// --- DeleteWorkout tests ---

func TestDeleteWorkout_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/workout-service/workout/42" {
			t.Errorf("path = %s, want /workout-service/workout/42", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	_, client := testServer(t, handler)
	err := client.DeleteWorkout(context.Background(), "42")
	if err != nil {
		t.Fatalf("DeleteWorkout: %v", err)
	}
}

func TestDeleteWorkout_NotFound(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	})

	_, client := testServer(t, handler)
	err := client.DeleteWorkout(context.Background(), "999")
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *GarminAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected GarminAPIError, got %T: %v", err, err)
	}
}

// --- Token refresh on 401 ---

func TestDownloadWorkout_401WithRefresh(t *testing.T) {
	calls := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte("FIT-DATA"))
	})

	ts := httptest.NewServer(handler)
	defer ts.Close()

	orig := refreshTokensFn
	refreshTokensFn = func(ctx context.Context, tokens *garminauth.Tokens, opts garminauth.LoginOptions) (*garminauth.Tokens, error) {
		newTok := *tokens
		newTok.OAuth2AccessToken = "refreshed-token"
		return &newTok, nil
	}
	defer func() { refreshTokensFn = orig }()

	tokens := testTokens()
	client := NewClient(tokens, WithBaseURL(ts.URL))
	data, err := client.DownloadWorkout(context.Background(), "42")
	if err != nil {
		t.Fatalf("expected success after refresh, got: %v", err)
	}
	if string(data) != "FIT-DATA" {
		t.Errorf("got %q, want FIT-DATA", data)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls, got %d", calls)
	}
}
