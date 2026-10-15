package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/99designs/keyring"

	"github.com/bpauli/gccli/internal/config"
	"github.com/bpauli/gccli/internal/garminapi"
	"github.com/bpauli/gccli/internal/garminauth"
	"github.com/bpauli/gccli/internal/outfmt"
	"github.com/bpauli/gccli/internal/secrets"
	"github.com/bpauli/gccli/internal/testutil"
)

func TestResolveClient_ProactiveRefresh(t *testing.T) {
	testutil.TempConfigDir(t)
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)

	expired := testTokens()
	expired.OAuth2ExpiresAt = time.Now().Add(-time.Hour)
	storeTestTokens(t, store, "test@example.com", expired)

	// Override proactive refresh to return fresh tokens.
	origRefresh := refreshFn
	t.Cleanup(func() { refreshFn = origRefresh })
	refreshFn = func(_ context.Context, tokens *garminauth.Tokens, _ garminauth.LoginOptions) (*garminauth.Tokens, error) {
		newTokens := *tokens
		newTokens.OAuth2AccessToken = "proactively-refreshed"
		newTokens.OAuth2ExpiresAt = time.Now().Add(time.Hour)
		return &newTokens, nil
	}

	// Override newClientFn to capture the tokens it receives.
	origClient := newClientFn
	t.Cleanup(func() { newClientFn = origClient })
	var receivedTokens *garminauth.Tokens
	newClientFn = func(tokens *garminauth.Tokens) *garminapi.Client {
		receivedTokens = tokens
		return garminapi.NewClient(tokens)
	}

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	_, err := resolveClient(g)
	if err != nil {
		t.Fatalf("resolveClient: %v", err)
	}

	if receivedTokens == nil {
		t.Fatal("newClientFn was not called")
	}
	if receivedTokens.OAuth2AccessToken != "proactively-refreshed" {
		t.Errorf("token = %q, want proactively-refreshed", receivedTokens.OAuth2AccessToken)
	}

	// Verify refreshed tokens were persisted to keyring.
	data, err := store.Get("test@example.com")
	if err != nil {
		t.Fatalf("store.Get: %v", err)
	}
	persisted, err := garminauth.UnmarshalTokens(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if persisted.OAuth2AccessToken != "proactively-refreshed" {
		t.Errorf("persisted token = %q, want proactively-refreshed", persisted.OAuth2AccessToken)
	}
}

func TestResolveClient_ValidTokens_NoRefresh(t *testing.T) {
	dir := testutil.TempConfigDir(t)
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)

	tokens := testTokens() // Not expired.
	storeTestTokens(t, store, "test@example.com", tokens)

	// Override proactive refresh — should NOT be called.
	origRefresh := refreshFn
	t.Cleanup(func() { refreshFn = origRefresh })
	refreshCalled := false
	refreshFn = func(_ context.Context, _ *garminauth.Tokens, _ garminauth.LoginOptions) (*garminauth.Tokens, error) {
		refreshCalled = true
		return nil, nil
	}

	origClient := newClientFn
	t.Cleanup(func() { newClientFn = origClient })
	var receivedTokens *garminauth.Tokens
	newClientFn = func(tokens *garminauth.Tokens) *garminapi.Client {
		receivedTokens = tokens
		return garminapi.NewClient(tokens)
	}

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	_, err := resolveClient(g)
	if err != nil {
		t.Fatalf("resolveClient: %v", err)
	}

	if refreshCalled {
		t.Error("refresh was called for valid tokens")
	}
	if receivedTokens.OAuth2AccessToken != "test-access-token" {
		t.Errorf("token = %q, want test-access-token", receivedTokens.OAuth2AccessToken)
	}
	if _, err := os.Stat(filepath.Join(dir, "token.lock")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("token.lock stat err = %v, want not exist: a valid token takes no lock", err)
	}
}

