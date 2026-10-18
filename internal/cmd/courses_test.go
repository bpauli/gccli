package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bpauli/gccli/internal/outfmt"
)

func coursesTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/web-gateway/course/owner/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"coursesForUser":[{"courseId":123,"courseName":"Morning Run","courseType":{"typeKey":"running"},"distanceInMeters":5000,"createdDate":"2024-06-15 10:30:00"}]}`))
	})

	mux.HandleFunc("/course-service/course/import", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"courseName":"GPX Route","geoPoints":[{"lat":47.0,"lon":8.0,"distance":0},{"lat":47.1,"lon":8.1,"distance":1000}]}`))
	})

	mux.HandleFunc("/course-service/course/elevation", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[[47.0,8.0,500.0],[47.1,8.1,510.0]]`))
	})

	mux.HandleFunc("/course-service/course/favorites", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"courseId":456,"courseName":"Fav Route","courseType":{"typeKey":"cycling"},"distanceInMeters":20000,"createdDate":"2024-03-01 08:00:00"}]`))
	})

	mux.HandleFunc("/course-service/course/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Extract course ID from path for detail/delete requests.
		if strings.HasSuffix(r.URL.Path, "/123") {
			switch r.Method {
			case http.MethodGet:
				_, _ = w.Write([]byte(`{"courseId":123,"courseName":"Morning Run","distanceInMeters":5000}`))
			case http.MethodDelete:
				w.WriteHeader(http.StatusNoContent)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	})

	mux.HandleFunc("/course-service/course", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		body, _ := io.ReadAll(r.Body)
		var course map[string]any
		if err := json.Unmarshal(body, &course); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		name := "GPX Route"
		if n, ok := course["courseName"].(string); ok {
			name = n
		}
		_, _ = w.Write([]byte(`{"courseId":999,"courseName":"` + name + `","distanceMeter":10000,"elevationGainMeter":150,"elevationLossMeter":140,"createDate":"2024-06-15"}`))
	})

	mux.HandleFunc("/device-service/devicemessage/messages", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		body, _ := io.ReadAll(r.Body)
		var payload []map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"messageId":999,"messageStatus":"new","deviceName":"Forerunner 265"}]`))
	})

	return httptest.NewServer(mux)
}

// --- Execute-level tests ---

// --- CoursesListCmd tests ---

func TestCoursesList_Success_JSON(t *testing.T) {
	server := coursesTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &CoursesListCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestCoursesList_Success_Table(t *testing.T) {
	server := coursesTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &CoursesListCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestCoursesList_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/web-gateway/course/owner/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &CoursesListCmd{}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- CoursesFavoritesCmd tests ---

func TestCoursesFavorites_Success(t *testing.T) {
	server := coursesTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &CoursesFavoritesCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// --- CourseDetailCmd tests ---

func TestCourseDetail_Success(t *testing.T) {
	server := coursesTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &CourseDetailCmd{ID: "123"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// --- CourseSendCmd tests ---

func TestCourseSend_Success_Table(t *testing.T) {
	server := coursesTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &CourseSendCmd{CourseID: "123", DeviceID: "456"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "Morning Run") {
		t.Errorf("expected success message with course name, got: %s", buf.String())
	}
}

func TestCourseSend_CourseNotFound(t *testing.T) {
	server := coursesTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &CourseSendCmd{CourseID: "99999", DeviceID: "456"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- CourseImportCmd tests ---

func newTestGPXFile(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "route.gpx")
	if err := os.WriteFile(path, []byte("<gpx>test</gpx>"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}

func TestCourseImport_InvalidType(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &CourseImportCmd{File: "route.gpx", Type: "swimming"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unknown activity type") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCourseImport_Success(t *testing.T) {
	server := coursesTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	gpxPath := newTestGPXFile(t)

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &CourseImportCmd{File: gpxPath, Type: "cycling"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestCourseImport_Flags(t *testing.T) {
	var save map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("/course-service/course/import", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"courseName":"GPX Route","geoPoints":[{"lat":47.0,"lon":8.0}]}`))
	})
	mux.HandleFunc("/course-service/course/elevation", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[[47.0,8.0,500.0]]`))
	})
	mux.HandleFunc("/course-service/course", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &save)
		_, _ = w.Write([]byte(`{"courseId":999}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &CourseImportCmd{File: newTestGPXFile(t), Name: "Sunday Ride", Type: "hiking", Privacy: 1}
	if err := cmd.Run(g); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if save["courseName"] != "Sunday Ride" || save["activityTypePk"] != float64(3) || save["coursePrivacy"] != float64(1) {
		t.Errorf("save body = %v, want courseName Sunday Ride, activityTypePk 3, coursePrivacy 1", save)
	}
}

// --- CourseDeleteCmd tests ---

func TestCourseDelete_Success(t *testing.T) {
	server := coursesTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &CourseDeleteCmd{ID: "123", Force: true}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "Deleted course") {
		t.Fatalf("expected 'Deleted course' message, got: %q", buf.String())
	}
}

func TestCourseDelete_Cancelled(t *testing.T) {
	server := coursesTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	orig := confirmReader
	confirmReader = strings.NewReader("n\n")
	t.Cleanup(func() { confirmReader = orig })

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &CourseDeleteCmd{ID: "123"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "Cancelled") {
		t.Fatalf("expected 'Cancelled' message, got: %q", buf.String())
	}
}

func TestCourseDelete_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/course-service/course/123", func(w http.ResponseWriter, _ *http.Request) {
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
	cmd := &CourseDeleteCmd{ID: "123", Force: true}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "delete course") {
		t.Fatalf("unexpected error: %v", err)
	}
}
