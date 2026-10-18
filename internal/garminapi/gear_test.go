package garminapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGearEndpoints(t *testing.T) {
	link := map[string]string{"gearUUID": "abc-123", "activityID": "456"}
	testEndpoints(t, gearEndpoints, []endpointCase{
		{"gear.list", map[string]string{"userProfilePK": "12345"}, "GET /gear-service/gear/filterGear?userProfilePk=12345"},
		{"gear.stats", map[string]string{"gearUUID": "abc-123"}, "GET /gear-service/gear/stats/abc-123"},
		{"gear.activities", map[string]string{"gearUUID": "abc-123", "limit": "20"}, "GET /activitylist-service/activities/abc-123/gear?limit=20"},
		{"gear.defaults", map[string]string{"userProfileNumber": "12345"}, "GET /gear-service/gear/user/12345/activityTypes"},
		{"gear.link", link, "PUT /gear-service/gear/link/abc-123/activity/456"},
		{"gear.unlink", link, "PUT /gear-service/gear/unlink/abc-123/activity/456"},
	})
}

// --- GetSocialProfile tests ---

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
