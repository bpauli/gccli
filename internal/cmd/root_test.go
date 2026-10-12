package cmd

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/bpauli/gccli/internal/ui"
)

func TestExecute_Version(t *testing.T) {
	code := Execute([]string{"--version"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_Help(t *testing.T) {
	code := Execute([]string{"--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_AuthHelp(t *testing.T) {
	code := Execute([]string{"auth", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_Auth(t *testing.T) {
	code := Execute([]string{"auth"}, "1.0.0", "abc123", "2024-01-01")
	if code == 0 {
		t.Fatal("expected non-zero exit code for auth without subcommand")
	}
}

func TestExecute_UnknownCommand(t *testing.T) {
	code := Execute([]string{"nonexistent"}, "1.0.0", "abc123", "2024-01-01")
	if code == 0 {
		t.Fatal("expected non-zero exit code for unknown command")
	}
}

func TestExecute_NoArgs(t *testing.T) {
	code := Execute([]string{}, "1.0.0", "abc123", "2024-01-01")
	if code == 0 {
		t.Fatal("expected non-zero exit code when no command is given")
	}
}

func TestExitCode_Nil(t *testing.T) {
	if got := exitCode(nil); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}

func TestExitCode_ExitError(t *testing.T) {
	err := &ExitError{Code: 42}
	if got := exitCode(err); got != 42 {
		t.Fatalf("expected 42, got %d", got)
	}
}

func TestExitCode_GenericError(t *testing.T) {
	err := &ExitError{Code: 2, Err: nil}
	if got := exitCode(err); got != 2 {
		t.Fatalf("expected 2, got %d", got)
	}
}

func TestExitError_Error(t *testing.T) {
	err := &ExitError{Code: 1, Err: nil}
	if !strings.Contains(err.Error(), "exit code 1") {
		t.Fatalf("unexpected error message: %s", err.Error())
	}
}

func TestExitError_Unwrap(t *testing.T) {
	inner := &ExitError{Code: 0}
	outer := &ExitError{Code: 1, Err: inner}
	if outer.Unwrap() != inner {
		t.Fatal("Unwrap did not return inner error")
	}
}

func TestPrintNextPageHint(t *testing.T) {
	var buf bytes.Buffer
	u := ui.NewWithWriter(&buf, "never")

	// No hint when all results are shown.
	PrintNextPageHint(u, 0, 20, 10)
	if buf.String() != "" {
		t.Fatalf("expected no output, got %q", buf.String())
	}

	// Hint when there are more results.
	PrintNextPageHint(u, 0, 20, 50)
	if !strings.Contains(buf.String(), "--start 20") {
		t.Fatalf("expected hint with --start 20, got %q", buf.String())
	}
}

func TestTableWriter(t *testing.T) {
	tw := TableWriter()
	if tw == nil {
		t.Fatal("expected non-nil tabwriter")
	}
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stderr
	os.Stderr = w
	t.Cleanup(func() { os.Stderr = orig })

	done := make(chan string)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()
	fn()
	_ = w.Close()
	os.Stderr = orig
	return <-done
}

func TestExecute_RateLimitShowsHint(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var code int
	stderr := captureStderr(t, func() {
		code = Execute([]string{"--color", "never", "--account", "test@example.com", "badges", "earned"}, "1.0.0", "abc123", "2024-01-01")
	})
	if code == 0 {
		t.Fatal("exit code = 0, want non-zero")
	}
	if want := "Rate limited. Wait 30 and retry."; !strings.Contains(stderr, want) {
		t.Fatalf("stderr = %q, want it to contain %q", stderr, want)
	}
}
