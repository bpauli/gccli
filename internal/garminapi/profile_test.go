package garminapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bpauli/gccli/internal/units"
)

func TestProfileEndpoints(t *testing.T) {
	testEndpoints(t, profileEndpoints, []endpointCase{
		{"profile.get", nil, "GET /userprofile-service/userprofile/settings"},
		{"profile.user-settings", nil, "GET /userprofile-service/userprofile/user-settings"},
		{"profile.social", map[string]string{"displayName": "testuser"}, "GET /userprofile-service/socialProfile/testuser"},
		{"profile.privacy-settings", nil, "GET /privacysettings-service/privacy-settings"},
		{"profile.privacy-zones", nil, "GET /privacyzone-service/privacyZones"},
		{"profile.privacy-zone.create", nil, "POST /privacyzone-service/privacyZones"},
		{"profile.privacy-zone.delete", map[string]string{"privacyZoneID": "123"}, "DELETE /privacyzone-service/privacyZones/123"},
		{"profile.hr-zones", nil, "GET /biometric-service/heartRateZones/v2"},
		{"profile.power-zones", nil, "GET /biometric-service/powerZones/sports/all"},
	})
}

func TestGetDisplayName_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/userprofile-service/userprofile/settings", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"displayName":"testuser","timeZone":"America/New_York"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testTokens(), WithHTTPClient(server.Client()), WithBaseURL(server.URL))
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

	client := NewClient(testTokens(), WithHTTPClient(server.Client()), WithBaseURL(server.URL))
	_, err := client.GetDisplayName(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetSocialProfile_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/userprofile-service/userprofile/settings", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"displayName":"testuser"}`))
	})
	mux.HandleFunc("/userprofile-service/socialProfile/testuser", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"displayName":"testuser","profileId":12345678}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	client := NewClient(testTokens(), WithHTTPClient(srv.Client()), WithBaseURL(srv.URL))
	data, err := client.GetSocialProfile(context.Background())
	if err != nil {
		t.Fatalf("GetSocialProfile: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["displayName"] != "testuser" {
		t.Errorf("displayName = %v, want testuser", result["displayName"])
	}
	if result["profileId"] != float64(12345678) {
		t.Errorf("profileId = %v, want 12345678", result["profileId"])
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
	return NewClient(testTokens(), WithHTTPClient(server.Client()), WithBaseURL(server.URL))
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

func TestGetProfilePK(t *testing.T) {
	const invalidPK = "user profile profileId field is not an unsigned integer"
	tests := []struct {
		name    string
		profile string
		want    string
		wantErr string
	}{
		{"number", `{"profileId":12345678901234567,"displayName":"tester"}`, "12345678901234567", ""},
		{"30-digit number", `{"profileId":123456789012345678901234567890}`, "123456789012345678901234567890", ""},
		{"digit string", `{"profileId":"12345678"}`, "12345678", ""},
		{"missing profileId", `{"displayName":"tester"}`, "", "user profile missing profileId field"},
		{"null", `{"profileId":null}`, "", invalidPK},
		{"object", `{"profileId":{"a":1}}`, "", invalidPK},
		{"array", `{"profileId":[1]}`, "", invalidPK},
		{"bool", `{"profileId":true}`, "", invalidPK},
		{"empty string", `{"profileId":""}`, "", invalidPK},
		{"non-digit string", `{"profileId":"12a"}`, "", invalidPK},
		{"float", `{"profileId":1.5}`, "", invalidPK},
		{"exponent", `{"profileId":1e3}`, "", invalidPK},
		{"negative", `{"profileId":-5}`, "", invalidPK},
		{"bad JSON", `{"profileId":`, "", "parse user profile: unexpected EOF"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/userprofile-service/userprofile/settings", func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{"displayName":"tester"}`))
			})
			mux.HandleFunc("/userprofile-service/socialProfile/tester", func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(tt.profile))
			})
			_, client := testServer(t, mux)

			got, err := client.GetProfilePK(context.Background())
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("err = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("GetProfilePK = %q, %v, want %q", got, err, tt.want)
			}
		})
	}
}
