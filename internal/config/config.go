package config

import (
	"encoding/json"
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

// Write saves the configuration to the default config file path.
func Write(f *File) error {
	p, err := ConfigFilePath()
	if err != nil {
		return err
	}
	return WriteTo(f, p)
}

// WriteTo saves the configuration to the given file path,
// creating parent directories as needed.
func WriteTo(f *File, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
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