func TestResolveClient_ProactiveRefreshFails_UsesOriginalTokens(t *testing.T) {
	testutil.TempConfigDir(t)
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)

	expired := testTokens()
	expired.OAuth2ExpiresAt = time.Now().Add(-time.Hour)
	storeTestTokens(t, store, "test@example.com", expired)

	// Override proactive refresh to fail.
	origRefresh := refreshFn
	t.Cleanup(func() { refreshFn = origRefresh })
	refreshFn = func(_ context.Context, _ *garminauth.Tokens, _ garminauth.LoginOptions) (*garminauth.Tokens, error) {
		return nil, fmt.Errorf("refresh unavailable")
	}

	origClient := newClientFn
	t.Cleanup(func() { newClientFn = origClient })
	var receivedTokens *garminauth.Tokens
	newClientFn = func(tokens *garminauth.Tokens) *garminapi.Client {
		receivedTokens = tokens
		return garminapi.NewClient(tokens)
	}

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	_, err := resolveClient(g)
	if err != nil {
		t.Fatalf("resolveClient: %v", err)
	}

	// Should proceed with the original expired tokens — the 401 retry path can still refresh.
	if receivedTokens.OAuth2AccessToken != "test-access-token" {
		t.Errorf("token = %q, want test-access-token (original)", receivedTokens.OAuth2AccessToken)
	}
}

func TestResolveClient_RefresherRefreshesAndSaves(t *testing.T) {
	testutil.TempConfigDir(t)
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	storeTestTokens(t, store, "test@example.com", testTokens())

	origRefresh := refreshFn
	t.Cleanup(func() { refreshFn = origRefresh })
	refreshFn = func(_ context.Context, tokens *garminauth.Tokens, _ garminauth.LoginOptions) (*garminauth.Tokens, error) {
		fresh := *tokens
		fresh.OAuth2AccessToken = "401-refreshed-token"
		return &fresh, nil
	}

	var buf bytes.Buffer
	client, err := resolveClient(testGlobals(t, &buf, outfmt.Table, "test@example.com"))
	if err != nil {
		t.Fatalf("resolveClient: %v", err)
	}
	got, err := client.Refresher(context.Background(), client.Tokens())
	if err != nil {
		t.Fatalf("Refresher: %v", err)
	}
	if got.OAuth2AccessToken != "401-refreshed-token" {
		t.Errorf("token = %q, want 401-refreshed-token", got.OAuth2AccessToken)
	}
	if persisted := storedAccessToken(t, store); persisted != "401-refreshed-token" {
		t.Errorf("persisted token = %q, want 401-refreshed-token", persisted)
	}
}

func TestResolveClient_RefresherAdoptsNewerStoredToken(t *testing.T) {
	testutil.TempConfigDir(t)
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	storeTestTokens(t, store, "test@example.com", testTokens())

	origRefresh := refreshFn
	t.Cleanup(func() { refreshFn = origRefresh })
	refreshFn = func(context.Context, *garminauth.Tokens, garminauth.LoginOptions) (*garminauth.Tokens, error) {
		t.Error("refreshFn called, want the newer stored token")
		return nil, fmt.Errorf("unexpected")
	}

	var buf bytes.Buffer
	client, err := resolveClient(testGlobals(t, &buf, outfmt.Table, "test@example.com"))
	if err != nil {
		t.Fatalf("resolveClient: %v", err)
	}
	other := testTokens()
	other.OAuth2AccessToken = "other-process-token"
	storeTestTokens(t, store, "test@example.com", other)

	got, err := client.Refresher(context.Background(), client.Tokens())
	if err != nil {
		t.Fatalf("Refresher: %v", err)
	}
	if got.OAuth2AccessToken != "other-process-token" {
		t.Errorf("token = %q, want other-process-token", got.OAuth2AccessToken)
	}
}

func TestResolveClient_RefresherRefreshesNewerExpiredStoredToken(t *testing.T) {
	testutil.TempConfigDir(t)
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	storeTestTokens(t, store, "test@example.com", testTokens())

	origRefresh := refreshFn
	t.Cleanup(func() { refreshFn = origRefresh })
	var usedRefresh string
	refreshFn = func(_ context.Context, tokens *garminauth.Tokens, _ garminauth.LoginOptions) (*garminauth.Tokens, error) {
		usedRefresh = tokens.OAuth2RefreshToken
		fresh := *tokens
		fresh.OAuth2AccessToken = "fresh"
		return &fresh, nil
	}

	var buf bytes.Buffer
	client, err := resolveClient(testGlobals(t, &buf, outfmt.Table, "test@example.com"))
	if err != nil {
		t.Fatalf("resolveClient: %v", err)
	}
	other := testTokens()
	other.OAuth2AccessToken = "other-process-token"
	other.OAuth2RefreshToken = "other-process-refresh"
	other.OAuth2ExpiresAt = time.Now().Add(-time.Minute)
	storeTestTokens(t, store, "test@example.com", other)

	if _, err := client.Refresher(context.Background(), client.Tokens()); err != nil {
		t.Fatalf("Refresher: %v", err)
	}
	if usedRefresh != "other-process-refresh" {
		t.Errorf("refresh token sent = %q, want the stored other-process-refresh", usedRefresh)
	}
}

