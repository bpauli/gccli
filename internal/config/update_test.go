package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func tempConfigPath(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	path, err := ConfigFilePath()
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func writeConfig(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func readKeys(t *testing.T, path string) map[string]string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("config is not a JSON object: %v\n%s", err, data)
	}
	keys := make(map[string]string, len(raw))
	for k, v := range raw {
		var buf bytes.Buffer
		if err := json.Compact(&buf, v); err != nil {
			t.Fatal(err)
		}
		keys[k] = buf.String()
	}
	return keys
}

func TestUpdate_KeepsKeysTheCallerDoesNotSet(t *testing.T) {
	path := tempConfigPath(t)
	writeConfig(t, path, `{
  "future_key": {"nested": [1, 2.50, "x"]},
  "activity_summary": {},
  "units": "metric",
  "default_account": "old@example.com"
}`)

	if err := Update(func(f *File) { f.DefaultAccount = "new@example.com" }); err != nil {
		t.Fatalf("Update: %v", err)
	}

	want := map[string]string{
		"future_key":       `{"nested":[1,2.50,"x"]}`,
		"activity_summary": `{}`,
		"units":            `"metric"`,
		"default_account":  `"new@example.com"`,
	}
	got := readKeys(t, path)
	for k, v := range want {
		if got[k] != v {
			t.Errorf("key %q = %s, want %s", k, got[k], v)
		}
	}
	for k := range got {
		if _, ok := want[k]; !ok {
			t.Errorf("unexpected key %q = %s", k, got[k])
		}
	}
}

func TestUpdate_ClearedKeyIsRemoved(t *testing.T) {
	path := tempConfigPath(t)
	writeConfig(t, path, `{"default_account": "old@example.com", "future_key": true}`)

	if err := Update(func(f *File) { f.DefaultAccount = "" }); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got := readKeys(t, path)
	if _, ok := got["default_account"]; ok {
		t.Errorf("default_account still present: %v", got)
	}
	if got["future_key"] != "true" {
		t.Errorf("future_key = %q, want true", got["future_key"])
	}
}

func TestUpdate_RoundTrip(t *testing.T) {
	path := tempConfigPath(t)

	err := Update(func(f *File) {
		f.DefaultAccount = "user@example.com"
		f.KeyringBackend = "file"
		f.DomainName = "garmin.cn"
		f.DefaultFormat = "json"
		f.Units = "statute_us"
		f.ActivitySummary = map[string][]string{"cycling": {"distance", "duration"}}
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	want := File{
		DefaultAccount:  "user@example.com",
		KeyringBackend:  "file",
		DomainName:      "garmin.cn",
		DefaultFormat:   "json",
		Units:           "statute_us",
		ActivitySummary: map[string][]string{"cycling": {"distance", "duration"}},
	}
	if fmt.Sprint(*got) != fmt.Sprint(want) {
		t.Errorf("Read after Update = %+v, want %+v", *got, want)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("file perm = %o, want 0600", perm)
	}
}

func TestUpdate_ConcurrentWritersAllLand(t *testing.T) {
	path := tempConfigPath(t)
	writeConfig(t, path, `{"future_key": 1}`)

	const writers = 50
	errs := make(chan error, writers)
	var wg sync.WaitGroup
	for i := range writers {
		wg.Go(func() {
			errs <- Update(func(f *File) {
				if f.ActivitySummary == nil {
					f.ActivitySummary = map[string][]string{}
				}
				f.ActivitySummary[fmt.Sprintf("w%02d", i)] = []string{"distance"}
			})
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("Update: %v", err)
		}
	}

	got, err := Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(got.ActivitySummary) != writers {
		t.Errorf("%d of %d concurrent writes landed", len(got.ActivitySummary), writers)
	}
	if readKeys(t, path)["future_key"] != "1" {
		t.Errorf("future_key lost under concurrent writes")
	}
}

func TestUpdate_InvalidJSONIsNotOverwritten(t *testing.T) {
	path := tempConfigPath(t)
	writeConfig(t, path, `{invalid`)

	if err := Update(func(f *File) { f.DefaultAccount = "new@example.com" }); err == nil {
		t.Error("Update on invalid JSON returned nil error")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{invalid` {
		t.Errorf("invalid config was overwritten with:\n%s", data)
	}
}

func TestUpdate_NullFileIsEmptyConfig(t *testing.T) {
	path := tempConfigPath(t)
	writeConfig(t, path, `null`)

	if err := Update(func(f *File) { f.Units = "metric" }); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if got := readKeys(t, path); len(got) != 1 || got["units"] != `"metric"` {
		t.Errorf("config = %v, want only units", got)
	}
}

func TestUpdate_CreatesMissingConfigDir(t *testing.T) {
	path := tempConfigPath(t)

	if err := Update(func(f *File) { f.DomainName = "garmin.com" }); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if got := readKeys(t, path); got["domain"] != `"garmin.com"` {
		t.Errorf("domain = %s, want \"garmin.com\"", got["domain"])
	}
}

func TestUpdate_Symlinks(t *testing.T) {
	t.Run("config dir", func(t *testing.T) {
		path := tempConfigPath(t)
		real := filepath.Join(t.TempDir(), "real")
		if err := os.MkdirAll(real, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(filepath.Dir(path)), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(real, filepath.Dir(path)); err != nil {
			t.Fatal(err)
		}
		writeConfig(t, path, `{"future_key": 1}`)

		if err := Update(func(f *File) { f.Units = "metric" }); err != nil {
			t.Fatalf("Update: %v", err)
		}

		got := readKeys(t, filepath.Join(real, "config.json"))
		if got["units"] != `"metric"` || got["future_key"] != "1" {
			t.Errorf("config in real dir = %v", got)
		}
	})

	t.Run("config file", func(t *testing.T) {
		path := tempConfigPath(t)
		target := filepath.Join(t.TempDir(), "dotfiles", "gccli.json")
		writeConfig(t, target, `{"future_key": 1}`)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, path); err != nil {
			t.Fatal(err)
		}

		if err := Update(func(f *File) { f.Units = "metric" }); err != nil {
			t.Fatalf("Update: %v", err)
		}

		info, err := os.Lstat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode()&os.ModeSymlink == 0 {
			t.Errorf("config.json was replaced by a regular file, want the symlink kept")
		}
		got := readKeys(t, target)
		if got["units"] != `"metric"` || got["future_key"] != "1" {
			t.Errorf("symlink target = %v", got)
		}
	})
}
