package garminapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestCoursesEndpoints(t *testing.T) {
	id := map[string]string{"courseID": "12345"}
	testEndpoints(t, coursesEndpoints, []endpointCase{
		{"courses.list", nil, "GET /web-gateway/course/owner/"},
		{"courses.get", id, "GET /course-service/course/12345"},
		{"courses.favorites", nil, "GET /course-service/course/favorites"},
		{"courses.elevation", nil, "POST /course-service/course/elevation?smoothingEnabled=true"},
		{"courses.save", nil, "POST /course-service/course"},
		{"courses.delete", id, "DELETE /course-service/course/12345"},
	})
}

// --- ImportCourseGPX tests ---

func TestImportCourseGPX_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/course-service/course/import" {
			t.Errorf("path = %s, want /course-service/course/import", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "multipart/form-data") {
			t.Errorf("Content-Type = %s, want multipart/form-data", ct)
		}

		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("FormFile: %v", err)
		}
		defer func() { _ = file.Close() }()

		if header.Filename != "route.gpx" {
			t.Errorf("filename = %s, want route.gpx", header.Filename)
		}
		content, _ := io.ReadAll(file)
		if string(content) != "<gpx>test</gpx>" {
			t.Errorf("content = %q, want <gpx>test</gpx>", content)
		}

		_, _ = w.Write([]byte(`{"courseName":"Test Route","geoPoints":[{"lat":47.0,"lon":8.0}]}`))
	})

	_, client := testServer(t, handler)

	dir := t.TempDir()
	path := filepath.Join(dir, "route.gpx")
	if err := os.WriteFile(path, []byte("<gpx>test</gpx>"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	data, err := client.ImportCourseGPX(context.Background(), path)
	if err != nil {
		t.Fatalf("ImportCourseGPX: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["courseName"] != "Test Route" {
		t.Errorf("courseName = %v, want Test Route", result["courseName"])
	}
}

func TestImportCourseGPX_InvalidFormat(t *testing.T) {
	_, client := testServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	dir := t.TempDir()
	path := filepath.Join(dir, "route.tcx")
	if err := os.WriteFile(path, []byte("<tcx/>"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	_, err := client.ImportCourseGPX(context.Background(), path)
	if err == nil {
		t.Fatal("expected error for non-GPX file")
	}

	var fmtErr *InvalidFileFormatError
	if !errors.As(err, &fmtErr) {
		t.Fatalf("expected InvalidFileFormatError, got %T: %v", err, err)
	}
}

func TestImportCourseGPX_FileNotFound(t *testing.T) {
	_, client := testServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	_, err := client.ImportCourseGPX(context.Background(), "/nonexistent/route.gpx")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

// --- SaveCourse tests ---

type importCourseServer struct {
	client   *Client
	requests []string
	elev     string
	save     string
}

func newImportCourseServer(t *testing.T, importBody string, elevStatus int) *importCourseServer {
	t.Helper()
	s := &importCourseServer{}
	_, s.client = testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		s.requests = append(s.requests, r.Method+" "+r.URL.RequestURI())
		switch r.URL.Path {
		case "/course-service/course/import":
			_, _ = w.Write([]byte(importBody))
		case "/course-service/course/elevation":
			s.elev = string(body)
			if elevStatus != 0 {
				w.WriteHeader(elevStatus)
				return
			}
			_, _ = w.Write([]byte(`[[47,8,500],[47.1,8.1,510]]`))
		case "/course-service/course":
			s.save = string(body)
			_, _ = w.Write([]byte(`{"courseId":999}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	return s
}

func writeGPX(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte("<gpx/>"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}

func TestImportCourse(t *testing.T) {
	tests := []struct {
		name       string
		importBody string
		opts       CourseImportOptions
		wantSave   string
	}{
		{
			name:       "fills defaults and the file name",
			importBody: `{"courseName":"","extra":1,"geoPoints":[{"latitude":47,"longitude":8},{"lat":47.1,"lon":8.1}]}`,
			opts:       CourseImportOptions{ActivityTypePK: 3, Privacy: 1},
			wantSave: `{"activityTypePk":3,"coordinateSystem":"WGS84","courseName":"route","coursePrivacy":1,
				"geoPoints":[{"latitude":47,"longitude":8,"elevation":500},{"lat":47.1,"lon":8.1,"elevation":510}],
				"rulePK":2,"sourceTypeId":3,"startPoint":{"latitude":47,"longitude":8,"elevation":500}}`,
		},
		{
			name:       "keeps the GPX values",
			importBody: `{"courseName":"Loop","coordinateSystem":"X","sourceTypeId":5,"rulePK":7,"startPoint":{"latitude":1},"geoPoints":[{"latitude":47,"longitude":8}]}`,
			opts:       CourseImportOptions{ActivityTypePK: 2, Privacy: 2},
			wantSave: `{"activityTypePk":2,"coordinateSystem":"X","courseName":"Loop","coursePrivacy":2,
				"geoPoints":[{"latitude":47,"longitude":8,"elevation":500}],
				"rulePK":7,"sourceTypeId":5,"startPoint":{"latitude":1}}`,
		},
		{
			name:       "name option replaces the GPX name",
			importBody: `{"courseName":"Loop","geoPoints":[{"latitude":47,"longitude":8}]}`,
			opts:       CourseImportOptions{Name: "Ride", ActivityTypePK: 2, Privacy: 4},
			wantSave: `{"activityTypePk":2,"coordinateSystem":"WGS84","courseName":"Ride","coursePrivacy":4,
				"geoPoints":[{"latitude":47,"longitude":8,"elevation":500}],
				"rulePK":2,"sourceTypeId":3,"startPoint":{"latitude":47,"longitude":8,"elevation":500}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newImportCourseServer(t, tt.importBody, 0)

			saved, err := s.client.ImportCourse(context.Background(), writeGPX(t, "route.gpx"), tt.opts)
			if err != nil {
				t.Fatalf("ImportCourse: %v", err)
			}
			if string(saved) != `{"courseId":999}` {
				t.Errorf("saved = %s, want the save response", saved)
			}
			wantRequests := []string{
				"POST /course-service/course/import",
				"POST /course-service/course/elevation?smoothingEnabled=true",
				"POST /course-service/course",
			}
			if !slices.Equal(s.requests, wantRequests) {
				t.Errorf("requests = %q, want %q", s.requests, wantRequests)
			}
			var gotSave, wantSave any
			if err := json.Unmarshal([]byte(s.save), &gotSave); err != nil {
				t.Fatalf("save body %q: %v", s.save, err)
			}
			if err := json.Unmarshal([]byte(tt.wantSave), &wantSave); err != nil {
				t.Fatalf("want save: %v", err)
			}
			if !reflect.DeepEqual(gotSave, wantSave) {
				t.Errorf("save body = %s\nwant %s", s.save, tt.wantSave)
			}
		})
	}
}

func TestImportCourse_ElevationInput(t *testing.T) {
	s := newImportCourseServer(t, `{"geoPoints":[{"latitude":47,"longitude":8},{"lat":47.1,"lon":8.1},{"latitude":1},"x"]}`, 0)

	if _, err := s.client.ImportCourse(context.Background(), writeGPX(t, "route.gpx"), CourseImportOptions{}); err != nil {
		t.Fatalf("ImportCourse: %v", err)
	}
	if want := `[[47,8,null],[47.1,8.1,null]]`; s.elev != want {
		t.Errorf("elevation body = %s, want %s", s.elev, want)
	}
}

func TestImportCourse_Errors(t *testing.T) {
	tests := []struct {
		name         string
		file         string
		importBody   string
		elevStatus   int
		wantErr      string
		wantRequests int
	}{
		{"not a GPX file", "run.fit", "", 0, "import GPX: invalid file format: fit", 0},
		{"bad import JSON", "route.gpx", `{`, 0, "parse imported course: unexpected end of JSON input", 1},
		{"no geo points", "route.gpx", `{"geoPoints":[]}`, 0, "imported course has no geo points", 1},
		{"elevation fails", "route.gpx", `{"geoPoints":[{"latitude":47,"longitude":8}]}`, http.StatusInternalServerError, "get elevation: garmin api: 500 Internal Server Error", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newImportCourseServer(t, tt.importBody, tt.elevStatus)

			_, err := s.client.ImportCourse(context.Background(), writeGPX(t, tt.file), CourseImportOptions{})
			if err == nil || !strings.HasPrefix(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want prefix %q", err, tt.wantErr)
			}
			if len(s.requests) != tt.wantRequests {
				t.Errorf("requests = %q, want %d", s.requests, tt.wantRequests)
			}
		})
	}
}

// --- SendCourseToDevice tests ---

func TestSendCourseToDevice_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/device-service/devicemessage/messages" {
			t.Errorf("path = %s, want /device-service/devicemessage/messages", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}

		body, _ := io.ReadAll(r.Body)
		var payload []map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}
		if len(payload) != 1 {
			t.Fatalf("expected 1 message, got %d", len(payload))
		}
		msg := payload[0]
		if msg["messageType"] != "courses" {
			t.Errorf("messageType = %v, want courses", msg["messageType"])
		}
		if msg["messageName"] != "Morning Run" {
			t.Errorf("messageName = %v, want Morning Run", msg["messageName"])
		}
		if msg["fileType"] != "FIT" {
			t.Errorf("fileType = %v, want FIT", msg["fileType"])
		}
		if msg["messageUrl"] != "course-service/course/fit/12345/67890?elevation=true" {
			t.Errorf("messageUrl = %v, want course-service/course/fit/12345/67890?elevation=true", msg["messageUrl"])
		}
		// deviceId should be numeric
		if msg["deviceId"] != float64(67890) {
			t.Errorf("deviceId = %v, want 67890", msg["deviceId"])
		}
		// metaDataId should be numeric
		if msg["metaDataId"] != float64(12345) {
			t.Errorf("metaDataId = %v, want 12345", msg["metaDataId"])
		}

		_, _ = w.Write([]byte(`[{"messageId":999,"messageStatus":"new","deviceName":"Forerunner 265"}]`))
	})

	_, client := testServer(t, handler)
	data, err := client.SendCourseToDevice(context.Background(), 12345, 67890, "Morning Run")
	if err != nil {
		t.Fatalf("SendCourseToDevice: %v", err)
	}

	var result []map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result[0]["messageStatus"] != "new" {
		t.Errorf("messageStatus = %v, want new", result[0]["messageStatus"])
	}
}

func TestSendCourseToDevice_ServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})

	_, client := testServer(t, handler)
	_, err := client.SendCourseToDevice(context.Background(), 12345, 67890, "Test")
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *GarminAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected GarminAPIError, got %T: %v", err, err)
	}
}