func TestResolveClient_RefresherRejected(t *testing.T) {
	testutil.TempConfigDir(t)
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	storeTestTokens(t, store, "test@example.com", testTokens())

	origRefresh := refreshFn
	t.Cleanup(func() { refreshFn = origRefresh })
	refreshFn = func(context.Context, *garminauth.Tokens, garminauth.LoginOptions) (*garminauth.Tokens, error) {
		return nil, fmt.Errorf("HTTP 400: invalid_grant")
	}

	var buf bytes.Buffer
	client, err := resolveClient(testGlobals(t, &buf, outfmt.Table, "test@example.com"))
	if err != nil {
		t.Fatalf("resolveClient: %v", err)
	}
	_, err = client.Refresher(context.Background(), client.Tokens())
	var expired *garminapi.TokenExpiredError
	if !errors.As(err, &expired) {
		t.Fatalf("err = %T %v, want TokenExpiredError", err, err)
	}
}

func TestResolveClient_LockFails(t *testing.T) {
	dir := testutil.TempConfigDir(t)
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	expired := testTokens()
	expired.OAuth2ExpiresAt = time.Now().Add(-time.Hour)
	storeTestTokens(t, store, "test@example.com", expired)
	if err := os.Mkdir(filepath.Join(dir, "token.lock"), 0o700); err != nil {
		t.Fatal(err)
	}

	origRefresh := refreshFn
	t.Cleanup(func() { refreshFn = origRefresh })
	refreshFn = func(context.Context, *garminauth.Tokens, garminauth.LoginOptions) (*garminauth.Tokens, error) {
		t.Error("refreshFn called without the lock")
		return nil, fmt.Errorf("unexpected")
	}

	var buf bytes.Buffer
	_, err := resolveClient(testGlobals(t, &buf, outfmt.Table, "test@example.com"))
	if err == nil || !strings.Contains(err.Error(), "token.lock") {
		t.Fatalf("err = %v, want one that names token.lock", err)
	}
}

type failSetKeyring struct{ *memKeyring }

func (failSetKeyring) Set(keyring.Item) error { return fmt.Errorf("keyring is read-only") }

func TestResolveClient_SaveFails(t *testing.T) {
	testutil.TempConfigDir(t)
	ring := &memKeyring{items: map[string]keyring.Item{}}
	expired := testTokens()
	expired.OAuth2ExpiresAt = time.Now().Add(-time.Hour)
	storeTestTokens(t, secrets.OpenWithKeyring(ring), "test@example.com", expired)
	overrideLoadSecrets(t, secrets.OpenWithKeyring(failSetKeyring{ring}))

	origRefresh := refreshFn
	t.Cleanup(func() { refreshFn = origRefresh })
	refreshFn = func(_ context.Context, tokens *garminauth.Tokens, _ garminauth.LoginOptions) (*garminauth.Tokens, error) {
		fresh := *tokens
		fresh.OAuth2AccessToken = "unsaved"
		fresh.OAuth2ExpiresAt = time.Now().Add(time.Hour)
		return &fresh, nil
	}

	var buf bytes.Buffer
	_, err := resolveClient(testGlobals(t, &buf, outfmt.Table, "test@example.com"))
	if err == nil || !strings.Contains(err.Error(), "save refreshed token") {
		t.Fatalf("err = %v, want a save error", err)
	}
}

