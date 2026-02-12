package secrets

import (
	"fmt"
	"runtime"

	"github.com/99designs/keyring"

	"github.com/bpauli/gccli/internal/config"
)

const (
	// keyPrefix is the prefix for all keyring keys.
	keyPrefix = "gccli:token:"

	// serviceName is the keyring service name.
	serviceName = "gccli"
)

// Store wraps a keyring for reading and writing token data.
type Store struct {
	ring keyring.Keyring
}

// openKeyringFn is the function used to open a keyring.
// Variable for dependency injection in tests.
var openKeyringFn = keyring.Open

// Open opens a keyring store with the given backend name.
// Supported backends: "keychain" (macOS), "secret-service" (Linux D-Bus),
// "file" (encrypted file fallback), or "" for auto-detection.
// On Linux, if D-Bus is unavailable, it falls back to the file backend.
func Open(backend string) (*Store, error) {
	credDir, err := config.CredentialsDir()
	if err != nil {
		return nil, fmt.Errorf("credentials dir: %w", err)
	}

	cfg := keyring.Config{
		ServiceName:                    serviceName,
		KeychainTrustApplication:       true,
		KeychainSynchronizable:         false,
		KeychainAccessibleWhenUnlocked: true,
		FileDir:                        credDir,
		FilePasswordFunc:               keyring.FixedStringPrompt("gccli-file-store"),
	}

	backends := resolveBackends(backend)
	if len(backends) > 0 {
		cfg.AllowedBackends = backends
	}

	ring, err := openKeyringFn(cfg)
	if err != nil {
		// On Linux, fall back to file backend if D-Bus/secret-service fails.
		if runtime.GOOS == "linux" && backend == "" {
			cfg.AllowedBackends = []keyring.BackendType{keyring.FileBackend}
			ring, err = openKeyringFn(cfg)
			if err != nil {
				return nil, fmt.Errorf("open keyring (file fallback): %w", err)
			}
			return &Store{ring: ring}, nil
		}
		return nil, fmt.Errorf("open keyring: %w", err)
	}

	return &Store{ring: ring}, nil
}

// OpenWithKeyring creates a Store from an existing keyring.Keyring.
// This is useful for testing.
func OpenWithKeyring(ring keyring.Keyring) *Store {
	return &Store{ring: ring}
}

// Get retrieves token data for the given email.
func (s *Store) Get(email string) ([]byte, error) {
	item, err := s.ring.Get(keyFor(email))
	if err != nil {
		if err == keyring.ErrKeyNotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("keyring get: %w", err)
	}
	return item.Data, nil
}

// Set stores token data for the given email.
func (s *Store) Set(email string, data []byte) error {
	err := s.ring.Set(keyring.Item{
		Key:  keyFor(email),
		Data: data,
	})
	if err != nil {
		return fmt.Errorf("keyring set: %w", err)
	}
	return nil
}

// Delete removes token data for the given email.
func (s *Store) Delete(email string) error {
	err := s.ring.Remove(keyFor(email))
	if err != nil {
		if err == keyring.ErrKeyNotFound {
			return ErrNotFound
		}
		return fmt.Errorf("keyring remove: %w", err)
	}
	return nil
}

// keyFor returns the keyring key for the given email.
func keyFor(email string) string {
	return keyPrefix + email
}

// resolveBackends maps a backend name string to keyring backend types.
func resolveBackends(name string) []keyring.BackendType {
	switch name {
	case "keychain":
		return []keyring.BackendType{keyring.KeychainBackend}
	case "secret-service":
		return []keyring.BackendType{keyring.SecretServiceBackend}
	case "file":
		return []keyring.BackendType{keyring.FileBackend}
	default:
		return nil
	}
}
