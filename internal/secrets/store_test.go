package secrets

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/99designs/keyring"

	"github.com/bpauli/gccli/internal/config"
	"github.com/bpauli/gccli/internal/testutil"
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

func newFileStore(t *testing.T) *Store {
	t.Helper()
	ring, err := keyring.Open(keyring.Config{
		ServiceName:      serviceName,
		AllowedBackends:  []keyring.BackendType{keyring.FileBackend},
		FileDir:          t.TempDir(),
		FilePasswordFunc: keyring.FixedStringPrompt("test"),
	})
	if err != nil {
		t.Fatalf("open file keyring: %v", err)
	}
	return OpenWithKeyring(ring)
}

func TestDeleteTwiceFileBackend(t *testing.T) {
	s := newFileStore(t)
	email := "twice@example.com"

	if err := s.Set(email, []byte(`{"token":"xyz"}`)); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := s.Delete(email); err != nil {
		t.Fatalf("first Delete: %v", err)
	}

	err := s.Delete(email)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("second Delete = %v, want ErrNotFound", err)
	}
}

func TestFileBackendReadDuringWrite(t *testing.T) {
	testutil.TempConfigDir(t)
	open := func() *Store {
		s, err := Open("file")
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		return s
	}
	email := "race@example.com"
	data := []byte(`{"oauth2_access_token":"` + strings.Repeat("a", 1<<16) + `"}`)
	writer := open()
	if err := writer.Set(email, data); err != nil {
		t.Fatalf("Set: %v", err)
	}

	var stop atomic.Bool
	var wg sync.WaitGroup
	errs := make(chan error, 4)
	for range 4 {
		reader := open()
		wg.Go(func() {
			for !stop.Load() {
				got, err := reader.Get(email)
				if err == nil && string(got) != string(data) {
					err = fmt.Errorf("got %d bytes, want %d", len(got), len(data))
				}
				if err != nil {
					errs <- err
					return
				}
			}
		})
	}
	for range 100 {
		if err := writer.Set(email, data); err != nil {
			t.Fatalf("Set: %v", err)
		}
	}
	stop.Store(true)
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("Get during a write: %v", err)
	}
}

// leaveTempDirs creates what a write killed before its rename leaves in
// credentials/: an empty temp dir and one that holds a copy of the token.
func leaveTempDirs(t *testing.T) (credDir string, left []string) {
	t.Helper()
	credDir, err := config.CredentialsDir()
	if err != nil {
		t.Fatalf("CredentialsDir: %v", err)
	}
	left = []string{filepath.Join(credDir, ".set-111"), filepath.Join(credDir, ".set-222")}
	for _, dir := range left {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(left[1], "gccli:token:left@example.com"), []byte("copy"), 0o600); err != nil {
		t.Fatal(err)
	}
	return credDir, left
}

func assertGone(t *testing.T, paths ...string) {
	t.Helper()
	for _, p := range paths {
		if _, err := os.Lstat(p); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s: Lstat = %v, want not exist", filepath.Base(p), err)
		}
	}
}

func TestFileBackendSetRemovesLeftTempDirs(t *testing.T) {
	testutil.TempConfigDir(t)
	credDir, left := leaveTempDirs(t)
	userFile := filepath.Join(credDir, ".set-notmine")
	if err := os.WriteFile(userFile, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := Open("file")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	if err := s.Set("t@example.com", []byte("data")); err != nil {
		t.Fatalf("Set: %v", err)
	}

	assertGone(t, left...)
	if _, err := os.Stat(userFile); err != nil {
		t.Errorf("Set removed a file that gccli did not create: %v", err)
	}
	if got, err := s.Get("t@example.com"); err != nil || string(got) != "data" {
		t.Errorf("Get = %q, %v, want %q", got, err, "data")
	}
}

func TestFileBackendDeleteRemovesLeftTempDirs(t *testing.T) {
	for _, stored := range []bool{true, false} {
		t.Run(fmt.Sprintf("stored=%v", stored), func(t *testing.T) {
			testutil.TempConfigDir(t)
			s, err := Open("file")
			if err != nil {
				t.Fatalf("Open: %v", err)
			}
			want := error(nil)
			if stored {
				if err := s.Set("t@example.com", []byte("data")); err != nil {
					t.Fatalf("Set: %v", err)
				}
			} else {
				want = ErrNotFound
			}
			_, left := leaveTempDirs(t)

			if err := s.Delete("t@example.com"); !errors.Is(err, want) {
				t.Fatalf("Delete = %v, want %v", err, want)
			}
			assertGone(t, left...)
		})
	}
}

func TestFileBackendParallelWriters(t *testing.T) {
	testutil.TempConfigDir(t)
	email := "parallel@example.com"
	data := []byte(`{"oauth2_access_token":"` + strings.Repeat("a", 1<<12) + `"}`)
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for range 8 {
		s, err := Open("file")
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		wg.Go(func() {
			for range 40 {
				if err := s.Set(email, data); err != nil {
					errs <- err
					return
				}
				if got, err := s.Get(email); err != nil || string(got) != string(data) {
					errs <- fmt.Errorf("Get = %d bytes, %v", len(got), err)
					return
				}
			}
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("parallel write: %v", err)
	}
	credDir, _ := config.CredentialsDir()
	if left, _ := filepath.Glob(filepath.Join(credDir, ".set-*")); len(left) != 0 {
		t.Errorf("temp dirs left after the writes: %v", left)
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

func TestOpen_FallsThroughToNextBackend(t *testing.T) {
	orig := openKeyringFn
	t.Cleanup(func() { openKeyringFn = orig })

	backends := keyring.AvailableBackends()
	if len(backends) < 2 {
		t.Skipf("AvailableBackends = %v, want at least 2", backends)
	}
	second := &memKeyring{items: make(map[string]keyring.Item)}
	var tried []keyring.BackendType
	openKeyringFn = func(cfg keyring.Config) (keyring.Keyring, error) {
		tried = append(tried, cfg.AllowedBackends...)
		if len(tried) == 1 {
			return nil, fmt.Errorf("backend %s unavailable", cfg.AllowedBackends[0])
		}
		return second, nil
	}

	store, err := Open("")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if store.ring != second {
		t.Errorf("Open returned a store on %v, want the second backend", store.ring)
	}
	if len(tried) != 2 || tried[0] != backends[0] || tried[1] != backends[1] {
		t.Errorf("tried backends %v, want %v", tried, backends[:2])
	}
}

func TestOpen_AllBackendsFail(t *testing.T) {
	orig := openKeyringFn
	t.Cleanup(func() { openKeyringFn = orig })

	backends := keyring.AvailableBackends()
	var errs []error
	openKeyringFn = func(cfg keyring.Config) (keyring.Keyring, error) {
		err := fmt.Errorf("backend %s unavailable", cfg.AllowedBackends[0])
		errs = append(errs, err)
		return nil, err
	}

	_, err := Open("")
	if err == nil {
		t.Fatal("expected error")
	}
	if len(errs) != len(backends) {
		t.Fatalf("tried %d backends, want %d", len(errs), len(backends))
	}
	if last := errs[len(errs)-1]; !errors.Is(err, last) {
		t.Errorf("error = %q, want it to wrap the last error %q", err, last)
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
