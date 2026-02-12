package testutil

import (
	"fmt"
	"testing"

	"github.com/99designs/keyring"

	"github.com/bpauli/gccli/internal/secrets"
)

// MemKeyring is a simple in-memory keyring implementation for testing.
type MemKeyring struct {
	Items map[string]keyring.Item
}

// NewMemKeyring creates a new empty in-memory keyring.
func NewMemKeyring() *MemKeyring {
	return &MemKeyring{Items: make(map[string]keyring.Item)}
}

func (m *MemKeyring) Get(key string) (keyring.Item, error) {
	item, ok := m.Items[key]
	if !ok {
		return keyring.Item{}, keyring.ErrKeyNotFound
	}
	return item, nil
}

func (m *MemKeyring) GetMetadata(_ string) (keyring.Metadata, error) {
	return keyring.Metadata{}, nil
}

func (m *MemKeyring) Set(item keyring.Item) error {
	m.Items[item.Key] = item
	return nil
}

func (m *MemKeyring) Remove(key string) error {
	if _, ok := m.Items[key]; !ok {
		return keyring.ErrKeyNotFound
	}
	delete(m.Items, key)
	return nil
}

func (m *MemKeyring) Keys() ([]string, error) {
	keys := make([]string, 0, len(m.Items))
	for k := range m.Items {
		keys = append(keys, k)
	}
	return keys, nil
}

// NewTestSecretsStore creates a Store backed by an in-memory keyring.
func NewTestSecretsStore(t *testing.T) *secrets.Store {
	t.Helper()
	return secrets.OpenWithKeyring(NewMemKeyring())
}

// StoreTestTokens marshals and stores test tokens in the given store.
func StoreTestTokens(t *testing.T, store *secrets.Store, email string) {
	t.Helper()
	tokens := TestTokens()
	tokens.Email = email
	data, err := tokens.Marshal()
	if err != nil {
		t.Fatalf("marshal tokens: %v", err)
	}
	if err := store.Set(email, data); err != nil {
		t.Fatalf("store tokens: %v", err)
	}
}

// ErrKeyring is a keyring that returns errors for all operations.
type ErrKeyring struct {
	Err error
}

func (e *ErrKeyring) Get(_ string) (keyring.Item, error) { return keyring.Item{}, e.Err }
func (e *ErrKeyring) GetMetadata(_ string) (keyring.Metadata, error) {
	return keyring.Metadata{}, e.Err
}
func (e *ErrKeyring) Set(_ keyring.Item) error { return e.Err }
func (e *ErrKeyring) Remove(_ string) error    { return e.Err }
func (e *ErrKeyring) Keys() ([]string, error)  { return nil, e.Err }

// NewErrSecretsStore creates a Store backed by an always-erroring keyring.
func NewErrSecretsStore(t *testing.T) *secrets.Store {
	t.Helper()
	return secrets.OpenWithKeyring(&ErrKeyring{Err: fmt.Errorf("keyring unavailable")})
}
