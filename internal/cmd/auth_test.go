package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/99designs/keyring"

	"github.com/bpauli/gccli/internal/config"
	"github.com/bpauli/gccli/internal/garminauth"
	"github.com/bpauli/gccli/internal/outfmt"
	"github.com/bpauli/gccli/internal/secrets"
	"github.com/bpauli/gccli/internal/testutil"
	"github.com/bpauli/gccli/internal/ui"
	"github.com/bpauli/gccli/internal/units"
)

// memKeyring is a simple in-memory keyring implementation for testing.
type memKeyring struct {
	items map[string]keyring.Item
}

func (m *memKeyring) Get(key string) (keyring.Item, error) {
	item, ok := m.items[key]
	if !ok {
		return keyring.Item{}, keyring.ErrKeyNotFound
	}
	return item, nil
}

func (m *memKeyring) GetMetadata(_ string) (keyring.Metadata, error) {
	return keyring.Metadata{}, nil
}

func (m *memKeyring) Set(item keyring.Item) error {
	m.items[item.Key] = item
	return nil
}

func (m *memKeyring) Remove(key string) error {
	if _, ok := m.items[key]; !ok {
		return keyring.ErrKeyNotFound
	}
	delete(m.items, key)
	return nil
}

func (m *memKeyring) Keys() ([]string, error) {
	keys := make([]string, 0, len(m.items))
	for k := range m.items {
		keys = append(keys, k)
	}
	return keys, nil
}

func newTestSecretsStore(t *testing.T) *secrets.Store {
	t.Helper()
	ring := &memKeyring{items: make(map[string]keyring.Item)}
	return secrets.OpenWithKeyring(ring)
}

func testGlobals(t *testing.T, buf *bytes.Buffer, mode outfmt.Mode, account string) *Globals {
	t.Helper()
	u := ui.NewWithWriter(buf, "never")
	ctx := context.Background()
	ctx = outfmt.NewContext(ctx, mode)
	ctx = ui.NewContext(ctx, u)
	return &Globals{
		Context: ctx,
		UI:      u,
		Stdout:  liveOSFile{&os.Stdout},
		Stderr:  liveOSFile{&os.Stderr},
		Account: account,
		Units:   units.Pin(units.Metric),
	}
}

type liveOSFile struct{ f **os.File }

func (o liveOSFile) Write(p []byte) (int, error) { return (*o.f).Write(p) }

func overrideLoadSecrets(t *testing.T, store *secrets.Store) {
	t.Helper()
	orig := loadSecretsStore
	loadSecretsStore = func() (*secrets.Store, error) { return store, nil }
	t.Cleanup(func() { loadSecretsStore = orig })
}

func testTokens() *garminauth.Tokens {
	return &garminauth.Tokens{
		OAuth1Token:        "oauth1-tok",
		OAuth1Secret:       "oauth1-sec",
		OAuth2AccessToken:  "test-access-token",
		OAuth2RefreshToken: "test-refresh-token",
		OAuth2ExpiresAt:    time.Now().Add(time.Hour),
		Domain:             "garmin.com",
		Email:              "test@example.com",
	}
}

func storeTestTokens(t *testing.T, store *secrets.Store, email string, tokens *garminauth.Tokens) {
	t.Helper()
	data, err := tokens.Marshal()
	if err != nil {
		t.Fatalf("marshal tokens: %v", err)
	}
	if err := store.Set(email, data); err != nil {
		t.Fatalf("store tokens: %v", err)
	}
}

// --- Execute-level tests (command parsing) ---

