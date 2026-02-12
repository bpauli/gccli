package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigDir(t *testing.T) {
	dir, err := ConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(dir) {
		t.Errorf("ConfigDir() returned relative path: %s", dir)
	}
	if filepath.Base(dir) != appName {
		t.Errorf("ConfigDir() = %s, want base dir %q", dir, appName)
	}
}

func TestConfigFilePath(t *testing.T) {
	p, err := ConfigFilePath()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(p) != "config.json" {
		t.Errorf("ConfigFilePath() = %s, want basename config.json", p)
	}
}

func TestCredentialsDir(t *testing.T) {
	dir, err := CredentialsDir()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(dir) != "credentials" {
		t.Errorf("CredentialsDir() = %s, want base dir credentials", dir)
	}
}

func TestReadFrom_Missing(t *testing.T) {
	f, err := ReadFrom(filepath.Join(t.TempDir(), "nonexistent.json"))
	if err != nil {
		t.Fatal(err)
	}
	if f.KeyringBackend != "" || f.DomainName != "" || f.DefaultFormat != "" {
		t.Errorf("ReadFrom missing file should return zero-value File, got %+v", f)
	}
}

func TestWriteToAndReadFrom(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "config.json")

	want := &File{
		KeyringBackend: "file",
		DomainName:     "garmin.cn",
		DefaultFormat:  "json",
	}

	if err := WriteTo(want, path); err != nil {
		t.Fatal(err)
	}

	got, err := ReadFrom(path)
	if err != nil {
		t.Fatal(err)
	}

	if got.KeyringBackend != want.KeyringBackend {
		t.Errorf("KeyringBackend = %q, want %q", got.KeyringBackend, want.KeyringBackend)
	}
	if got.DomainName != want.DomainName {
		t.Errorf("DomainName = %q, want %q", got.DomainName, want.DomainName)
	}
	if got.DefaultFormat != want.DefaultFormat {
		t.Errorf("DefaultFormat = %q, want %q", got.DefaultFormat, want.DefaultFormat)
	}
}

func TestWriteTo_FilePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	if err := WriteTo(&File{}, path); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	perm := info.Mode().Perm()
	if perm != 0o600 {
		t.Errorf("file perm = %o, want 0600", perm)
	}
}

func TestReadFrom_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("{invalid"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := ReadFrom(path)
	if err == nil {
		t.Error("ReadFrom invalid JSON should return error")
	}
}

func TestFile_Domain(t *testing.T) {
	// Default when nothing is set.
	f := &File{}
	if got := f.Domain(); got != "garmin.com" {
		t.Errorf("Domain() = %q, want garmin.com", got)
	}

	// Config file value.
	f.DomainName = "garmin.cn"
	if got := f.Domain(); got != "garmin.cn" {
		t.Errorf("Domain() = %q, want garmin.cn", got)
	}

	// Env var overrides config.
	t.Setenv(EnvDomain, "garmin.com")
	if got := f.Domain(); got != "garmin.com" {
		t.Errorf("Domain() = %q, want garmin.com (env override)", got)
	}
}

func TestFile_KeyringBackendValue(t *testing.T) {
	f := &File{KeyringBackend: "keychain"}
	if got := f.KeyringBackendValue(); got != "keychain" {
		t.Errorf("KeyringBackendValue() = %q, want keychain", got)
	}

	t.Setenv(EnvKeyringBackend, "file")
	if got := f.KeyringBackendValue(); got != "file" {
		t.Errorf("KeyringBackendValue() = %q, want file (env override)", got)
	}
}

func TestFile_Account(t *testing.T) {
	// Empty when nothing is set.
	t.Setenv(EnvAccount, "")
	f := &File{}
	if got := f.Account(); got != "" {
		t.Errorf("Account() = %q, want empty", got)
	}

	// Config file value.
	f.DefaultAccount = "config@example.com"
	if got := f.Account(); got != "config@example.com" {
		t.Errorf("Account() = %q, want config@example.com", got)
	}

	// Env var overrides config.
	t.Setenv(EnvAccount, "env@example.com")
	if got := f.Account(); got != "env@example.com" {
		t.Errorf("Account() = %q, want env@example.com (env override)", got)
	}
}

func TestWriteToAndReadFrom_DefaultAccount(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	want := &File{
		DefaultAccount: "user@example.com",
		DomainName:     "garmin.com",
	}
	if err := WriteTo(want, path); err != nil {
		t.Fatal(err)
	}
	got, err := ReadFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.DefaultAccount != want.DefaultAccount {
		t.Errorf("DefaultAccount = %q, want %q", got.DefaultAccount, want.DefaultAccount)
	}
}

