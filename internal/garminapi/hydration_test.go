package garminapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetHydration_Success(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/usersummary-service/usersummary/hydration/daily/", func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/2024-01-15") {
			t.Errorf("path = %s, want suffix /2024-01-15", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"totalIntakeML":2500,"goalML":3000,"calendarDate":"2024-01-15"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testTokens(), WithBaseURL(server.URL))
	data, err := client.GetHydration(context.Background(), "2024-01-15")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected API to be called")
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty response")
	}
}

func TestGetHydration_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/usersummary-service/usersummary/hydration/daily/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testTokens(), WithBaseURL(server.URL))
	_, err := client.GetHydration(context.Background(), "2024-01-15")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestAddHydration_Success(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/usersummary-service/usersummary/hydration/log", func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if payload["valueInML"] != 500.0 {
			t.Errorf("valueInML = %v, want 500", payload["valueInML"])
		}
		if payload["calendarDate"] != "2024-01-15" {
			t.Errorf("calendarDate = %v, want 2024-01-15", payload["calendarDate"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"totalIntakeML":3000}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testTokens(), WithBaseURL(server.URL))
	data, err := client.AddHydration(context.Background(), "2024-01-15", 500.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected API to be called")
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty response")
	}
}

func TestAddHydration_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/usersummary-service/usersummary/hydration/log", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testTokens(), WithBaseURL(server.URL))
	_, err := client.AddHydration(context.Background(), "2024-01-15", 500.0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
