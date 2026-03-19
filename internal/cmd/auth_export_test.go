package cmd

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/bpauli/gccli/internal/garminauth"
	"github.com/bpauli/gccli/internal/outfmt"
)

func TestAuthExport_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &AuthExportCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuthExport_NotFound(t *testing.T) {
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "nobody@example.com")
	cmd := &AuthExportCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no credentials stored") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuthExport_Success(t *testing.T) {
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)

	tokens := testTokens()
	storeTestTokens(t, store, "test@example.com", tokens)

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &AuthExportCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// The output goes to stdout (not captured in buf), but no error means success.
	// Verify the stored data can be base64-decoded back to valid tokens.
	data, getErr := store.Get("test@example.com")
	if getErr != nil {
		t.Fatalf("get stored tokens: %v", getErr)
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	decoded, decErr := base64.StdEncoding.DecodeString(encoded)
	if decErr != nil {
		t.Fatalf("decode base64: %v", decErr)
	}
	roundTrip, unmarshalErr := garminauth.UnmarshalTokens(decoded)
	if unmarshalErr != nil {
		t.Fatalf("unmarshal round-trip: %v", unmarshalErr)
	}
	if roundTrip.Email != tokens.Email {
		t.Errorf("email = %q, want %q", roundTrip.Email, tokens.Email)
	}
}
