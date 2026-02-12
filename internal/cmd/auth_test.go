package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/99designs/keyring"

	"github.com/bpauli/gccli/internal/garminauth"
	"github.com/bpauli/gccli/internal/outfmt"
	"github.com/bpauli/gccli/internal/secrets"
	"github.com/bpauli/gccli/internal/ui"
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
	return &Globals{Context: ctx, UI: u, Account: account}
}

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
		DisplayName:        "Test User",
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
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "No credentials stored") {
		t.Fatalf("expected 'No credentials stored' warning, got: %q", buf.String())
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

func TestAuthToken_Success(t *testing.T) {
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)

	tokens := testTokens()
	storeTestTokens(t, store, "test@example.com", tokens)

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &AuthTokenCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	// Token is printed to stdout (not captured), but no error means success.
}

func TestAuthToken_Expired(t *testing.T) {
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)

	tokens := testTokens()
	tokens.OAuth2ExpiresAt = time.Now().Add(-time.Hour)
	storeTestTokens(t, store, "test@example.com", tokens)

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &AuthTokenCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "expired") {
		t.Fatalf("expected expired warning, got: %q", buf.String())
	}
}

// --- AuthLoginCmd tests ---

func TestAuthLogin_Browser(t *testing.T) {
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

func TestAuthLogin_Headless(t *testing.T) {
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
