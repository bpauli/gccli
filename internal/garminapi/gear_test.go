package garminapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

// --- GetSocialProfile tests ---

func TestGetSocialProfile_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/userprofile-service/usersocialprofile" {
			t.Errorf("path = %s, want /userprofile-service/usersocialprofile", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"userProfileNumber":12345678,"displayName":"Test User","profileId":87654321}`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetSocialProfile(context.Background())
	if err != nil {
		t.Fatalf("GetSocialProfile: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["displayName"] != "Test User" {
		t.Errorf("displayName = %v, want Test User", result["displayName"])
	}
	if result["userProfileNumber"] != float64(12345678) {
		t.Errorf("userProfileNumber = %v, want 12345678", result["userProfileNumber"])
	}
}

// --- GetGear tests ---

func TestGetGear_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/gear-service/gear/filterGear" {
			t.Errorf("path = %s, want /gear-service/gear/filterGear", r.URL.Path)
		}
		if r.URL.Query().Get("userProfilePk") != "12345678" {
			t.Errorf("userProfilePk = %s, want 12345678", r.URL.Query().Get("userProfilePk"))
		}
		_, _ = w.Write([]byte(`[{"uuid":"abc-123","displayName":"Nike Pegasus 40","gearMakeName":"Nike"}]`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetGear(context.Background(), "12345678")
	if err != nil {
		t.Fatalf("GetGear: %v", err)
	}

	var result []map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 gear item, got %d", len(result))
	}
	if result[0]["displayName"] != "Nike Pegasus 40" {
		t.Errorf("displayName = %v, want Nike Pegasus 40", result[0]["displayName"])
	}
}

func TestGetGear_ServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})

	_, client := testServer(t, handler)
	_, err := client.GetGear(context.Background(), "12345678")
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *GarminAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected GarminAPIError, got %T: %v", err, err)
	}
}

// --- GetGearStats tests ---

func TestGetGearStats_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/gear-service/gear/stats/abc-123" {
			t.Errorf("path = %s, want /gear-service/gear/stats/abc-123", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"totalDistance":1234567.0,"totalActivities":42}`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetGearStats(context.Background(), "abc-123")
	if err != nil {
		t.Fatalf("GetGearStats: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["totalActivities"] != float64(42) {
		t.Errorf("totalActivities = %v, want 42", result["totalActivities"])
	}
}

func TestGetGearStats_NotFound(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	})

	_, client := testServer(t, handler)
	_, err := client.GetGearStats(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *GarminAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected GarminAPIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", apiErr.StatusCode, http.StatusNotFound)
	}
}

// --- GetGearActivities tests ---

func TestGetGearActivities_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/activitylist-service/activities/abc-123/gear" {
			t.Errorf("path = %s, want /activitylist-service/activities/abc-123/gear", r.URL.Path)
		}
		if r.URL.Query().Get("limit") != "20" {
			t.Errorf("limit = %s, want 20", r.URL.Query().Get("limit"))
		}
		_, _ = w.Write([]byte(`[{"activityId":100,"activityName":"Morning Run"}]`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetGearActivities(context.Background(), "abc-123", 20)
	if err != nil {
		t.Fatalf("GetGearActivities: %v", err)
	}

	var result []map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(result))
	}
	if result[0]["activityName"] != "Morning Run" {
		t.Errorf("activityName = %v, want Morning Run", result[0]["activityName"])
	}
}

func TestGetGearActivities_ServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})

	_, client := testServer(t, handler)
	_, err := client.GetGearActivities(context.Background(), "abc-123", 20)
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *GarminAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected GarminAPIError, got %T: %v", err, err)
	}
}

// --- GetGearDefaults tests ---

func TestGetGearDefaults_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/gear-service/gear/user/12345678/activityTypes" {
			t.Errorf("path = %s, want /gear-service/gear/user/12345678/activityTypes", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"activityTypePk":1,"gearUuid":"abc-123"}]`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetGearDefaults(context.Background(), "12345678")
	if err != nil {
		t.Fatalf("GetGearDefaults: %v", err)
	}

	var result []map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 default, got %d", len(result))
	}
}

// --- LinkGear tests ---

func TestLinkGear_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/gear-service/gear/link/abc-123/activity/100" {
			t.Errorf("path = %s, want /gear-service/gear/link/abc-123/activity/100", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	_, client := testServer(t, handler)
	_, err := client.LinkGear(context.Background(), "abc-123", "100")
	if err != nil {
		t.Fatalf("LinkGear: %v", err)
	}
}

func TestLinkGear_NotFound(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	})

	_, client := testServer(t, handler)
	_, err := client.LinkGear(context.Background(), "nonexistent", "100")
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *GarminAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected GarminAPIError, got %T: %v", err, err)
	}
}

// --- UnlinkGear tests ---

func TestUnlinkGear_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/gear-service/gear/unlink/abc-123/activity/100" {
			t.Errorf("path = %s, want /gear-service/gear/unlink/abc-123/activity/100", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	_, client := testServer(t, handler)
	_, err := client.UnlinkGear(context.Background(), "abc-123", "100")
	if err != nil {
		t.Fatalf("UnlinkGear: %v", err)
	}
}

func TestUnlinkGear_NotFound(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	})

	_, client := testServer(t, handler)
	_, err := client.UnlinkGear(context.Background(), "nonexistent", "100")
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *GarminAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected GarminAPIError, got %T: %v", err, err)
	}
}
