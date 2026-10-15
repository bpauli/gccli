package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bpauli/gccli/internal/outfmt"
)

func goalsTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	serveDisplayName(mux)

	mux.HandleFunc("/goal-service/goal/goals", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":1,"goalType":"steps","targetValue":10000,"status":"active"}]`))
	})

	mux.HandleFunc("/badge-service/badge/earned", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"badgeId":42,"badgeName":"5K Runner","badgeProgressValue":1.0}]`))
	})

	mux.HandleFunc("/badge-service/badge/available", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"badgeId":100,"badgeName":"Marathon Runner"}]`))
	})

	mux.HandleFunc("/badge-service/badge/in-progress", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"badgeId":50,"badgeName":"10K Beginner","badgeProgressValue":0.5}]`))
	})

	mux.HandleFunc("/challenge-service/challenge/joined", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"challengeId":200,"challengeName":"January Step Challenge"}]`))
	})

	mux.HandleFunc("/badge-service/badge/challenges", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"badgeId":300,"badgeName":"Step Challenge Badge"}]`))
	})

	mux.HandleFunc("/personalrecord-service/personalrecord/prs/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"typeId":1,"typeName":"Longest Run","value":42195.0}]`))
	})

	return httptest.NewServer(mux)
}

// --- Execute-level tests ---

func TestExecute_GoalsHelp(t *testing.T) {
	code := Execute([]string{"goals", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_GoalsListHelp(t *testing.T) {
	code := Execute([]string{"goals", "list", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_BadgesHelp(t *testing.T) {
	code := Execute([]string{"badges", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_BadgesEarnedHelp(t *testing.T) {
	code := Execute([]string{"badges", "earned", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_BadgesAvailableHelp(t *testing.T) {
	code := Execute([]string{"badges", "available", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_BadgesInProgressHelp(t *testing.T) {
	code := Execute([]string{"badges", "in-progress", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_ChallengesHelp(t *testing.T) {
	code := Execute([]string{"challenges", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_ChallengesBadgeHelp(t *testing.T) {
	code := Execute([]string{"challenges", "badge", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_RecordsHelp(t *testing.T) {
	code := Execute([]string{"records", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_GoalsListSendsStatus(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"goals", "list"}, "active"},
		{[]string{"goals"}, "active"},
		{[]string{"goals", "list", "--status", "past"}, "past"},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			tmp := t.TempDir()
			t.Setenv("HOME", tmp)
			t.Setenv("XDG_CONFIG_HOME", tmp)
			var got []string
			mux := http.NewServeMux()
			mux.HandleFunc("/goal-service/goal/goals", func(w http.ResponseWriter, r *http.Request) {
				got = append(got, r.URL.Query().Get("status"))
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`[]`))
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			store := newTestSecretsStore(t)
			overrideLoadSecrets(t, store)
			overrideNewClient(t, server)
			storeTestTokens(t, store, "test@example.com", testTokens())

			args := append([]string{"--json", "--account", "test@example.com"}, tt.args...)
			if code := Execute(args, "1.0.0", "abc123", "2024-01-01"); code != 0 {
				t.Fatalf("exit code = %d, want 0", code)
			}
			if len(got) != 1 || got[0] != tt.want {
				t.Fatalf("status sent = %q, want [%q]", got, tt.want)
			}
		})
	}
}

func TestExecute_GoalsListRejectsUnknownStatus(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/goal-service/goal/goals", func(w http.ResponseWriter, _ *http.Request) {
		called = true
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	code := Execute([]string{"--account", "test@example.com", "goals", "list", "--status", "completed"}, "1.0.0", "abc123", "2024-01-01")
	if code == 0 {
		t.Fatal("exit code = 0, want non-zero for an unknown status")
	}
	if called {
		t.Fatal("API called with an unknown status")
	}
}

// --- GoalsListCmd tests ---

func TestGoalsList_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &GoalsListCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGoalsList_Success(t *testing.T) {
	server := goalsTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &GoalsListCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestGoalsList_WithStatus(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/goal-service/goal/goals", func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.URL.Query().Get("status") != "active" {
			t.Errorf("status = %s, want active", r.URL.Query().Get("status"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":1,"status":"active"}]`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &GoalsListCmd{Status: "active"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !called {
		t.Fatal("expected API to be called")
	}
}

func TestGoalsList_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/goal-service/goal/goals", func(w http.ResponseWriter, _ *http.Request) {
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
	cmd := &GoalsListCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- BadgesEarnedCmd tests ---

func TestBadgesEarned_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &BadgesEarnedCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBadgesEarned_Success(t *testing.T) {
	server := goalsTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &BadgesEarnedCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// --- BadgesAvailableCmd tests ---

func TestBadgesAvailable_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &BadgesAvailableCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBadgesAvailable_Success(t *testing.T) {
	server := goalsTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &BadgesAvailableCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// --- BadgesInProgressCmd tests ---

func TestBadgesInProgress_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &BadgesInProgressCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBadgesInProgress_Success(t *testing.T) {
	server := goalsTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &BadgesInProgressCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// --- ChallengesListCmd tests ---

func TestChallengesList_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &ChallengesListCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestChallengesList_Success(t *testing.T) {
	server := goalsTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &ChallengesListCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestChallengesList_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/challenge-service/challenge/joined", func(w http.ResponseWriter, _ *http.Request) {
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
	cmd := &ChallengesListCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- ChallengesBadgeCmd tests ---

func TestChallengesBadge_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &ChallengesBadgeCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestChallengesBadge_Success(t *testing.T) {
	server := goalsTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &ChallengesBadgeCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// --- RecordsCmd tests ---

func TestRecords_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &RecordsCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRecords_Success(t *testing.T) {
	server := goalsTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &RecordsCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestRecords_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	serveDisplayName(mux)
	mux.HandleFunc("/personalrecord-service/personalrecord/prs/", func(w http.ResponseWriter, _ *http.Request) {
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
	cmd := &RecordsCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
