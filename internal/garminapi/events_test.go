package garminapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func TestGetEvents_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/calendar-service/events" {
			t.Errorf("path = %s, want /calendar-service/events", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("startDate") != "2026-03-11" {
			t.Errorf("startDate = %s, want 2026-03-11", q.Get("startDate"))
		}
		if q.Get("pageIndex") != "1" {
			t.Errorf("pageIndex = %s, want 1", q.Get("pageIndex"))
		}
		if q.Get("limit") != "20" {
			t.Errorf("limit = %s, want 20", q.Get("limit"))
		}
		if q.Get("sortOrder") != "eventDate_asc" {
			t.Errorf("sortOrder = %s, want eventDate_asc", q.Get("sortOrder"))
		}
		_, _ = w.Write([]byte(`[{"id":1,"eventName":"Race"}]`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetEvents(context.Background(), "2026-03-11", 1, 20, "eventDate_asc")
	if err != nil {
		t.Fatalf("GetEvents: %v", err)
	}

	var events []map[string]any
	if err := json.Unmarshal(data, &events); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("got %d events, want 1", len(events))
	}
}

func TestAddEvent_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/calendar-service/event" {
			t.Errorf("path = %s, want /calendar-service/event", r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["eventName"] != "Test Race" {
			t.Errorf("eventName = %v, want Test Race", body["eventName"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":12345,"eventName":"Test Race","date":"2026-04-01"}`))
	})

	_, client := testServer(t, handler)
	payload := json.RawMessage(`{"eventName":"Test Race","date":"2026-04-01","eventType":"running"}`)
	data, err := client.AddEvent(context.Background(), payload)
	if err != nil {
		t.Fatalf("AddEvent: %v", err)
	}

	var resp map[string]any
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["eventName"] != "Test Race" {
		t.Errorf("eventName = %v, want Test Race", resp["eventName"])
	}
}

func TestAddEvent_ServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})

	_, client := testServer(t, handler)
	payload := json.RawMessage(`{"eventName":"Test"}`)
	_, err := client.AddEvent(context.Background(), payload)
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *GarminAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected GarminAPIError, got %T: %v", err, err)
	}
}

func TestDeleteEvent_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if r.URL.Path != "/calendar-service/event/99999" {
			t.Errorf("path = %s, want /calendar-service/event/99999", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	_, client := testServer(t, handler)
	err := client.DeleteEvent(context.Background(), "99999")
	if err != nil {
		t.Fatalf("DeleteEvent: %v", err)
	}
}

func TestDeleteEvent_ServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})

	_, client := testServer(t, handler)
	err := client.DeleteEvent(context.Background(), "99999")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetEvents_ServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})

	_, client := testServer(t, handler)
	_, err := client.GetEvents(context.Background(), "2026-03-11", 1, 20, "eventDate_asc")
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *GarminAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected GarminAPIError, got %T: %v", err, err)
	}
}
