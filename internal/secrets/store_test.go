package secrets

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/99designs/keyring"
)

// newTestStore returns a Store backed by an in-memory keyring for testing.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	ring := &memKeyring{items: make(map[string]keyring.Item)}
	return OpenWithKeyring(ring)
}

func TestSetAndGet(t *testing.T) {
	s := newTestStore(t)
	email := "test@example.com"
	data := []byte(`{"token":"abc123"}`)

	if err := s.Set(email, data); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := s.Get(email)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("Get = %q, want %q", got, data)
	}
}

func TestGetNotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.Get("nobody@example.com")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get = %v, want ErrNotFound", err)
	}
}

func TestDelete(t *testing.T) {
	s := newTestStore(t)
	email := "delete@example.com"
	data := []byte(`{"token":"xyz"}`)

	if err := s.Set(email, data); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := s.Delete(email); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := s.Get(email)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get after Delete = %v, want ErrNotFound", err)
	}
}

func TestDeleteNotFound(t *testing.T) {
	s := newTestStore(t)
	err := s.Delete("nobody@example.com")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Delete = %v, want ErrNotFound", err)
	}
}

func TestOverwrite(t *testing.T) {
	s := newTestStore(t)
	email := "overwrite@example.com"

	if err := s.Set(email, []byte("first")); err != nil {
		t.Fatalf("Set first: %v", err)
	}
	if err := s.Set(email, []byte("second")); err != nil {
		t.Fatalf("Set second: %v", err)
	}

	got, err := s.Get(email)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != "second" {
		t.Errorf("Get = %q, want %q", got, "second")
	}
}

func TestMultipleEmails(t *testing.T) {
	s := newTestStore(t)
	emails := map[string][]byte{
		"a@example.com": []byte("data-a"),
		"b@example.com": []byte("data-b"),
	}

	for email, data := range emails {
		if err := s.Set(email, data); err != nil {
			t.Fatalf("Set %s: %v", email, err)
		}
	}

	for email, want := range emails {
		got, err := s.Get(email)
		if err != nil {
			t.Fatalf("Get %s: %v", email, err)
		}
		if string(got) != string(want) {
			t.Errorf("Get %s = %q, want %q", email, got, want)
		}
	}
}

func TestKeyFor(t *testing.T) {
	got := keyFor("user@garmin.com")
	want := "gccli:token:user@garmin.com"
	if got != want {
		t.Errorf("keyFor = %q, want %q", got, want)
	}
}

func TestResolveBackends(t *testing.T) {
	tests := []struct {
		name string
		want int
	}{
		{"keychain", 1},
		{"secret-service", 1},
		{"file", 1},
		{"", 0},
		{"unknown", 0},
	}
	for _, tt := range tests {
		got := resolveBackends(tt.name)
		if len(got) != tt.want {
			t.Errorf("resolveBackends(%q) returned %d backends, want %d", tt.name, len(got), tt.want)
		}
	}
}

// errKeyring is a keyring that returns errors for all operations.
type errKeyring struct {
	err error
}

func (e *errKeyring) Get(_ string) (keyring.Item, error) { return keyring.Item{}, e.err }
func (e *errKeyring) GetMetadata(_ string) (keyring.Metadata, error) {
	return keyring.Metadata{}, e.err
}
func (e *errKeyring) Set(_ keyring.Item) error { return e.err }
func (e *errKeyring) Remove(_ string) error    { return e.err }
func (e *errKeyring) Keys() ([]string, error)  { return nil, e.err }

func TestGetError(t *testing.T) {
	s := OpenWithKeyring(&errKeyring{err: fmt.Errorf("disk error")})
	_, err := s.Get("user@example.com")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "keyring get") {
		t.Errorf("error = %q, want keyring get wrapper", err)
	}
}

