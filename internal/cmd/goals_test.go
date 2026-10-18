package cmd

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- Execute-level tests ---

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
