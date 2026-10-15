package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
)

// Environment variable names for configuration overrides.
const (
	EnvAccount        = "GCCLI_ACCOUNT"
	EnvDomain         = "GCCLI_DOMAIN"
	EnvColor          = "GCCLI_COLOR"
	EnvJSON           = "GCCLI_JSON"
	EnvPlain          = "GCCLI_PLAIN"
	EnvKeyringBackend = "GCCLI_KEYRING_BACKEND"
)

// File represents the on-disk configuration file.
type File struct {
	DefaultAccount  string              `json:"default_account,omitempty"`
	KeyringBackend  string              `json:"keyring_backend,omitempty"`
	DomainName      string              `json:"domain,omitempty"`
	DefaultFormat   string              `json:"default_format,omitempty"`
	ActivitySummary map[string][]string `json:"activity_summary,omitempty"`
	Units           string              `json:"units,omitempty"`
}

// Read loads the configuration from the default config file path.
// If the file does not exist, it returns a zero-value File (no error).
func Read() (*File, error) {
	p, err := ConfigFilePath()
	if err != nil {
		return nil, err
	}
	return ReadFrom(p)
}

// ReadFrom loads configuration from the given file path.
// If the file does not exist, it returns a zero-value File (no error).
func ReadFrom(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &File{}, nil
		}
		return nil, err
	}
	var f File
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	return &f, nil
}

// Update applies fn to the config file under an exclusive lock and writes
// back the keys that fn sets. All other keys, also keys that File does not
// know, keep their values. The new file replaces the old one by rename. If
// the file is not valid JSON, Update returns an error and does not write.
func Update(fn func(*File)) error {
	path, err := ConfigFilePath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	unlock, err := LockFile(filepath.Join(dir, "config.lock"), nil)
	if err != nil {
		return err
	}
	defer unlock()

	if target, err := filepath.EvalSymlinks(path); err == nil {
		path = target
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		data = []byte("{}")
	} else if err != nil {
		return err
	}

	var keys map[string]json.RawMessage
	var f File
	if err := json.Unmarshal(data, &keys); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	if err := json.Unmarshal(data, &f); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	if keys == nil {
		keys = map[string]json.RawMessage{}
	}

	before, err := knownKeys(&f)
	if err != nil {
		return err
	}
	fn(&f)
	after, err := knownKeys(&f)
	if err != nil {
		return err
	}
	maps.Copy(keys, after)
	for k := range before {
		if _, ok := after[k]; !ok {
			delete(keys, k)
		}
	}

	out, err := json.MarshalIndent(keys, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(path, append(out, '\n'))
}

func knownKeys(f *File) (map[string]json.RawMessage, error) {
	data, err := json.Marshal(f)
	if err != nil {
		return nil, err
	}
	var keys map[string]json.RawMessage
	err = json.Unmarshal(data, &keys)
	return keys, err
}

func writeFileAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// Domain returns the effective domain, checking the environment variable first,
// then the config file, then falling back to the default.
func (f *File) Domain() string {
	if v := os.Getenv(EnvDomain); v != "" {
		return v
	}
	if f.DomainName != "" {
		return f.DomainName
	}
	return "garmin.com"
}

// KeyringBackendValue returns the effective keyring backend, checking the
// environment variable first, then the config file.
func (f *File) KeyringBackendValue() string {
	if v := os.Getenv(EnvKeyringBackend); v != "" {
		return v
	}
	return f.KeyringBackend
}

// Account returns the effective account email, checking the environment variable
// first, then the config file's default account.
func (f *File) Account() string {
	if v := os.Getenv(EnvAccount); v != "" {
		return v
	}
	return f.DefaultAccount
}

// IsJSON returns true if the GCCLI_JSON environment variable is set to a truthy value.
func IsJSON() bool {
	return isTruthy(os.Getenv(EnvJSON))
}

// IsPlain returns true if the GCCLI_PLAIN environment variable is set to a truthy value.
func IsPlain() bool {
	return isTruthy(os.Getenv(EnvPlain))
}

// ColorMode returns the color mode from the GCCLI_COLOR environment variable.
// Returns empty string if not set.
func ColorMode() string {
	return os.Getenv(EnvColor)
}

func isTruthy(v string) bool {
	switch v {
	case "1", "true", "yes":
		return true
	}
	return false
}
