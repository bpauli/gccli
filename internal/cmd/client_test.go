package cmd

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/bpauli/gccli/internal/config"
	"github.com/bpauli/gccli/internal/garminapi"
	"github.com/bpauli/gccli/internal/garminauth"
	"github.com/bpauli/gccli/internal/outfmt"
	"github.com/bpauli/gccli/internal/testutil"
)

func TestResolveClient_ProactiveRefresh(t *testing.T) {
	// Set up store with expired tokens.
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)

	expired := testTokens()
	expired.OAuth2ExpiresAt = time.Now().Add(-time.Hour)
	storeTestTokens(t, store, "test@example.com", expired)

	// Override proactive refresh to return fresh tokens.
	origRefresh := proactiveRefreshFn
	t.Cleanup(func() { proactiveRefreshFn = origRefresh })
	proactiveRefreshFn = func(_ context.Context, tokens *garminauth.Tokens, _ garminauth.LoginOptions) (*garminauth.Tokens, error) {
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
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)

	tokens := testTokens() // Not expired.
	storeTestTokens(t, store, "test@example.com", tokens)

	// Override proactive refresh — should NOT be called.
	origRefresh := proactiveRefreshFn
	t.Cleanup(func() { proactiveRefreshFn = origRefresh })
	refreshCalled := false
	proactiveRefreshFn = func(_ context.Context, _ *garminauth.Tokens, _ garminauth.LoginOptions) (*garminauth.Tokens, error) {
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
}

func TestResolveClient_ProactiveRefreshFails_UsesOriginalTokens(t *testing.T) {
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)

	expired := testTokens()
	expired.OAuth2ExpiresAt = time.Now().Add(-time.Hour)
	storeTestTokens(t, store, "test@example.com", expired)

	// Override proactive refresh to fail.
	origRefresh := proactiveRefreshFn
	t.Cleanup(func() { proactiveRefreshFn = origRefresh })
	proactiveRefreshFn = func(_ context.Context, _ *garminauth.Tokens, _ garminauth.LoginOptions) (*garminauth.Tokens, error) {
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

func TestResolveClient_OnTokenRefreshPersistsTokens(t *testing.T) {
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)

	tokens := testTokens() // Valid tokens — no proactive refresh.
	storeTestTokens(t, store, "test@example.com", tokens)

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	client, err := resolveClient(g)
	if err != nil {
		t.Fatalf("resolveClient: %v", err)
	}

	if client.OnTokenRefresh == nil {
		t.Fatal("OnTokenRefresh callback was not set")
	}

	// Simulate a 401-triggered refresh by invoking the callback directly.
	refreshed := testTokens()
	refreshed.OAuth2AccessToken = "401-refreshed-token"
	client.OnTokenRefresh(refreshed)

	// Verify the refreshed tokens were persisted to the keyring.
	data, err := store.Get("test@example.com")
	if err != nil {
		t.Fatalf("store.Get: %v", err)
	}
	persisted, err := garminauth.UnmarshalTokens(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if persisted.OAuth2AccessToken != "401-refreshed-token" {
		t.Errorf("persisted token = %q, want 401-refreshed-token", persisted.OAuth2AccessToken)
	}
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
				if err := config.Write(&config.File{DomainName: tt.config}); err != nil {
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
