package garminapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bpauli/gccli/internal/units"
)

func TestGetProfile_Success(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/userprofile-service/userprofile/settings", func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"displayName":"Test User","timeZone":"Europe/Paris"}`))
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
	mux.HandleFunc("/userprofile-service/userprofile/settings", func(w http.ResponseWriter, _ *http.Request) {
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

func TestGetDisplayName_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/userprofile-service/userprofile/settings", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"displayName":"testuser","timeZone":"America/New_York"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testTokens(), WithBaseURL(server.URL))
	name, err := client.GetDisplayName(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "testuser" {
		t.Errorf("displayName = %q, want testuser", name)
	}
}

func TestGetDisplayName_MissingField(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/userprofile-service/userprofile/settings", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"timeZone":"America/New_York"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testTokens(), WithBaseURL(server.URL))
	_, err := client.GetDisplayName(context.Background())
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

func userSettingsServer(t *testing.T, status int, body string) *Client {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/userprofile-service/userprofile/user-settings", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return NewClient(testTokens(), WithBaseURL(server.URL))
}

func TestGetMeasurementSystem(t *testing.T) {
	tests := []struct {
		name string
		body string
		want units.System
	}{
		{"metric", `{"userData":{"measurementSystem":"metric","weight":75500}}`, units.Metric},
		{"statute_us", `{"userData":{"measurementSystem":"statute_us"}}`, units.StatuteUS},
		{"statute_uk", `{"userData":{"measurementSystem":"statute_uk"}}`, units.StatuteUK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := userSettingsServer(t, http.StatusOK, tt.body)
			got, err := client.GetMeasurementSystem(context.Background())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("GetMeasurementSystem() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetMeasurementSystem_InvalidPayload(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"missing field", `{"userData":{"weight":75500}}`},
		{"missing userData", `{"id":1}`},
		{"unknown value", `{"userData":{"measurementSystem":"furlongs"}}`},
		{"not json", `<html>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := userSettingsServer(t, http.StatusOK, tt.body)
			if got, err := client.GetMeasurementSystem(context.Background()); err == nil {
				t.Fatalf("expected error, got %v", got)
			}
		})
	}
}

func TestGetMeasurementSystem_HTTPError(t *testing.T) {
	client := userSettingsServer(t, http.StatusNotFound, `{"message":"not found"}`)
	_, err := client.GetMeasurementSystem(context.Background())
	var apiErr *GarminAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *GarminAPIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
}