func TestExecute_AuthLoginHelp(t *testing.T) {
	code := Execute([]string{"auth", "login", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_AuthStatusHelp(t *testing.T) {
	code := Execute([]string{"auth", "status", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_AuthRemoveHelp(t *testing.T) {
	code := Execute([]string{"auth", "remove", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecute_AuthTokenHelp(t *testing.T) {
	code := Execute([]string{"auth", "token", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

// --- AuthStatusCmd tests ---

func TestAuthStatus_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &AuthStatusCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuthStatus_NotFound(t *testing.T) {
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "nobody@example.com")
	cmd := &AuthStatusCmd{}
	err := cmd.Run(g)
	if err == nil || !strings.Contains(err.Error(), "no credentials stored for nobody@example.com") {
		t.Fatalf("expected no-credentials error, got: %v", err)
	}
}

func TestAuthStatus_Success(t *testing.T) {
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)

	tokens := testTokens()
	storeTestTokens(t, store, "test@example.com", tokens)

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &AuthStatusCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "Authenticated as test@example.com") {
		t.Fatalf("expected 'Authenticated' message, got: %q", buf.String())
	}
}

func TestAuthStatus_Expired(t *testing.T) {
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)

	tokens := testTokens()
	tokens.OAuth2ExpiresAt = time.Now().Add(-time.Hour)
	storeTestTokens(t, store, "test@example.com", tokens)

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &AuthStatusCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "Token expired") {
		t.Fatalf("expected 'Token expired' warning, got: %q", buf.String())
	}
}

// --- AuthRemoveCmd tests ---

func TestAuthRemove_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &AuthRemoveCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuthRemove_NotFound(t *testing.T) {
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "nobody@example.com")
	cmd := &AuthRemoveCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "No credentials stored") {
		t.Fatalf("expected 'No credentials stored' warning, got: %q", buf.String())
	}
}

func TestAuthRemove_Success(t *testing.T) {
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)

	tokens := testTokens()
	storeTestTokens(t, store, "test@example.com", tokens)

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &AuthRemoveCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "Removed credentials") {
		t.Fatalf("expected 'Removed credentials' message, got: %q", buf.String())
	}

	// Verify tokens were actually removed from the store.
	_, getErr := store.Get("test@example.com")
	if !errors.Is(getErr, secrets.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got: %v", getErr)
	}
}

// --- AuthTokenCmd tests ---

func TestAuthToken_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &AuthTokenCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuthToken_NotFound(t *testing.T) {
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "nobody@example.com")
	cmd := &AuthTokenCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no credentials stored") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func runAuthToken(t *testing.T, stored *garminauth.Tokens, refresh func(context.Context, *garminauth.Tokens, garminauth.LoginOptions) (*garminauth.Tokens, error)) (store *secrets.Store, stdout, stderr string, err error) {
	t.Helper()
	testutil.TempConfigDir(t)
	store = newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	storeTestTokens(t, store, "test@example.com", stored)

	origRefresh := refreshFn
	t.Cleanup(func() { refreshFn = origRefresh })
	refreshFn = refresh

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	stdout, stderr = captureOutput(t, func() { err = (&AuthTokenCmd{}).Run(g) })
	return store, stdout, buf.String() + stderr, err
}

func TestAuthToken_Success(t *testing.T) {
	_, stdout, stderr, err := runAuthToken(t, testTokens(), func(context.Context, *garminauth.Tokens, garminauth.LoginOptions) (*garminauth.Tokens, error) {
		t.Error("refreshFn called for a valid token")
		return nil, fmt.Errorf("unexpected")
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if stdout != "test-access-token\n" || stderr != "" {
		t.Fatalf("stdout = %q, stderr = %q, want the stored token and no stderr", stdout, stderr)
	}
}

func TestAuthToken_ExpiredRefreshes(t *testing.T) {
	expired := testTokens()
	expired.OAuth2ExpiresAt = time.Now().Add(-time.Hour)
	store, stdout, stderr, err := runAuthToken(t, expired, func(_ context.Context, tokens *garminauth.Tokens, _ garminauth.LoginOptions) (*garminauth.Tokens, error) {
		fresh := *tokens
		fresh.OAuth2AccessToken = "AT-1"
		fresh.OAuth2ExpiresAt = time.Now().Add(time.Hour)
		return &fresh, nil
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if stdout != "AT-1\n" || stderr != "" {
		t.Fatalf("stdout = %q, stderr = %q, want the refreshed token and no stderr", stdout, stderr)
	}
	if got := storedAccessToken(t, store); got != "AT-1" {
		t.Errorf("stored token = %q, want AT-1", got)
	}
}

func TestAuthToken_ExpiredRefreshFails(t *testing.T) {
	expired := testTokens()
	expired.OAuth2ExpiresAt = time.Now().Add(-time.Hour)
	store, stdout, _, err := runAuthToken(t, expired, func(context.Context, *garminauth.Tokens, garminauth.LoginOptions) (*garminauth.Tokens, error) {
		return nil, fmt.Errorf("HTTP 400: invalid_grant")
	})
	if code := exitCode(err); code != 3 {
		t.Fatalf("exit code = %d (err %v), want 3", code, err)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if got := storedAccessToken(t, store); got != "test-access-token" {
		t.Errorf("stored token = %q, want the old token kept", got)
	}
}

// --- AuthLoginCmd tests ---

func TestAuthLogin_Browser(t *testing.T) {
	testutil.TempConfigDir(t)
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)

	origBrowser := loginBrowserFn
	loginBrowserFn = func(_ context.Context, email string, _ garminauth.LoginOptions) (*garminauth.Tokens, error) {
		return &garminauth.Tokens{
			OAuth2AccessToken: "browser-access-token",
			OAuth2ExpiresAt:   time.Now().Add(time.Hour),
			Domain:            "garmin.com",
			Email:             email,
		}, nil
	}
	t.Cleanup(func() { loginBrowserFn = origBrowser })

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &AuthLoginCmd{Email: "test@example.com"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "Logged in") {
		t.Fatalf("expected 'Logged in' message, got: %q", buf.String())
	}

	// Verify tokens were stored.
	data, getErr := store.Get("test@example.com")
	if getErr != nil {
		t.Fatalf("expected stored tokens, got: %v", getErr)
	}
	stored, unmarshalErr := garminauth.UnmarshalTokens(data)
	if unmarshalErr != nil {
		t.Fatalf("unmarshal stored tokens: %v", unmarshalErr)
	}
	if stored.OAuth2AccessToken != "browser-access-token" {
		t.Fatalf("expected browser-access-token, got %q", stored.OAuth2AccessToken)
	}
}

func TestAuthLogin_SavesDefaultAccount(t *testing.T) {
	testutil.TempConfigDir(t)
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)

	origBrowser := loginBrowserFn
	loginBrowserFn = func(_ context.Context, email string, _ garminauth.LoginOptions) (*garminauth.Tokens, error) {
		return &garminauth.Tokens{
			OAuth2AccessToken: "access-token",
			OAuth2ExpiresAt:   time.Now().Add(time.Hour),
			Domain:            "garmin.com",
			Email:             email,
		}, nil
	}
	t.Cleanup(func() { loginBrowserFn = origBrowser })

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &AuthLoginCmd{Email: "saved@example.com"}
	if err := cmd.Run(g); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify default account was saved to config.
	cfg, err := config.Read()
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if cfg.DefaultAccount != "saved@example.com" {
		t.Errorf("DefaultAccount = %q, want saved@example.com", cfg.DefaultAccount)
	}
}

func TestAuthLogin_Headless(t *testing.T) {
	testutil.TempConfigDir(t)
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)

	origHeadless := loginHeadlessFn
	loginHeadlessFn = func(_ context.Context, email, password string, _ garminauth.LoginOptions) (*garminauth.Tokens, error) {
		if password != "test-password" {
			return nil, fmt.Errorf("wrong password")
		}
		return &garminauth.Tokens{
			OAuth2AccessToken: "headless-access-token",
			OAuth2ExpiresAt:   time.Now().Add(time.Hour),
			Domain:            "garmin.com",
			Email:             email,
		}, nil
	}
	t.Cleanup(func() { loginHeadlessFn = origHeadless })

	origPw := readPasswordFn
	readPasswordFn = func(_ int) ([]byte, error) {
		return []byte("test-password"), nil
	}
	t.Cleanup(func() { readPasswordFn = origPw })

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &AuthLoginCmd{Email: "test@example.com", Headless: true}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "Logged in") {
		t.Fatalf("expected 'Logged in' message, got: %q", buf.String())
	}

	// Verify tokens were stored.
	data, getErr := store.Get("test@example.com")
	if getErr != nil {
		t.Fatalf("expected stored tokens, got: %v", getErr)
	}
	stored, unmarshalErr := garminauth.UnmarshalTokens(data)
	if unmarshalErr != nil {
		t.Fatalf("unmarshal stored tokens: %v", unmarshalErr)
	}
	if stored.OAuth2AccessToken != "headless-access-token" {
		t.Fatalf("expected headless-access-token, got %q", stored.OAuth2AccessToken)
	}
}

func TestAuthLogin_HeadlessBadPassword(t *testing.T) {
	testutil.TempConfigDir(t)
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)

	origHeadless := loginHeadlessFn
	loginHeadlessFn = func(_ context.Context, _, _ string, _ garminauth.LoginOptions) (*garminauth.Tokens, error) {
		return nil, fmt.Errorf("authentication failed")
	}
	t.Cleanup(func() { loginHeadlessFn = origHeadless })

	origPw := readPasswordFn
	readPasswordFn = func(_ int) ([]byte, error) {
		return []byte("wrong"), nil
	}
	t.Cleanup(func() { readPasswordFn = origPw })

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &AuthLoginCmd{Email: "test@example.com", Headless: true}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "login") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuthLogin_BrowserError(t *testing.T) {
	testutil.TempConfigDir(t)
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)

	origBrowser := loginBrowserFn
	loginBrowserFn = func(_ context.Context, _ string, _ garminauth.LoginOptions) (*garminauth.Tokens, error) {
		return nil, fmt.Errorf("browser login failed")
	}
	t.Cleanup(func() { loginBrowserFn = origBrowser })

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &AuthLoginCmd{Email: "test@example.com"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "login") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecute_AuthLoginUnknownDomain(t *testing.T) {
	tests := []struct {
		name   string
		env    string
		config string
	}{
		{name: "env", env: "foo.example"},
		{name: "config", config: "foo.example"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.TempConfigDir(t)
			t.Setenv(config.EnvDomain, tt.env)
			if tt.config != "" {
				if err := config.Update(func(f *config.File) { f.DomainName = tt.config }); err != nil {
					t.Fatal(err)
				}
			}

			origHeadless := loginHeadlessFn
			loginHeadlessFn = func(context.Context, string, string, garminauth.LoginOptions) (*garminauth.Tokens, error) {
				t.Error("login ran for an unknown domain")
				return nil, errors.New("unexpected login")
			}
			t.Cleanup(func() { loginHeadlessFn = origHeadless })
			origPw := readPasswordFn
			readPasswordFn = func(int) ([]byte, error) {
				t.Error("password prompt ran for an unknown domain")
				return nil, errors.New("unexpected prompt")
			}
			t.Cleanup(func() { readPasswordFn = origPw })

			var stdout, stderr bytes.Buffer
			code := execute([]string{"auth", "login", "x@example.com", "--headless"}, "test", "", "", &stdout, &stderr)
			if code != 2 {
				t.Errorf("exit code = %d, want 2", code)
			}
			if !strings.Contains(stderr.String(), "foo.example") {
				t.Errorf("stderr does not name the domain: %q", stderr.String())
			}
			if strings.Contains(stderr.String(), "Password:") {
				t.Errorf("stderr has a password prompt: %q", stderr.String())
			}
		})
	}
}