func storedAccessToken(t *testing.T, store *secrets.Store) string {
	t.Helper()
	data, err := store.Get("test@example.com")
	if err != nil {
		t.Fatalf("store.Get: %v", err)
	}
	tokens, err := garminauth.UnmarshalTokens(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return tokens.OAuth2AccessToken
}

func TestResolveClient_Domain(t *testing.T) {
	tests := []struct {
		name        string
		tokenDomain string
		env         string
		config      string
		want        string
	}{
		{name: "token wins", tokenDomain: "garmin.com", env: "garmin.cn", want: "garmin.com"},
		{name: "empty token, env", env: "garmin.cn", want: "garmin.cn"},
		{name: "empty token, config", config: "garmin.cn", want: "garmin.cn"},
		{name: "empty token, default", want: "garmin.com"},
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
			store := newTestSecretsStore(t)
			overrideLoadSecrets(t, store)
			tokens := testTokens()
			tokens.Domain = tt.tokenDomain
			storeTestTokens(t, store, "test@example.com", tokens)

			origClient := newClientFn
			t.Cleanup(func() { newClientFn = origClient })
			var got string
			newClientFn = func(tokens *garminauth.Tokens) *garminapi.Client {
				got = tokens.Domain
				return garminapi.NewClient(tokens)
			}

			var buf bytes.Buffer
			if _, err := resolveClient(testGlobals(t, &buf, outfmt.Table, "test@example.com")); err != nil {
				t.Fatalf("resolveClient: %v", err)
			}
			if got != tt.want {
				t.Errorf("client domain = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveClient_UnknownDomain(t *testing.T) {
	testutil.TempConfigDir(t)
	t.Setenv(config.EnvDomain, "foo.example")
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	tokens := testTokens()
	tokens.Domain = ""
	storeTestTokens(t, store, "test@example.com", tokens)

	var buf bytes.Buffer
	_, err := resolveClient(testGlobals(t, &buf, outfmt.Table, "test@example.com"))
	if err == nil || !strings.Contains(err.Error(), "foo.example") {
		t.Fatalf("error = %v, want one that names foo.example", err)
	}
}

type syncKeyring struct {
	mu sync.Mutex
	*memKeyring
}

func (k *syncKeyring) Get(key string) (keyring.Item, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.memKeyring.Get(key)
}

func (k *syncKeyring) Set(item keyring.Item) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.memKeyring.Set(item)
}

func TestResolveClient_ConcurrentExpiredRefreshesOnce(t *testing.T) {
	testutil.TempConfigDir(t)
	store := secrets.OpenWithKeyring(&syncKeyring{memKeyring: &memKeyring{items: map[string]keyring.Item{}}})
	overrideLoadSecrets(t, store)
	expired := testTokens()
	expired.OAuth2ExpiresAt = time.Now().Add(-time.Hour)
	storeTestTokens(t, store, "test@example.com", expired)

	origRefresh := refreshFn
	t.Cleanup(func() { refreshFn = origRefresh })
	var calls atomic.Int32
	var used sync.Map
	refreshFn = func(_ context.Context, tokens *garminauth.Tokens, _ garminauth.LoginOptions) (*garminauth.Tokens, error) {
		n := calls.Add(1)
		time.Sleep(100 * time.Millisecond)
		if _, reused := used.LoadOrStore(tokens.OAuth2RefreshToken, true); reused {
			return nil, fmt.Errorf("HTTP 400: invalid_grant")
		}
		fresh := *tokens
		fresh.OAuth2AccessToken = fmt.Sprintf("refreshed-%d", n)
		fresh.OAuth2RefreshToken = fmt.Sprintf("refresh-%d", n)
		fresh.OAuth2ExpiresAt = time.Now().Add(time.Hour)
		return &fresh, nil
	}

	got := make([]string, 5)
	var wg sync.WaitGroup
	for i := range got {
		wg.Go(func() {
			var buf bytes.Buffer
			client, err := resolveClient(testGlobals(t, &buf, outfmt.Table, "test@example.com"))
			if err != nil {
				t.Errorf("resolveClient: %v", err)
				return
			}
			got[i] = client.Tokens().OAuth2AccessToken
		})
	}
	wg.Wait()

	if n := calls.Load(); n != 1 {
		t.Errorf("refresh calls = %d, want 1", n)
	}
	for i, tok := range got {
		if tok != "refreshed-1" {
			t.Errorf("client %d token = %q, want refreshed-1", i, tok)
		}
	}
	if persisted := storedAccessToken(t, store); persisted != "refreshed-1" {
		t.Errorf("persisted token = %q, want refreshed-1", persisted)
	}
}