func TestIsJSON(t *testing.T) {
	t.Setenv(EnvJSON, "")
	if IsJSON() {
		t.Error("IsJSON() should be false when unset")
	}

	t.Setenv(EnvJSON, "1")
	if !IsJSON() {
		t.Error("IsJSON() should be true when set to 1")
	}

	t.Setenv(EnvJSON, "true")
	if !IsJSON() {
		t.Error("IsJSON() should be true when set to true")
	}

	t.Setenv(EnvJSON, "no")
	if IsJSON() {
		t.Error("IsJSON() should be false for non-truthy value")
	}
}

func TestIsPlain(t *testing.T) {
	t.Setenv(EnvPlain, "yes")
	if !IsPlain() {
		t.Error("IsPlain() should be true when set to yes")
	}

	t.Setenv(EnvPlain, "0")
	if IsPlain() {
		t.Error("IsPlain() should be false for non-truthy value")
	}
}

func TestColorMode(t *testing.T) {
	t.Setenv(EnvColor, "never")
	if got := ColorMode(); got != "never" {
		t.Errorf("ColorMode() = %q, want never", got)
	}

	t.Setenv(EnvColor, "")
	if got := ColorMode(); got != "" {
		t.Errorf("ColorMode() = %q, want empty", got)
	}
}

func TestRead_ReturnsFileOrZeroValue(t *testing.T) {
	// Read() uses the real config path. It should either find a config
	// file and parse it, or return a zero-value File if none exists.
	f, err := Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if f == nil {
		t.Fatal("Read returned nil")
	}
}

func TestWrite_AndRead_RoundTrip(t *testing.T) {
	// Set HOME to a temp dir so Write() doesn't pollute the real config.
	t.Setenv("HOME", t.TempDir())

	want := &File{
		KeyringBackend: "file",
		DomainName:     "garmin.cn",
		DefaultFormat:  "json",
	}

	if err := Write(want); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got, err := Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	if got.KeyringBackend != want.KeyringBackend {
		t.Errorf("KeyringBackend = %q, want %q", got.KeyringBackend, want.KeyringBackend)
	}
	if got.DomainName != want.DomainName {
		t.Errorf("DomainName = %q, want %q", got.DomainName, want.DomainName)
	}
	if got.DefaultFormat != want.DefaultFormat {
		t.Errorf("DefaultFormat = %q, want %q", got.DefaultFormat, want.DefaultFormat)
	}
}

func TestReadFrom_ActivitySummary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	data := []byte(`{
		"activity_summary": {
			"cycling": ["distance", "duration", "avg_power"],
			"running": ["distance", "avg_pace"]
		}
	}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := ReadFrom(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(got.ActivitySummary) != 2 {
		t.Fatalf("ActivitySummary has %d entries, want 2", len(got.ActivitySummary))
	}
	cycling := got.ActivitySummary["cycling"]
	if len(cycling) != 3 || cycling[0] != "distance" || cycling[1] != "duration" || cycling[2] != "avg_power" {
		t.Errorf("cycling fields = %v, want [distance duration avg_power]", cycling)
	}
	running := got.ActivitySummary["running"]
	if len(running) != 2 || running[0] != "distance" || running[1] != "avg_pace" {
		t.Errorf("running fields = %v, want [distance avg_pace]", running)
	}
}

func TestWriteToAndReadFrom_ActivitySummary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	want := &File{
		ActivitySummary: map[string][]string{
			"cycling": {"distance", "duration"},
		},
	}
	if err := WriteTo(want, path); err != nil {
		t.Fatal(err)
	}
	got, err := ReadFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.ActivitySummary["cycling"]) != 2 {
		t.Errorf("ActivitySummary cycling = %v, want [distance duration]", got.ActivitySummary["cycling"])
	}
}

func TestReadFrom_ActivitySummaryOmittedWhenEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := WriteTo(&File{DomainName: "garmin.com"}, path); err != nil {
		t.Fatal(err)
	}
	got, err := ReadFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.ActivitySummary != nil {
		t.Errorf("ActivitySummary should be nil when not set, got %v", got.ActivitySummary)
	}
}

func TestReadFrom_PermissionError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "unreadable.json")

	if err := os.WriteFile(path, []byte(`{}`), 0o000); err != nil {
		t.Fatal(err)
	}

	_, err := ReadFrom(path)
	if err == nil {
		t.Error("ReadFrom unreadable file should return error")
	}
}

func TestWriteTo_MarshalError(t *testing.T) {
	// WriteTo calls json.MarshalIndent which cannot fail for File struct.
	// But we can verify it handles directory creation.
	dir := t.TempDir()
	path := filepath.Join(dir, "deep", "nested", "config.json")

	if err := WriteTo(&File{DomainName: "garmin.com"}, path); err != nil {
		t.Fatalf("WriteTo nested: %v", err)
	}

	got, err := ReadFrom(path)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}
	if got.DomainName != "garmin.com" {
		t.Errorf("DomainName = %q, want garmin.com", got.DomainName)
	}
}