func TestSetError(t *testing.T) {
	s := OpenWithKeyring(&errKeyring{err: fmt.Errorf("disk error")})
	err := s.Set("user@example.com", []byte("data"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "keyring set") {
		t.Errorf("error = %q, want keyring set wrapper", err)
	}
}

func TestDeleteError(t *testing.T) {
	s := OpenWithKeyring(&errKeyring{err: fmt.Errorf("disk error")})
	err := s.Delete("user@example.com")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "keyring remove") {
		t.Errorf("error = %q, want keyring remove wrapper", err)
	}
}

func TestOpen_Success(t *testing.T) {
	orig := openKeyringFn
	t.Cleanup(func() { openKeyringFn = orig })

	openKeyringFn = func(_ keyring.Config) (keyring.Keyring, error) {
		return &memKeyring{items: make(map[string]keyring.Item)}, nil
	}

	store, err := Open("")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if store == nil {
		t.Fatal("Open returned nil store")
	}

	// Verify the store is functional.
	if err := store.Set("test@example.com", []byte("data")); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := store.Get("test@example.com")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != "data" {
		t.Errorf("Get = %q, want %q", got, "data")
	}
}

func TestOpen_WithBackend(t *testing.T) {
	orig := openKeyringFn
	t.Cleanup(func() { openKeyringFn = orig })

	var capturedCfg keyring.Config
	openKeyringFn = func(cfg keyring.Config) (keyring.Keyring, error) {
		capturedCfg = cfg
		return &memKeyring{items: make(map[string]keyring.Item)}, nil
	}

	_, err := Open("file")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if len(capturedCfg.AllowedBackends) != 1 || capturedCfg.AllowedBackends[0] != keyring.FileBackend {
		t.Errorf("AllowedBackends = %v, want [FileBackend]", capturedCfg.AllowedBackends)
	}
}

func TestOpen_Error(t *testing.T) {
	orig := openKeyringFn
	t.Cleanup(func() { openKeyringFn = orig })

	openKeyringFn = func(_ keyring.Config) (keyring.Keyring, error) {
		return nil, fmt.Errorf("keyring unavailable")
	}

	_, err := Open("")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "open keyring") {
		t.Errorf("error = %q, want 'open keyring' wrapper", err)
	}
}

func TestOpen_ConfigFields(t *testing.T) {
	orig := openKeyringFn
	t.Cleanup(func() { openKeyringFn = orig })

	var capturedCfg keyring.Config
	openKeyringFn = func(cfg keyring.Config) (keyring.Keyring, error) {
		capturedCfg = cfg
		return &memKeyring{items: make(map[string]keyring.Item)}, nil
	}

	_, err := Open("")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	if capturedCfg.ServiceName != serviceName {
		t.Errorf("ServiceName = %q, want %q", capturedCfg.ServiceName, serviceName)
	}
	if !capturedCfg.KeychainTrustApplication {
		t.Error("KeychainTrustApplication = false, want true")
	}
	if capturedCfg.FileDir == "" {
		t.Error("FileDir is empty, want credentials dir")
	}
}

// memKeyring is a simple in-memory keyring implementation for testing.
type memKeyring struct {
	items map[string]keyring.Item
}

func (m *memKeyring) Get(key string) (keyring.Item, error) {
	item, ok := m.items[key]
	if !ok {
		return keyring.Item{}, keyring.ErrKeyNotFound
	}
	return item, nil
}

func (m *memKeyring) GetMetadata(_ string) (keyring.Metadata, error) {
	return keyring.Metadata{}, nil
}

func (m *memKeyring) Set(item keyring.Item) error {
	m.items[item.Key] = item
	return nil
}

func (m *memKeyring) Remove(key string) error {
	if _, ok := m.items[key]; !ok {
		return keyring.ErrKeyNotFound
	}
	delete(m.items, key)
	return nil
}

func (m *memKeyring) Keys() ([]string, error) {
	keys := make([]string, 0, len(m.items))
	for k := range m.items {
		keys = append(keys, k)
	}
	return keys, nil
}
