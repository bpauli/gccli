package cmd

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/bpauli/gccli/internal/garminauth"
	"github.com/bpauli/gccli/internal/outfmt"
)

func TestAuthImport_NoInput(t *testing.T) {
	origStdin := readStdinFn
	readStdinFn = func() ([]byte, error) { return []byte(""), nil }
	t.Cleanup(func() { readStdinFn = origStdin })

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &AuthImportCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no token data provided") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuthImport_InvalidBase64(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &AuthImportCmd{Token: "not-valid-base64!!!"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid base64") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuthImport_InvalidJSON(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("not json"))
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &AuthImportCmd{Token: encoded}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid token data") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuthImport_MissingEmail(t *testing.T) {
	tokens := &garminauth.Tokens{
		OAuth2AccessToken: "at",
		Domain:            "garmin.com",
	}
	data, _ := tokens.Marshal()
	encoded := base64.StdEncoding.EncodeToString(data)

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &AuthImportCmd{Token: encoded}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "missing email") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuthImport_Success(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)

	tokens := testTokens()
	data, _ := tokens.Marshal()
	encoded := base64.StdEncoding.EncodeToString(data)

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &AuthImportCmd{Token: encoded}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "Imported credentials") {
		t.Fatalf("expected 'Imported credentials' message, got: %q", buf.String())
	}

	// Verify tokens were stored.
	stored, getErr := store.Get("test@example.com")
	if getErr != nil {
		t.Fatalf("expected stored tokens, got: %v", getErr)
	}
	roundTrip, unmarshalErr := garminauth.UnmarshalTokens(stored)
	if unmarshalErr != nil {
		t.Fatalf("unmarshal stored tokens: %v", unmarshalErr)
	}
	if roundTrip.OAuth2AccessToken != "test-access-token" {
		t.Errorf("access_token = %q, want %q", roundTrip.OAuth2AccessToken, "test-access-token")
	}
}

func TestAuthImport_FromStdin(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)

	tokens := testTokens()
	data, _ := tokens.Marshal()
	encoded := base64.StdEncoding.EncodeToString(data)

	origStdin := readStdinFn
	readStdinFn = func() ([]byte, error) { return []byte(encoded + "\n"), nil }
	t.Cleanup(func() { readStdinFn = origStdin })

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &AuthImportCmd{} // No Token arg — reads from stdin.
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "Imported credentials") {
		t.Fatalf("expected 'Imported credentials' message, got: %q", buf.String())
	}
}

func TestAuthImport_ExpiredWarning(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)

	tokens := testTokens()
	tokens.OAuth2ExpiresAt = time.Now().Add(-time.Hour)
	data, _ := tokens.Marshal()
	encoded := base64.StdEncoding.EncodeToString(data)

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &AuthImportCmd{Token: encoded}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "expired") {
		t.Fatalf("expected expired warning, got: %q", buf.String())
	}
}
