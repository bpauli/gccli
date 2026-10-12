package garminapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetNutritionDailyFoodLog_Success(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/nutrition-service/food/logs/2024-01-15", func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"entries":[]}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testTokens(), WithHTTPClient(server.Client()), WithBaseURL(server.URL))
	data, err := client.GetNutritionDailyFoodLog(context.Background(), "2024-01-15")
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

func TestGetNutritionDailyMeals_Success(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/nutrition-service/meals/2024-01-15", func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"meals":[]}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testTokens(), WithHTTPClient(server.Client()), WithBaseURL(server.URL))
	data, err := client.GetNutritionDailyMeals(context.Background(), "2024-01-15")
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

func TestGetNutritionDailySettings_Success(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/nutrition-service/settings/2024-01-15", func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"settings":{}}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testTokens(), WithHTTPClient(server.Client()), WithBaseURL(server.URL))
	data, err := client.GetNutritionDailySettings(context.Background(), "2024-01-15")
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

func TestGetNutritionDailyFoodLog_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/nutrition-service/food/logs/2024-01-15", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testTokens(), WithHTTPClient(server.Client()), WithBaseURL(server.URL))
	_, err := client.GetNutritionDailyFoodLog(context.Background(), "2024-01-15")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
