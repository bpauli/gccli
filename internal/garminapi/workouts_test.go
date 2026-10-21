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

func TestWorkoutsEndpoints(t *testing.T) {
	id := map[string]string{"workoutID": "42"}
	testEndpoints(t, workoutsEndpoints, []endpointCase{
		{"workouts.list", map[string]string{"start": "0", "limit": "10"}, "GET /workout-service/workouts?start=0&limit=10"},
		{"workouts.get", id, "GET /workout-service/workout/42"},
		{"workouts.upload", nil, "POST /workout-service/workout"},
		{"workouts.update", id, "PUT /workout-service/workout/42"},
		{"workouts.schedule", id, "POST /workout-service/schedule/42"},
		{"workouts.unschedule", map[string]string{"scheduleID": "456"}, "DELETE /workout-service/schedule/456"},
		{"workouts.delete", id, "DELETE /workout-service/workout/42"},
		{"workouts.types", nil, "GET /workout-service/workout/types"},
		{"workouts.sport-types", nil, "GET /workout-service/workout/types/sport"},
		{"workouts.export-schedule", map[string]string{"startDate": "2024-01-15"}, "GET /workout-service/schedule/export/JSON?startDate=2024-01-15"},
		{"workouts.benchmarks", nil, "GET /workout-service/benchmarks"},
	})
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

func TestCopyWorkout(t *testing.T) {
	tests := []struct {
		name, id, copyName string
		wantPath, wantBody string
	}{
		{"named", "42", `Tempo "B" ü`, "/workout-service/workout/copy/42", `Tempo "B" ü`},
		{"no name", "42", "", "/workout-service/workout/copy/42", ""},
		{"id escaped", "4/2", "x", "/workout-service/workout/copy/4%2F2", "x"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var method, path, contentType, body string
			_, client := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				method, path, contentType = r.Method, r.URL.EscapedPath(), r.Header.Get("Content-Type")
				b, _ := io.ReadAll(r.Body)
				body = string(b)
				_, _ = w.Write([]byte(`{"workoutId":43}`))
			}))
			data, err := client.CopyWorkout(context.Background(), tc.id, tc.copyName)
			if err != nil {
				t.Fatalf("CopyWorkout: %v", err)
			}
			if string(data) != `{"workoutId":43}` {
				t.Errorf("data = %s, want the response body", data)
			}
			if method != http.MethodPost || path != tc.wantPath {
				t.Errorf("request = %s %s, want POST %s", method, path, tc.wantPath)
			}
			if contentType != "text/plain; charset=utf-8" {
				t.Errorf("Content-Type = %q, want text/plain; charset=utf-8: Garmin keeps the quotes of a JSON string in the name", contentType)
			}
			if body != tc.wantBody {
				t.Errorf("body = %q, want the raw name %q", body, tc.wantBody)
			}
		})
	}
}

func TestCopyWorkout_ServerError(t *testing.T) {
	_, client := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	_, err := client.CopyWorkout(context.Background(), "42", "x")
	var apiErr *GarminAPIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("err = %v, want a GarminAPIError with status 400", err)
	}
}

// --- GetCalendarWeek tests ---

func TestGetCalendarWeek_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// February is month 2, 0-indexed = 1.
		if r.URL.Path != "/calendar-service/year/2026/month/1/day/12/start/0" {
			t.Errorf("path = %s, want /calendar-service/year/2026/month/1/day/12/start/0", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		_, _ = w.Write([]byte(`{"calendarItems":[{"itemType":"workout","date":"2026-02-12"}]}`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetCalendarWeek(context.Background(), "2026-02-12")
	if err != nil {
		t.Fatalf("GetCalendarWeek: %v", err)
	}

	var calendar map[string]any
	if err := json.Unmarshal(data, &calendar); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if calendar["calendarItems"] == nil {
		t.Error("expected calendarItems in response")
	}
}

func TestGetCalendarWeek_ServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})

	_, client := testServer(t, handler)
	_, err := client.GetCalendarWeek(context.Background(), "2026-02-12")
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *GarminAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected GarminAPIError, got %T: %v", err, err)
	}
}

func TestGetCalendarWeek_InvalidDate(t *testing.T) {
	_, client := testServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	_, err := client.GetCalendarWeek(context.Background(), "not-a-date")
	if err == nil {
		t.Fatal("expected error")
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
	client := NewClient(tokens, WithHTTPClient(ts.Client()), WithBaseURL(ts.URL))
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
