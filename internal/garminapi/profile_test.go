package garminapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetProfile_Success(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/userprofile-service/usersocialprofile", func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"displayName":"Test User","profileImageUrlLarge":"https://example.com/img.jpg"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testTokens(), WithBaseURL(server.URL))
	data, err := client.GetProfile(context.Background())
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

func TestGetProfile_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/userprofile-service/usersocialprofile", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testTokens(), WithBaseURL(server.URL))
	_, err := client.GetProfile(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetUserSettings_Success(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/userprofile-service/userprofile/user-settings", func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"userData":{"weight":75.5,"height":180}}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testTokens(), WithBaseURL(server.URL))
	data, err := client.GetUserSettings(context.Background())
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

func TestGetUserSettings_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/userprofile-service/userprofile/user-settings", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testTokens(), WithBaseURL(server.URL))
	_, err := client.GetUserSettings(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
