package garminapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

// --- GetGoals tests ---

func TestGetGoals_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/goal-service/goal/goals" {
			t.Errorf("path = %s, want /goal-service/goal/goals", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"id":1,"goalType":"steps","targetValue":10000}]`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetGoals(context.Background(), "")
	if err != nil {
		t.Fatalf("GetGoals: %v", err)
	}

	var result []map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 goal, got %d", len(result))
	}
	if result[0]["goalType"] != "steps" {
		t.Errorf("goalType = %v, want steps", result[0]["goalType"])
	}
}

func TestGetGoals_WithStatus(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/goal-service/goal/goals" {
			t.Errorf("path = %s, want /goal-service/goal/goals", r.URL.Path)
		}
		if r.URL.Query().Get("status") != "active" {
			t.Errorf("status = %s, want active", r.URL.Query().Get("status"))
		}
		_, _ = w.Write([]byte(`[{"id":1,"goalType":"steps","status":"active"}]`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetGoals(context.Background(), "active")
	if err != nil {
		t.Fatalf("GetGoals: %v", err)
	}

	var result []map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 goal, got %d", len(result))
	}
}

func TestGetGoals_ServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})

	_, client := testServer(t, handler)
	_, err := client.GetGoals(context.Background(), "")
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *GarminAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected GarminAPIError, got %T: %v", err, err)
	}
}

// --- GetBadgesEarned tests ---

func TestGetBadgesEarned_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/badge-service/badge/earned" {
			t.Errorf("path = %s, want /badge-service/badge/earned", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"badgeId":42,"badgeName":"5K Runner","badgeProgressValue":1.0}]`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetBadgesEarned(context.Background())
	if err != nil {
		t.Fatalf("GetBadgesEarned: %v", err)
	}

	var result []map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 badge, got %d", len(result))
	}
	if result[0]["badgeName"] != "5K Runner" {
		t.Errorf("badgeName = %v, want 5K Runner", result[0]["badgeName"])
	}
}

func TestGetBadgesEarned_ServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})

	_, client := testServer(t, handler)
	_, err := client.GetBadgesEarned(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *GarminAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected GarminAPIError, got %T: %v", err, err)
	}
}

// --- GetBadgesAvailable tests ---

func TestGetBadgesAvailable_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/badge-service/badge/available" {
			t.Errorf("path = %s, want /badge-service/badge/available", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"badgeId":100,"badgeName":"Marathon Runner"}]`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetBadgesAvailable(context.Background())
	if err != nil {
		t.Fatalf("GetBadgesAvailable: %v", err)
	}

	var result []map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 badge, got %d", len(result))
	}
	if result[0]["badgeName"] != "Marathon Runner" {
		t.Errorf("badgeName = %v, want Marathon Runner", result[0]["badgeName"])
	}
}

// --- GetBadgesInProgress tests ---

func TestGetBadgesInProgress_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/badge-service/badge/in-progress" {
			t.Errorf("path = %s, want /badge-service/badge/in-progress", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"badgeId":50,"badgeName":"10K Beginner","badgeProgressValue":0.5}]`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetBadgesInProgress(context.Background())
	if err != nil {
		t.Fatalf("GetBadgesInProgress: %v", err)
	}

	var result []map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 badge, got %d", len(result))
	}
	if result[0]["badgeName"] != "10K Beginner" {
		t.Errorf("badgeName = %v, want 10K Beginner", result[0]["badgeName"])
	}
}

// --- GetChallenges tests ---

func TestGetChallenges_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/challenge-service/challenge/joined" {
			t.Errorf("path = %s, want /challenge-service/challenge/joined", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"challengeId":200,"challengeName":"January Step Challenge"}]`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetChallenges(context.Background())
	if err != nil {
		t.Fatalf("GetChallenges: %v", err)
	}

	var result []map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 challenge, got %d", len(result))
	}
	if result[0]["challengeName"] != "January Step Challenge" {
		t.Errorf("challengeName = %v, want January Step Challenge", result[0]["challengeName"])
	}
}

func TestGetChallenges_ServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})

	_, client := testServer(t, handler)
	_, err := client.GetChallenges(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *GarminAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected GarminAPIError, got %T: %v", err, err)
	}
}

// --- GetBadgeChallenges tests ---

func TestGetBadgeChallenges_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/badge-service/badge/challenges" {
			t.Errorf("path = %s, want /badge-service/badge/challenges", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"badgeId":300,"badgeName":"Step Challenge Badge"}]`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetBadgeChallenges(context.Background())
	if err != nil {
		t.Fatalf("GetBadgeChallenges: %v", err)
	}

	var result []map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 badge challenge, got %d", len(result))
	}
	if result[0]["badgeName"] != "Step Challenge Badge" {
		t.Errorf("badgeName = %v, want Step Challenge Badge", result[0]["badgeName"])
	}
}

// --- GetPersonalRecords tests ---

func TestGetPersonalRecords_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/personalrecord-service/personalrecord/prs/testuser" {
			t.Errorf("path = %s, want /personalrecord-service/personalrecord/prs/testuser", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"typeId":1,"typeName":"Longest Run","value":42195.0}]`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetPersonalRecords(context.Background(), "testuser")
	if err != nil {
		t.Fatalf("GetPersonalRecords: %v", err)
	}

	var result []map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 record, got %d", len(result))
	}
	if result[0]["typeName"] != "Longest Run" {
		t.Errorf("typeName = %v, want Longest Run", result[0]["typeName"])
	}
}

func TestGetPersonalRecords_ServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})

	_, client := testServer(t, handler)
	_, err := client.GetPersonalRecords(context.Background(), "testuser")
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *GarminAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected GarminAPIError, got %T: %v", err, err)
	}
}
