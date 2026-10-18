package cmd

import (
	"cmp"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/bpauli/gccli/internal/garminapi"
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

func captureOutput(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()
	capture := func(target **os.File) func() string {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		orig := *target
		*target = w
		t.Cleanup(func() {
			_ = w.Close()
			*target = orig
		})
		done := make(chan string)
		go func() {
			b, _ := io.ReadAll(r)
			done <- string(b)
		}()
		return func() string {
			_ = w.Close()
			*target = orig
			return <-done
		}
	}
	finishOut := capture(&os.Stdout)
	finishErr := capture(&os.Stderr)
	fn()
	return finishOut(), finishErr()
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
	_, stderr := captureOutput(t, func() {
		code = Execute([]string{"--color", "never", "--account", "test@example.com", "badges", "earned"}, "1.0.0", "abc123", "2024-01-01")
	})
	if code == 0 {
		t.Fatal("exit code = 0, want non-zero")
	}
	if want := "Rate limited. Wait 30 and retry."; !strings.Contains(stderr, want) {
		t.Fatalf("stderr = %q, want it to contain %q", stderr, want)
	}
}

func TestExecute_ExitCodes(t *testing.T) {
	tests := []struct {
		name   string
		status int
		args   []string
		want   int
	}{
		{name: "not found", status: http.StatusNotFound, args: []string{"badges", "earned"}, want: 4},
		{name: "rate limited", status: http.StatusTooManyRequests, args: []string{"badges", "earned"}, want: 5},
		{name: "server error", status: http.StatusInternalServerError, args: []string{"badges", "earned"}, want: 1},
		{name: "no stored credentials", args: []string{"--account", "nobody@example.com", "badges", "earned"}, want: 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			t.Setenv("HOME", tmp)
			t.Setenv("XDG_CONFIG_HOME", tmp)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
			}))
			defer server.Close()

			store := newTestSecretsStore(t)
			overrideLoadSecrets(t, store)
			overrideNewClient(t, server)
			storeTestTokens(t, store, "test@example.com", testTokens())

			args := append([]string{"--account", "test@example.com"}, tt.args...)
			var code int
			captureOutput(t, func() {
				code = Execute(args, "1.0.0", "abc123", "2024-01-01")
			})
			if code != tt.want {
				t.Fatalf("exit code = %d, want %d", code, tt.want)
			}
		})
	}
}

func TestExitCode_TokenExpired(t *testing.T) {
	err := fmt.Errorf("get goals: %w", &garminapi.TokenExpiredError{Email: "a@b.com"})
	if got := exitCode(err); got != 3 {
		t.Fatalf("exit code = %d, want 3", got)
	}
}

func TestExecute_UsageErrorGoesToStderrWithExitCode2(t *testing.T) {
	var code int
	stdout, stderr := captureOutput(t, func() {
		code = Execute([]string{"--bogus"}, "1.0.0", "abc123", "2024-01-01")
	})
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "unknown flag --bogus") || !strings.Contains(stderr, "Usage:") {
		t.Fatalf("stderr = %q, want the error and the usage text", stderr)
	}
}

func TestExecute_HeaderWithoutPlainIsUsageError(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		args []string
	}{
		{name: "table", args: []string{"--header"}},
		{name: "json", args: []string{"--json", "--header"}},
		{name: "json beats plain", args: []string{"--json", "--plain", "--header"}},
		{name: "env header", env: map[string]string{"GCCLI_HEADER": "1"}},
		{name: "env json", env: map[string]string{"GCCLI_JSON": "1"}, args: []string{"--plain", "--header"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			t.Setenv("HOME", tmp)
			t.Setenv("XDG_CONFIG_HOME", tmp)
			t.Setenv("GCCLI_ACCOUNT", "")
			for _, k := range []string{"GCCLI_JSON", "GCCLI_PLAIN", "GCCLI_HEADER"} {
				t.Setenv(k, cmp.Or(tt.env[k], "0"))
			}

			var code int
			stdout, stderr := captureOutput(t, func() {
				code = Execute(append(tt.args, "courses", "list"), "1.0.0", "abc123", "2024-01-01")
			})
			if code != 2 {
				t.Fatalf("exit code = %d, want 2", code)
			}
			if stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
			if !strings.Contains(stderr, "gccli: error: --header works only with --plain\n") || !strings.Contains(stderr, "Usage:") {
				t.Fatalf("stderr = %q, want the --header error and the usage text", stderr)
			}
		})
	}
}

func TestExecute_AuthCommandsNeedLogin(t *testing.T) {
	tests := [][]string{
		{"auth", "status"},
		{"auth", "token"},
		{"auth", "export"},
		{"--account", "nobody@example.com", "auth", "status"},
		{"--account", "nobody@example.com", "auth", "token"},
		{"--account", "nobody@example.com", "auth", "export"},
	}
	for _, args := range tests {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			tmp := t.TempDir()
			t.Setenv("HOME", tmp)
			t.Setenv("XDG_CONFIG_HOME", tmp)
			t.Setenv("GCCLI_ACCOUNT", "")
			overrideLoadSecrets(t, newTestSecretsStore(t))

			var code int
			captureOutput(t, func() {
				code = Execute(args, "1.0.0", "abc123", "2024-01-01")
			})
			if code != 3 {
				t.Fatalf("exit code = %d, want 3", code)
			}
		})
	}
}

func TestExecute_AuthStatusWithoutCredentialsPrintsJSONError(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)
	overrideLoadSecrets(t, newTestSecretsStore(t))

	var code int
	stdout, stderr := captureOutput(t, func() {
		code = Execute([]string{"--json", "--account", "nobody@example.com", "auth", "status"}, "1.0.0", "abc123", "2024-01-01")
	})
	if code != 3 {
		t.Fatalf("exit code = %d, want 3", code)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	want := `{"error":{"code":"auth_required","message":"no credentials stored for nobody@example.com; run: gccli auth login nobody@example.com"}}` + "\n"
	if stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
}

func TestExecute_JSONModePrintsErrorObject(t *testing.T) {
	tests := []struct {
		name   string
		status int
		want   string
	}{
		{
			name:   "rate limited",
			status: http.StatusTooManyRequests,
			want:   `{"error":{"code":"rate_limited","status":429,"message":"Rate limited. Wait 30 and retry."}}`,
		},
		{
			name:   "not found",
			status: http.StatusNotFound,
			want:   `{"error":{"code":"not_found","status":404,"message":"get earned badges: garmin api: 404 Not Found: gone"}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			t.Setenv("HOME", tmp)
			t.Setenv("XDG_CONFIG_HOME", tmp)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Retry-After", "30")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte("gone"))
			}))
			defer server.Close()

			store := newTestSecretsStore(t)
			overrideLoadSecrets(t, store)
			overrideNewClient(t, server)
			storeTestTokens(t, store, "test@example.com", testTokens())

			var code int
			stdout, stderr := captureOutput(t, func() {
				code = Execute([]string{"--json", "--account", "test@example.com", "badges", "earned"}, "1.0.0", "abc123", "2024-01-01")
			})
			if code == 0 {
				t.Fatal("exit code = 0, want non-zero")
			}
			if stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
			if got := strings.TrimSpace(stderr); got != tt.want {
				t.Fatalf("stderr = %s\nwant     %s", got, tt.want)
			}
		})
	}
}