// --- DownloadCourse tests ---

func TestDownloadCourse_FIT(t *testing.T) {
	want := []byte("raw-fit-bytes")
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/course-service/course/fit/12345/0" {
			t.Errorf("path = %s, want /course-service/course/fit/12345/0", r.URL.Path)
		}
		if r.URL.Query().Get("elevation") != "true" {
			t.Errorf("elevation = %q, want true", r.URL.Query().Get("elevation"))
		}
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		_, _ = w.Write(want)
	})

	_, client := testServer(t, handler)
	got, err := client.DownloadCourse(context.Background(), "12345", FormatFIT)
	if err != nil {
		t.Fatalf("DownloadCourse: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("body = %q, want %q", got, want)
	}
}

func TestDownloadCourse_GPX(t *testing.T) {
	want := []byte("<gpx/>")
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/course-service/course/gpx/12345" {
			t.Errorf("path = %s, want /course-service/course/gpx/12345", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		_, _ = w.Write(want)
	})

	_, client := testServer(t, handler)
	got, err := client.DownloadCourse(context.Background(), "12345", FormatGPX)
	if err != nil {
		t.Fatalf("DownloadCourse: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("body = %q, want %q", got, want)
	}
}

func TestDownloadCourse_InvalidFormat(t *testing.T) {
	_, client := testServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	_, err := client.DownloadCourse(context.Background(), "12345", FormatTCX)
	if err == nil {
		t.Fatal("expected error for unsupported course format")
	}

	var fmtErr *InvalidFileFormatError
	if !errors.As(err, &fmtErr) {
		t.Fatalf("expected InvalidFileFormatError, got %T: %v", err, err)
	}
	if fmtErr.Format != "tcx" {
		t.Errorf("Format = %q, want tcx", fmtErr.Format)
	}
}

func TestDownloadCourse_ServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	})

	_, client := testServer(t, handler)
	_, err := client.DownloadCourse(context.Background(), "12345", FormatGPX)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- downloadCoursePath tests ---

func TestDownloadCoursePath(t *testing.T) {
	tests := []struct {
		format DownloadFormat
		want   string
	}{
		{FormatFIT, "/course-service/course/fit/777/0?elevation=true"},
		{FormatGPX, "/course-service/course/gpx/777"},
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			got, err := downloadCoursePath("777", tt.format)
			if err != nil {
				t.Fatalf("downloadCoursePath: %v", err)
			}
			if got != tt.want {
				t.Errorf("downloadCoursePath = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDownloadCoursePath_Invalid(t *testing.T) {
	for _, format := range []DownloadFormat{FormatTCX, FormatKML, FormatCSV, "xyz"} {
		t.Run(string(format), func(t *testing.T) {
			_, err := downloadCoursePath("777", format)
			if err == nil {
				t.Fatalf("expected error for format %q", format)
			}
			var fmtErr *InvalidFileFormatError
			if !errors.As(err, &fmtErr) {
				t.Fatalf("expected InvalidFileFormatError, got %T: %v", err, err)
			}
		})
	}
}
