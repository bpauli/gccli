package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bpauli/gccli/internal/outfmt"
)

func gearTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/userprofile-service/usersocialprofile", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"userProfileNumber":12345678,"displayName":"Test User","profileId":87654321}`))
	})

	mux.HandleFunc("/gear-service/gear/filterGear", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"uuid":"abc-123","displayName":"Nike Pegasus 40","gearMakeName":"Nike"}]`))
	})

	mux.HandleFunc("/gear-service/gear/stats/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"totalDistance":1234567.0,"totalActivities":42}`))
	})

	mux.HandleFunc("/activitylist-service/activities/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"activityId":100,"activityName":"Morning Run"}]`))
	})

	mux.HandleFunc("/gear-service/gear/user/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"activityTypePk":1,"gearUuid":"abc-123"}]`))
	})

	mux.HandleFunc("/gear-service/gear/link/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("/gear-service/gear/unlink/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	return httptest.NewServer(mux)
}

// --- Execute-level tests ---

func TestExecute_GearHelp(t *testing.T) {
	code := Execute([]string{"gear", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_GearListHelp(t *testing.T) {
	code := Execute([]string{"gear", "list", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_GearStatsHelp(t *testing.T) {
	code := Execute([]string{"gear", "stats", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_GearActivitiesHelp(t *testing.T) {
	code := Execute([]string{"gear", "activities", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_GearDefaultsHelp(t *testing.T) {
	code := Execute([]string{"gear", "defaults", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_GearLinkHelp(t *testing.T) {
	code := Execute([]string{"gear", "link", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_GearUnlinkHelp(t *testing.T) {
	code := Execute([]string{"gear", "unlink", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

// --- GearListCmd tests ---

func TestGearList_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &GearListCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGearList_Success(t *testing.T) {
	server := gearTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &GearListCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestGearList_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/userprofile-service/usersocialprofile", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"userProfileNumber":12345678}`))
	})
	mux.HandleFunc("/gear-service/gear/filterGear", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &GearListCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGearList_ProfileError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/userprofile-service/usersocialprofile", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &GearListCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "get user profile") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- GearStatsCmd tests ---

func TestGearStats_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &GearStatsCmd{UUID: "abc-123"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGearStats_Success(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/gear-service/gear/stats/abc-123", func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"totalDistance":1234567.0,"totalActivities":42}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &GearStatsCmd{UUID: "abc-123"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !called {
		t.Fatal("expected API to be called")
	}
}

// --- GearActivitiesCmd tests ---

func TestGearActivities_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &GearActivitiesCmd{UUID: "abc-123", Limit: 20}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGearActivities_Success(t *testing.T) {
	server := gearTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &GearActivitiesCmd{UUID: "abc-123", Limit: 20}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestGearActivities_LimitParam(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/activitylist-service/activities/abc-123/gear", func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.URL.Query().Get("limit") != "50" {
			t.Errorf("limit = %s, want 50", r.URL.Query().Get("limit"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &GearActivitiesCmd{UUID: "abc-123", Limit: 50}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !called {
		t.Fatal("expected API to be called")
	}
}

// --- GearDefaultsCmd tests ---

func TestGearDefaults_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &GearDefaultsCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGearDefaults_Success(t *testing.T) {
	server := gearTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &GearDefaultsCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// --- GearLinkCmd tests ---

func TestGearLink_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &GearLinkCmd{UUID: "abc-123", ActivityID: "100"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGearLink_Success(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/gear-service/gear/link/abc-123/activity/100", func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &GearLinkCmd{UUID: "abc-123", ActivityID: "100"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !called {
		t.Fatal("expected API to be called")
	}
	if !strings.Contains(buf.String(), "Linked gear abc-123 to activity 100") {
		t.Errorf("output = %q, want success message", buf.String())
	}
}

func TestGearLink_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/gear-service/gear/link/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &GearLinkCmd{UUID: "abc-123", ActivityID: "100"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- GearUnlinkCmd tests ---

func TestGearUnlink_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &GearUnlinkCmd{UUID: "abc-123", ActivityID: "100"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGearUnlink_Success(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/gear-service/gear/unlink/abc-123/activity/100", func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &GearUnlinkCmd{UUID: "abc-123", ActivityID: "100"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !called {
		t.Fatal("expected API to be called")
	}
	if !strings.Contains(buf.String(), "Unlinked gear abc-123 from activity 100") {
		t.Errorf("output = %q, want success message", buf.String())
	}
}

func TestGearUnlink_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/gear-service/gear/unlink/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &GearUnlinkCmd{UUID: "abc-123", ActivityID: "100"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- getUserProfilePK tests ---

func TestGetUserProfilePK_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/userprofile-service/usersocialprofile", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"userProfileNumber":12345678,"displayName":"Test User"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	client, err := resolveClient(g)
	if err != nil {
		t.Fatalf("resolveClient: %v", err)
	}

	pk, err := getUserProfilePK(g, client)
	if err != nil {
		t.Fatalf("getUserProfilePK: %v", err)
	}
	if pk != "12345678" {
		t.Errorf("profilePK = %s, want 12345678", pk)
	}
}

func TestGetUserProfilePK_MissingField(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/userprofile-service/usersocialprofile", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"displayName":"Test User"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	client, err := resolveClient(g)
	if err != nil {
		t.Fatalf("resolveClient: %v", err)
	}

	_, err = getUserProfilePK(g, client)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "userProfileNumber") {
		t.Fatalf("unexpected error: %v", err)
	}
}
