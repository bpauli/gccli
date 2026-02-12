package garminapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetMenstrualCycleData_Success(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/periodichealth-service/menstrualcycle/dayview/", func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/2024-01-01/2024-01-31") {
			t.Errorf("path = %s, want suffix /2024-01-01/2024-01-31", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"cycleDayEntries":[]}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testTokens(), WithBaseURL(server.URL))
	data, err := client.GetMenstrualCycleData(context.Background(), "2024-01-01", "2024-01-31")
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

func TestGetMenstrualCycleData_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/periodichealth-service/menstrualcycle/dayview/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testTokens(), WithBaseURL(server.URL))
	_, err := client.GetMenstrualCycleData(context.Background(), "2024-01-01", "2024-01-31")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetMenstrualCycleSummary_Success(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/periodichealth-service/menstrualcycle/summary/", func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/2024-01-01/2024-01-31") {
			t.Errorf("path = %s, want suffix /2024-01-01/2024-01-31", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"averageCycleLength":28}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testTokens(), WithBaseURL(server.URL))
	data, err := client.GetMenstrualCycleSummary(context.Background(), "2024-01-01", "2024-01-31")
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

func TestGetPregnancySummary_Success(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/periodichealth-service/pregnancy/summary", func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"pregnancyStatus":"active"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testTokens(), WithBaseURL(server.URL))
	data, err := client.GetPregnancySummary(context.Background())
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

func TestGetPregnancySummary_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/periodichealth-service/pregnancy/summary", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testTokens(), WithBaseURL(server.URL))
	_, err := client.GetPregnancySummary(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRequestReload_Success(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/wellness-service/wellness/epoch/request/", func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/2024-01-15") {
			t.Errorf("path = %s, want suffix /2024-01-15", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"OK"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testTokens(), WithBaseURL(server.URL))
	data, err := client.RequestReload(context.Background(), "2024-01-15")
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

func TestRequestReload_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/wellness-service/wellness/epoch/request/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testTokens(), WithBaseURL(server.URL))
	_, err := client.RequestReload(context.Background(), "2024-01-15")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
