package cmd

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bpauli/gccli/internal/garminauth"
	"github.com/bpauli/gccli/internal/outfmt"
	"github.com/bpauli/gccli/internal/testutil"
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
	testutil.TempConfigDir(t)
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

func TestAuthImport_KeepsOtherConfigKeys(t *testing.T) {
	dir := testutil.TempConfigDir(t)
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"future_key": {"a": 1}, "units": "metric", "default_account": "old@example.com"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)

	data, _ := testTokens().Marshal()
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	if err := (&AuthImportCmd{Token: base64.StdEncoding.EncodeToString(data)}).Run(g); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(written, &got); err != nil {
		t.Fatalf("config is not JSON: %v\n%s", err, written)
	}
	if got["default_account"] != "test@example.com" {
		t.Errorf("default_account = %v, want test@example.com", got["default_account"])
	}
	if got["units"] != "metric" {
		t.Errorf("units = %v, want metric", got["units"])
	}
	if fk, ok := got["future_key"].(map[string]any); !ok || fk["a"] != 1.0 {
		t.Errorf("future_key = %v, want {\"a\": 1}\n%s", got["future_key"], written)
	}
}

func TestAuthImport_FromStdin(t *testing.T) {
	testutil.TempConfigDir(t)
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
	testutil.TempConfigDir(t)
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

func TestAuthImport_UnknownDomain(t *testing.T) {
	testutil.TempConfigDir(t)
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)

	tokens := testTokens()
	tokens.Domain = "foo.example"
	data, _ := tokens.Marshal()

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	err := (&AuthImportCmd{Token: base64.StdEncoding.EncodeToString(data)}).Run(g)
	if err == nil || !strings.Contains(err.Error(), "foo.example") {
		t.Fatalf("error = %v, want one that names foo.example", err)
	}
	if _, getErr := store.Get(tokens.Email); getErr == nil {
		t.Error("token with an unknown domain was stored")
	}
}

func TestAuthImport_EmptyDomainStoredAsIs(t *testing.T) {
	testutil.TempConfigDir(t)
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)

	tokens := testTokens()
	tokens.Domain = ""
	data, _ := tokens.Marshal()

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	if err := (&AuthImportCmd{Token: base64.StdEncoding.EncodeToString(data)}).Run(g); err != nil {
		t.Fatalf("Run: %v", err)
	}
	stored, err := store.Get(tokens.Email)
	if err != nil {
		t.Fatalf("store.Get: %v", err)
	}
	if !bytes.Equal(stored, data) {
		t.Errorf("stored = %s, want %s", stored, data)
	}
}

func TestAuthImport_UnsupportedVersion(t *testing.T) {
	dir := testutil.TempConfigDir(t)
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)

	data := `{"version":2,"oauth2_access_token":"at","domain":"garmin.com","email":"test@example.com"}`
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	err := (&AuthImportCmd{Token: base64.StdEncoding.EncodeToString([]byte(data))}).Run(g)
	if err == nil || !strings.Contains(err.Error(), "token version 2 is not supported") {
		t.Fatalf("error = %v, want the token version error", err)
	}
	if _, getErr := store.Get("test@example.com"); getErr == nil {
		t.Error("token with version 2 was stored")
	}
	written, readErr := os.ReadFile(filepath.Join(dir, "config.json"))
	if readErr != nil && !os.IsNotExist(readErr) {
		t.Fatal(readErr)
	}
	if bytes.Contains(written, []byte("default_account")) {
		t.Errorf("config.json = %s, want no default_account", written)
	}
}

func TestAuthImport_StoresCurrentFormat(t *testing.T) {
	testutil.TempConfigDir(t)
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)

	data := `{"oauth1_token":"","oauth1_secret":"","oauth2_access_token":"at","oauth2_refresh_token":"rt",` +
		`"oauth2_expires_at":"2099-01-01T00:00:00Z","domain":"garmin.com","display_name":"Old Name",` +
		`"email":"test@example.com","future_key":{"a":1}}`
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	if err := (&AuthImportCmd{Token: base64.StdEncoding.EncodeToString([]byte(data))}).Run(g); err != nil {
		t.Fatalf("Run: %v", err)
	}

	stored, err := store.Get("test@example.com")
	if err != nil {
		t.Fatalf("store.Get: %v", err)
	}
	want := `{"version":1,"oauth1_token":"","oauth1_secret":"","oauth2_access_token":"at",` +
		`"oauth2_refresh_token":"rt","oauth2_expires_at":"2099-01-01T00:00:00Z","domain":"garmin.com",` +
		`"email":"test@example.com"}`
	if string(stored) != want {
		t.Errorf("stored =\n%s\nwant\n%s", stored, want)
	}
}
