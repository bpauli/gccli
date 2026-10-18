package secrets

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/99designs/keyring"

	"github.com/bpauli/gccli/internal/config"
)

const (
	// keyPrefix is the prefix for all keyring keys.
	keyPrefix = "gccli:token:"

	// serviceName is the keyring service name.
	serviceName = "gccli"

	tempDirPrefix = ".set-"
	tempLockName  = ".set.lock"
)

// Store wraps a keyring for reading and writing token data.
type Store struct {
	ring        keyring.Keyring
	fileBackend *keyring.Config
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
	if backends == nil {
		backends = keyring.AvailableBackends()
	}
	for _, b := range backends {
		cfg.AllowedBackends = []keyring.BackendType{b}
		var ring keyring.Keyring
		if ring, err = openKeyringFn(cfg); err == nil {
			return newStore(ring, cfg), nil
		}
	}
	return nil, fmt.Errorf("open keyring: %w", err)
}

func newStore(ring keyring.Keyring, cfg keyring.Config) *Store {
	s := &Store{ring: ring}
	if cfg.AllowedBackends[0] == keyring.FileBackend {
		s.fileBackend = &cfg
	}
	return s
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
	item := keyring.Item{Key: keyFor(email), Data: data}
	var err error
	if s.fileBackend != nil {
		err = s.replaceFile(item)
	} else {
		err = s.ring.Set(item)
	}
	if err != nil {
		return fmt.Errorf("keyring set: %w", err)
	}
	return nil
}

// The file keyring truncates the file before it writes, so a concurrent
// reader could see an empty or cut file.
func (s *Store) replaceFile(item keyring.Item) error {
	dir := s.fileBackend.FileDir
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	lock, err := openTempLock(dir)
	if err != nil {
		return err
	}
	defer func() { _ = lock.Close() }()
	if err := lockShared(lock); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(dir, tempDirPrefix)
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tmp) }()

	cfg := *s.fileBackend
	cfg.FileDir = tmp
	ring, err := keyring.Open(cfg)
	if err != nil {
		return err
	}
	if err := ring.Set(item); err != nil {
		return err
	}
	written, err := os.ReadDir(tmp)
	if err != nil {
		return err
	}
	if len(written) != 1 {
		return fmt.Errorf("file keyring wrote %d files, want 1", len(written))
	}
	name := written[0].Name()
	return os.Rename(filepath.Join(tmp, name), filepath.Join(dir, name))
}

// openTempLock opens the lock that every write holds shared while its temp
// dir exists. When it gets the lock exclusively, no write is in flight, so
// every temp dir left in dir belongs to a write that was killed.
func openTempLock(dir string) (*os.File, error) {
	lock, err := os.OpenFile(filepath.Join(dir, tempLockName), os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}
	if tryLockExclusive(lock) {
		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			if e.IsDir() && strings.HasPrefix(e.Name(), tempDirPrefix) {
				_ = os.RemoveAll(filepath.Join(dir, e.Name()))
			}
		}
	}
	return lock, nil
}

// Delete removes token data for the given email.
func (s *Store) Delete(email string) error {
	if s.fileBackend != nil {
		if lock, err := openTempLock(s.fileBackend.FileDir); err == nil {
			_ = lock.Close()
		}
	}
	err := s.ring.Remove(keyFor(email))
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) || errors.Is(err, fs.ErrNotExist) {
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
