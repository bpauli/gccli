package cmd

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bpauli/gccli/internal/outfmt"
)

// makeZipWithFIT creates a zip archive containing a .fit file with the given content.
func makeZipWithFIT(t *testing.T, fitData []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.Create("12345678_ACTIVITY.fit")
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	if _, err := f.Write(fitData); err != nil {
		t.Fatalf("write zip entry: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	return buf.Bytes()
}

// makeZipWithoutFIT creates a zip archive containing a non-.fit file.
func makeZipWithoutFIT(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.Create("readme.txt")
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	if _, err := f.Write([]byte("hello")); err != nil {
		t.Fatalf("write zip entry: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	return buf.Bytes()
}

func downloadTestServer(t *testing.T, fitZipData, gpxData []byte) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/download-service/files/activity/12345678", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(fitZipData)
	})

	mux.HandleFunc("/download-service/export/gpx/activity/12345678", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gpx+xml")
		_, _ = w.Write(gpxData)
	})

	mux.HandleFunc("/download-service/export/tcx/activity/12345678", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte("<tcx>data</tcx>"))
	})

	mux.HandleFunc("/download-service/export/kml/activity/12345678", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte("<kml>data</kml>"))
	})

	mux.HandleFunc("/download-service/export/csv/activity/12345678", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte("col1,col2\nval1,val2"))
	})

	return httptest.NewServer(mux)
}

// --- Execute-level tests ---

func TestExecute_ActivityDownloadHelp(t *testing.T) {
	code := Execute([]string{"activity", "download", "--help"}, "1.0.0", "abc123", "2024-01-01")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

// --- ActivityDownloadCmd tests ---

func TestActivityDownload_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &ActivityDownloadCmd{ID: "12345678", Format: "fit"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestActivityDownload_FIT(t *testing.T) {
	fitContent := []byte("FIT-BINARY-DATA-HERE")
	zipData := makeZipWithFIT(t, fitContent)

	server := downloadTestServer(t, zipData, nil)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	outDir := t.TempDir()
	outPath := filepath.Join(outDir, "activity_12345678.fit")

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &ActivityDownloadCmd{ID: "12345678", Format: "fit", Output: outPath}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify the extracted FIT content (not the zip).
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if !bytes.Equal(data, fitContent) {
		t.Errorf("expected FIT content %q, got %q", fitContent, data)
	}
}

func TestActivityDownload_GPX(t *testing.T) {
	gpxContent := []byte("<gpx>track data</gpx>")

	server := downloadTestServer(t, nil, gpxContent)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	outDir := t.TempDir()
	outPath := filepath.Join(outDir, "activity_12345678.gpx")

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &ActivityDownloadCmd{ID: "12345678", Format: "gpx", Output: outPath}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if !bytes.Equal(data, gpxContent) {
		t.Errorf("expected GPX content %q, got %q", gpxContent, data)
	}
}

func TestActivityDownload_TCX(t *testing.T) {
	server := downloadTestServer(t, nil, nil)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	outDir := t.TempDir()
	outPath := filepath.Join(outDir, "activity_12345678.tcx")

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &ActivityDownloadCmd{ID: "12345678", Format: "tcx", Output: outPath}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if string(data) != "<tcx>data</tcx>" {
		t.Errorf("unexpected TCX content: %q", data)
	}
}

func TestActivityDownload_DefaultFilename(t *testing.T) {
	gpxContent := []byte("<gpx>data</gpx>")

	server := downloadTestServer(t, nil, gpxContent)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	// Change to temp dir so default filename lands there.
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &ActivityDownloadCmd{ID: "12345678", Format: "gpx"}
	err = cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	expectedFile := filepath.Join(tmpDir, "activity_12345678.gpx")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Fatalf("expected default output file %q to exist", expectedFile)
	}
}

func TestActivityDownload_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/download-service/files/activity/12345678", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &ActivityDownloadCmd{ID: "12345678", Format: "fit"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "download activity") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- extractFIT tests ---

func TestExtractFIT_Success(t *testing.T) {
	fitContent := []byte("FIT-BINARY-DATA")
	zipData := makeZipWithFIT(t, fitContent)

	result, err := extractFIT(zipData)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !bytes.Equal(result, fitContent) {
		t.Errorf("expected %q, got %q", fitContent, result)
	}
}

func TestExtractFIT_NoFitFile(t *testing.T) {
	zipData := makeZipWithoutFIT(t)

	_, err := extractFIT(zipData)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no .fit file found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExtractFIT_InvalidZip(t *testing.T) {
	_, err := extractFIT([]byte("not a zip file"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "open zip") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExtractFIT_EmptyZip(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	if err := w.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}

	_, err := extractFIT(buf.Bytes())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no .fit file found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- defaultFilename tests ---

func TestDefaultFilename(t *testing.T) {
	tests := []struct {
		id     string
		format string
		want   string
	}{
		{"12345678", "fit", "activity_12345678.fit"},
		{"99999", "gpx", "activity_99999.gpx"},
		{"1", "tcx", "activity_1.tcx"},
		{"42", "kml", "activity_42.kml"},
		{"100", "csv", "activity_100.csv"},
	}
	for _, tt := range tests {
		got := defaultActivityFilename(tt.id, tt.format)
		if got != tt.want {
			t.Errorf("defaultActivityFilename(%q, %q) = %q, want %q", tt.id, tt.format, got, tt.want)
		}
	}
}
