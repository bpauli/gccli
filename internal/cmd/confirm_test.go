package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
)

func useNonTerminalStdin(t *testing.T) {
	t.Helper()
	f, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	orig := confirmReader
	confirmReader = f
	t.Cleanup(func() { confirmReader = orig })
}

func TestConfirm_NonTerminalWithoutForce(t *testing.T) {
	useNonTerminalStdin(t)

	var buf bytes.Buffer
	ok, err := confirm(&buf, "Delete?", false)
	if err == nil {
		t.Fatal("expected an error when stdin is not a terminal and --force is not set")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Fatalf("error should name --force, got: %v", err)
	}
	if ok {
		t.Fatal("expected false")
	}
	if buf.String() != "" {
		t.Fatalf("expected no prompt, got: %q", buf.String())
	}
}

func TestConfirm_NonTerminalWithForce(t *testing.T) {
	useNonTerminalStdin(t)

	ok, err := confirm(&bytes.Buffer{}, "Delete?", true)
	if err != nil || !ok {
		t.Fatalf("confirm = %v, %v; want true, nil", ok, err)
	}
}

func TestExecute_DeleteWithoutForceFailsWhenNotInteractive(t *testing.T) {
	commands := [][]string{
		{"activity", "delete", "1"},
		{"body", "delete-weight", "1", "--date", "2024-01-15"},
		{"body", "delete-blood-pressure", "1", "--date", "2024-01-15"},
		{"courses", "delete", "1"},
		{"events", "delete", "1"},
		{"workouts", "delete", "1"},
		{"workouts", "schedule", "remove", "1"},
	}
	for _, args := range commands {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			tmp := t.TempDir()
			t.Setenv("HOME", tmp)
			t.Setenv("XDG_CONFIG_HOME", tmp)
			useNonTerminalStdin(t)

			var mu sync.Mutex
			var writes []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					mu.Lock()
					writes = append(writes, r.Method+" "+r.URL.Path)
					mu.Unlock()
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{}`))
			}))
			defer server.Close()

			store := newTestSecretsStore(t)
			overrideLoadSecrets(t, store)
			overrideNewClient(t, server)
			storeTestTokens(t, store, "test@example.com", testTokens())

			code := Execute(append([]string{"--account", "test@example.com"}, args...), "1.0.0", "abc123", "2024-01-01")
			if code == 0 {
				t.Fatal("exit code = 0, want non-zero without --force on a non-interactive stdin")
			}
			mu.Lock()
			defer mu.Unlock()
			if len(writes) != 0 {
				t.Fatalf("unexpected write requests: %v", writes)
			}
		})
	}
}

func TestConfirm_Force(t *testing.T) {
	var buf bytes.Buffer
	ok, err := confirm(&buf, "Delete?", true)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !ok {
		t.Fatal("expected true when force=true")
	}
	if buf.String() != "" {
		t.Fatalf("expected no prompt output with force, got: %q", buf.String())
	}
}

func TestConfirm_Yes(t *testing.T) {
	orig := confirmReader
	confirmReader = strings.NewReader("y\n")
	t.Cleanup(func() { confirmReader = orig })

	var buf bytes.Buffer
	ok, err := confirm(&buf, "Delete?", false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !ok {
		t.Fatal("expected true for 'y' input")
	}
	if !strings.Contains(buf.String(), "Delete?") {
		t.Fatalf("expected prompt in output, got: %q", buf.String())
	}
}

func TestConfirm_YesFull(t *testing.T) {
	orig := confirmReader
	confirmReader = strings.NewReader("yes\n")
	t.Cleanup(func() { confirmReader = orig })

	var buf bytes.Buffer
	ok, err := confirm(&buf, "Delete?", false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !ok {
		t.Fatal("expected true for 'yes' input")
	}
}

func TestConfirm_No(t *testing.T) {
	orig := confirmReader
	confirmReader = strings.NewReader("n\n")
	t.Cleanup(func() { confirmReader = orig })

	var buf bytes.Buffer
	ok, err := confirm(&buf, "Delete?", false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if ok {
		t.Fatal("expected false for 'n' input")
	}
}

func TestConfirm_EmptyInput(t *testing.T) {
	orig := confirmReader
	confirmReader = strings.NewReader("\n")
	t.Cleanup(func() { confirmReader = orig })

	var buf bytes.Buffer
	ok, err := confirm(&buf, "Delete?", false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if ok {
		t.Fatal("expected false for empty input (default is no)")
	}
}

func TestConfirm_EOF(t *testing.T) {
	orig := confirmReader
	confirmReader = strings.NewReader("")
	t.Cleanup(func() { confirmReader = orig })

	var buf bytes.Buffer
	ok, err := confirm(&buf, "Delete?", false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if ok {
		t.Fatal("expected false for EOF")
	}
}

func TestConfirm_CaseInsensitive(t *testing.T) {
	orig := confirmReader
	confirmReader = strings.NewReader("Y\n")
	t.Cleanup(func() { confirmReader = orig })

	var buf bytes.Buffer
	ok, err := confirm(&buf, "Delete?", false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !ok {
		t.Fatal("expected true for 'Y' input (case insensitive)")
	}
}

func TestConfirm_WhitespaceTrimmed(t *testing.T) {
	orig := confirmReader
	confirmReader = strings.NewReader("  yes  \n")
	t.Cleanup(func() { confirmReader = orig })

	var buf bytes.Buffer
	ok, err := confirm(&buf, "Delete?", false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !ok {
		t.Fatal("expected true for '  yes  ' input (trimmed)")
	}
}
