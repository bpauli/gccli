package garminapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetTrainingPlans_Success(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/trainingplan-service/trainingplan/plans", func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Query().Get("locale") != "en" {
			t.Errorf("locale = %s, want en", r.URL.Query().Get("locale"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"planId":"plan-1","planName":"5K Beginner"}]`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testTokens(), WithBaseURL(server.URL))
	data, err := client.GetTrainingPlans(context.Background(), "en")
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

func TestGetTrainingPlans_DefaultLocale(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/trainingplan-service/trainingplan/plans", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("locale") != "en" {
			t.Errorf("locale = %s, want en (default)", r.URL.Query().Get("locale"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testTokens(), WithBaseURL(server.URL))
	_, err := client.GetTrainingPlans(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetTrainingPlans_CustomLocale(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/trainingplan-service/trainingplan/plans", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("locale") != "de" {
			t.Errorf("locale = %s, want de", r.URL.Query().Get("locale"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testTokens(), WithBaseURL(server.URL))
	_, err := client.GetTrainingPlans(context.Background(), "de")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetTrainingPlans_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/trainingplan-service/trainingplan/plans", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testTokens(), WithBaseURL(server.URL))
	_, err := client.GetTrainingPlans(context.Background(), "en")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetTrainingPlan_Success(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/trainingplan-service/trainingplan/plan/", func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/plan-123") {
			t.Errorf("path = %s, want suffix /plan-123", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"planId":"plan-123","planName":"5K Beginner","description":"A beginner 5K plan"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testTokens(), WithBaseURL(server.URL))
	data, err := client.GetTrainingPlan(context.Background(), "plan-123")
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

func TestGetTrainingPlan_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/trainingplan-service/trainingplan/plan/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testTokens(), WithBaseURL(server.URL))
	_, err := client.GetTrainingPlan(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
