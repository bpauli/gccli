package garminapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bpauli/gccli/internal/garminauth"
)

// --- CountActivities tests ---

func TestCountActivities_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/activitylist-service/activities/count" {
			t.Errorf("path = %s, want /activitylist-service/activities/count", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		_, _ = w.Write([]byte(`42`))
	})

	_, client := testServer(t, handler)
	count, err := client.CountActivities(context.Background())
	if err != nil {
		t.Fatalf("CountActivities: %v", err)
	}
	if count != 42 {
		t.Errorf("count = %d, want 42", count)
	}
}

func TestCountActivities_InvalidJSON(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`not a number`))
	})

	_, client := testServer(t, handler)
	_, err := client.CountActivities(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestCountActivities_ServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})

	_, client := testServer(t, handler)
	_, err := client.CountActivities(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *GarminAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected GarminAPIError, got %T: %v", err, err)
	}
}

// --- GetActivities tests ---

func TestGetActivities_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/activitylist-service/activities/search/activities" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("start"); got != "0" {
			t.Errorf("start = %s, want 0", got)
		}
		if got := r.URL.Query().Get("limit"); got != "20" {
			t.Errorf("limit = %s, want 20", got)
		}
		_, _ = w.Write([]byte(`[{"activityId":1},{"activityId":2}]`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetActivities(context.Background(), 0, 20, "")
	if err != nil {
		t.Fatalf("GetActivities: %v", err)
	}

	var activities []map[string]any
	if err := json.Unmarshal(data, &activities); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(activities) != 2 {
		t.Errorf("len = %d, want 2", len(activities))
	}
}

func TestGetActivities_WithType(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("activityType"); got != "running" {
			t.Errorf("activityType = %s, want running", got)
		}
		_, _ = w.Write([]byte(`[]`))
	})

	_, client := testServer(t, handler)
	_, err := client.GetActivities(context.Background(), 0, 10, "running")
	if err != nil {
		t.Fatalf("GetActivities: %v", err)
	}
}

// --- GetActivity tests ---

func TestGetActivity_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/activity-service/activity/12345" {
			t.Errorf("path = %s, want /activity-service/activity/12345", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"activityId":12345,"activityName":"Morning Run"}`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetActivity(context.Background(), "12345")
	if err != nil {
		t.Fatalf("GetActivity: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["activityName"] != "Morning Run" {
		t.Errorf("activityName = %v, want Morning Run", result["activityName"])
	}
}

// --- GetActivityDetails tests ---

func TestGetActivityDetails_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/activity-service/activity/123/details" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("maxChartSize"); got != "2000" {
			t.Errorf("maxChartSize = %s, want 2000", got)
		}
		if got := r.URL.Query().Get("maxPolylineSize"); got != "4000" {
			t.Errorf("maxPolylineSize = %s, want 4000", got)
		}
		_, _ = w.Write([]byte(`{"details":true}`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetActivityDetails(context.Background(), "123", 2000, 4000)
	if err != nil {
		t.Fatalf("GetActivityDetails: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["details"] != true {
		t.Errorf("details = %v, want true", result["details"])
	}
}

// --- Activity sub-resource tests ---

func TestGetActivitySplits(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/activity-service/activity/100/splits" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"splits":[]}`))
	})

	_, client := testServer(t, handler)
	_, err := client.GetActivitySplits(context.Background(), "100")
	if err != nil {
		t.Fatalf("GetActivitySplits: %v", err)
	}
}

func TestGetActivityTypedSplits(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/activity-service/activity/100/typedsplits" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{}`))
	})

	_, client := testServer(t, handler)
	_, err := client.GetActivityTypedSplits(context.Background(), "100")
	if err != nil {
		t.Fatalf("GetActivityTypedSplits: %v", err)
	}
}

func TestGetActivitySplitSummaries(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/activity-service/activity/100/split_summaries" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{}`))
	})

	_, client := testServer(t, handler)
	_, err := client.GetActivitySplitSummaries(context.Background(), "100")
	if err != nil {
		t.Fatalf("GetActivitySplitSummaries: %v", err)
	}
}

func TestGetActivityWeather(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/activity-service/activity/100/weather" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"temp":20}`))
	})

	_, client := testServer(t, handler)
	_, err := client.GetActivityWeather(context.Background(), "100")
	if err != nil {
		t.Fatalf("GetActivityWeather: %v", err)
	}
}

func TestGetActivityHRZones(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/activity-service/activity/100/hrTimeInZones" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{}`))
	})

	_, client := testServer(t, handler)
	_, err := client.GetActivityHRZones(context.Background(), "100")
	if err != nil {
		t.Fatalf("GetActivityHRZones: %v", err)
	}
}

func TestGetActivityPowerZones(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/activity-service/activity/100/powerTimeInZones" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{}`))
	})

	_, client := testServer(t, handler)
	_, err := client.GetActivityPowerZones(context.Background(), "100")
	if err != nil {
		t.Fatalf("GetActivityPowerZones: %v", err)
	}
}

func TestGetActivityExerciseSets(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/activity-service/activity/100/exerciseSets" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{}`))
	})

	_, client := testServer(t, handler)
	_, err := client.GetActivityExerciseSets(context.Background(), "100")
	if err != nil {
		t.Fatalf("GetActivityExerciseSets: %v", err)
	}
}

func TestGetActivityGear(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/gear-service/gear" {
			t.Errorf("path = %s, want /gear-service/gear", r.URL.Path)
		}
		if got := r.URL.Query().Get("activityId"); got != "100" {
			t.Errorf("activityId = %s, want 100", got)
		}
		_, _ = w.Write([]byte(`[]`))
	})

	_, client := testServer(t, handler)
	_, err := client.GetActivityGear(context.Background(), "100")
	if err != nil {
		t.Fatalf("GetActivityGear: %v", err)
	}
}

// --- SearchActivities tests ---

func TestSearchActivities_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/activitylist-service/activities/search/activities" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("startDate"); got != "2024-01-01" {
			t.Errorf("startDate = %s, want 2024-01-01", got)
		}
		if got := r.URL.Query().Get("endDate"); got != "2024-01-31" {
			t.Errorf("endDate = %s, want 2024-01-31", got)
		}
		if got := r.URL.Query().Get("start"); got != "0" {
			t.Errorf("start = %s, want 0", got)
		}
		if got := r.URL.Query().Get("limit"); got != "50" {
			t.Errorf("limit = %s, want 50", got)
		}
		_, _ = w.Write([]byte(`[{"activityId":1}]`))
	})

	_, client := testServer(t, handler)
	data, err := client.SearchActivities(context.Background(), 0, 50, "2024-01-01", "2024-01-31")
	if err != nil {
		t.Fatalf("SearchActivities: %v", err)
	}

	var activities []map[string]any
	if err := json.Unmarshal(data, &activities); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(activities) != 1 {
		t.Errorf("len = %d, want 1", len(activities))
	}
}

func TestSearchActivities_NoEndDate(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("endDate"); got != "" {
			t.Errorf("endDate = %s, want empty", got)
		}
		_, _ = w.Write([]byte(`[]`))
	})

	_, client := testServer(t, handler)
	_, err := client.SearchActivities(context.Background(), 0, 20, "2024-01-01", "")
	if err != nil {
		t.Fatalf("SearchActivities: %v", err)
	}
}

// --- DownloadActivity tests ---

func TestDownloadActivity_FIT(t *testing.T) {
	fitData := []byte{0x0E, 0x10, 0x00, 0x00}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/download-service/files/activity/999" {
			t.Errorf("path = %s, want /download-service/files/activity/999", r.URL.Path)
		}
		_, _ = w.Write(fitData)
	})

	_, client := testServer(t, handler)
	data, err := client.DownloadActivity(context.Background(), "999", FormatFIT)
	if err != nil {
		t.Fatalf("DownloadActivity: %v", err)
	}
	if len(data) != len(fitData) {
		t.Errorf("len = %d, want %d", len(data), len(fitData))
	}
}

func TestDownloadActivity_TCX(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/download-service/export/tcx/activity/999" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte("<tcx/>"))
	})

	_, client := testServer(t, handler)
	_, err := client.DownloadActivity(context.Background(), "999", FormatTCX)
	if err != nil {
		t.Fatalf("DownloadActivity: %v", err)
	}
}

func TestDownloadActivity_GPX(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/download-service/export/gpx/activity/999" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte("<gpx/>"))
	})

	_, client := testServer(t, handler)
	_, err := client.DownloadActivity(context.Background(), "999", FormatGPX)
	if err != nil {
		t.Fatalf("DownloadActivity: %v", err)
	}
}

func TestDownloadActivity_KML(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/download-service/export/kml/activity/999" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte("<kml/>"))
	})

	_, client := testServer(t, handler)
	_, err := client.DownloadActivity(context.Background(), "999", FormatKML)
	if err != nil {
		t.Fatalf("DownloadActivity: %v", err)
	}
}

func TestDownloadActivity_CSV(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/download-service/export/csv/activity/999" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte("col1,col2\n"))
	})

	_, client := testServer(t, handler)
	_, err := client.DownloadActivity(context.Background(), "999", FormatCSV)
	if err != nil {
		t.Fatalf("DownloadActivity: %v", err)
	}
}

func TestDownloadActivity_InvalidFormat(t *testing.T) {
	_, client := testServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	_, err := client.DownloadActivity(context.Background(), "999", "xyz")
	if err == nil {
		t.Fatal("expected error for invalid format")
	}

	var fmtErr *InvalidFileFormatError
	if !errors.As(err, &fmtErr) {
		t.Fatalf("expected InvalidFileFormatError, got %T: %v", err, err)
	}
	if fmtErr.Format != "xyz" {
		t.Errorf("Format = %q, want xyz", fmtErr.Format)
	}
}

// --- downloadPath tests ---

func TestDownloadPath(t *testing.T) {
	tests := []struct {
		format ActivityDownloadFormat
		want   string
	}{
		{FormatFIT, "/download-service/files/activity/123"},
		{FormatTCX, "/download-service/export/tcx/activity/123"},
		{FormatGPX, "/download-service/export/gpx/activity/123"},
		{FormatKML, "/download-service/export/kml/activity/123"},
		{FormatCSV, "/download-service/export/csv/activity/123"},
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			got, err := downloadPath("123", tt.format)
			if err != nil {
				t.Fatalf("downloadPath: %v", err)
			}
			if got != tt.want {
				t.Errorf("downloadPath = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDownloadPath_Invalid(t *testing.T) {
	_, err := downloadPath("123", "invalid")
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
}

// --- UploadActivity tests ---

func TestUploadActivity_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/upload-service/upload" {
			t.Errorf("path = %s, want /upload-service/upload", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "multipart/form-data") {
			t.Errorf("Content-Type = %s, want multipart/form-data", ct)
		}

		// Parse multipart form.
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("FormFile: %v", err)
		}
		defer func() { _ = file.Close() }()

		if header.Filename != "test.fit" {
			t.Errorf("filename = %s, want test.fit", header.Filename)
		}
		content, _ := io.ReadAll(file)
		if string(content) != "fitfiledata" {
			t.Errorf("content = %q, want fitfiledata", content)
		}

		_, _ = w.Write([]byte(`{"detailedImportResult":{"uploadId":456}}`))
	})

	_, client := testServer(t, handler)

	// Create temp file.
	dir := t.TempDir()
	path := filepath.Join(dir, "test.fit")
	if err := os.WriteFile(path, []byte("fitfiledata"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	data, err := client.UploadActivity(context.Background(), path)
	if err != nil {
		t.Fatalf("UploadActivity: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
}

func TestUploadActivity_GPX(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}
		_, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("FormFile: %v", err)
		}
		if header.Filename != "route.gpx" {
			t.Errorf("filename = %s, want route.gpx", header.Filename)
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	_, client := testServer(t, handler)

	dir := t.TempDir()
	path := filepath.Join(dir, "route.gpx")
	if err := os.WriteFile(path, []byte("<gpx/>"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	_, err := client.UploadActivity(context.Background(), path)
	if err != nil {
		t.Fatalf("UploadActivity: %v", err)
	}
}

func TestUploadActivity_InvalidFormat(t *testing.T) {
	_, client := testServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	_, err := client.UploadActivity(context.Background(), path)
	if err == nil {
		t.Fatal("expected error for invalid format")
	}

	var fmtErr *InvalidFileFormatError
	if !errors.As(err, &fmtErr) {
		t.Fatalf("expected InvalidFileFormatError, got %T: %v", err, err)
	}
}

func TestUploadActivity_FileNotFound(t *testing.T) {
	_, client := testServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	_, err := client.UploadActivity(context.Background(), "/nonexistent/file.fit")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestUploadActivity_ServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad request"))
	})

	_, client := testServer(t, handler)

	dir := t.TempDir()
	path := filepath.Join(dir, "test.fit")
	if err := os.WriteFile(path, []byte("fitdata"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	_, err := client.UploadActivity(context.Background(), path)
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *GarminAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected GarminAPIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 400 {
		t.Errorf("StatusCode = %d, want 400", apiErr.StatusCode)
	}
}

func TestUploadActivity_Unauthorized_Refresh(t *testing.T) {
	calls := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer refreshed-token" {
			t.Errorf("Authorization after refresh = %q, want Bearer refreshed-token", got)
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	tokens := testTokens()
	client := NewClient(tokens,
		WithHTTPClient(srv.Client()),
		WithBaseURL(srv.URL),
	)

	origRefresh := refreshTokensFn
	t.Cleanup(func() { refreshTokensFn = origRefresh })
	refreshTokensFn = func(_ context.Context, _ *garminauth.Tokens, _ garminauth.LoginOptions) (*garminauth.Tokens, error) {
		newTokens := *tokens
		newTokens.OAuth2AccessToken = "refreshed-token"
		return &newTokens, nil
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "test.tcx")
	if err := os.WriteFile(path, []byte("<tcx/>"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	data, err := client.UploadActivity(context.Background(), path)
	if err != nil {
		t.Fatalf("UploadActivity: %v", err)
	}
	if data == nil {
		t.Error("expected non-nil response")
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
}

// --- CreateManualActivity tests ---

func TestCreateManualActivity_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/activity-service/activity" {
			t.Errorf("path = %s, want /activity-service/activity", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}

		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		if payload["activityName"] != "Morning Run" {
			t.Errorf("activityName = %v, want Morning Run", payload["activityName"])
		}
		actType, ok := payload["activityTypeDTO"].(map[string]any)
		if !ok {
			t.Fatal("activityTypeDTO not a map")
		}
		if actType["typeKey"] != "running" {
			t.Errorf("typeKey = %v, want running", actType["typeKey"])
		}
		summary, ok := payload["summaryDTO"].(map[string]any)
		if !ok {
			t.Fatal("summaryDTO not a map")
		}
		if summary["distance"] != float64(5000) {
			t.Errorf("distance = %v, want 5000", summary["distance"])
		}
		if summary["duration"] != float64(1800) {
			t.Errorf("duration = %v, want 1800", summary["duration"])
		}

		_, _ = w.Write([]byte(`{"activityId":99999}`))
	})

	_, client := testServer(t, handler)
	data, err := client.CreateManualActivity(context.Background(),
		"Morning Run", "running", 5000, 1800, "2024-01-15T08:00:00")
	if err != nil {
		t.Fatalf("CreateManualActivity: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
}

// --- RenameActivity tests ---

func TestRenameActivity_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/activity-service/activity/12345" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}

		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		if payload["activityName"] != "Evening Run" {
			t.Errorf("activityName = %v, want Evening Run", payload["activityName"])
		}
		// activityId should be numeric.
		if payload["activityId"] != float64(12345) {
			t.Errorf("activityId = %v, want 12345", payload["activityId"])
		}

		_, _ = w.Write([]byte(`{"activityId":12345,"activityName":"Evening Run"}`))
	})

	_, client := testServer(t, handler)
	_, err := client.RenameActivity(context.Background(), "12345", "Evening Run")
	if err != nil {
		t.Fatalf("RenameActivity: %v", err)
	}
}

func TestRenameActivity_InvalidID(t *testing.T) {
	_, client := testServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	_, err := client.RenameActivity(context.Background(), "not-a-number", "Name")
	if err == nil {
		t.Fatal("expected error for invalid ID")
	}
}

// --- RetypeActivity tests ---

func TestRetypeActivity_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/activity-service/activity/12345" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}

		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		actType, ok := payload["activityTypeDTO"].(map[string]any)
		if !ok {
			t.Fatal("activityTypeDTO not a map")
		}
		if actType["typeId"] != float64(1) {
			t.Errorf("typeId = %v, want 1", actType["typeId"])
		}
		if actType["typeKey"] != "running" {
			t.Errorf("typeKey = %v, want running", actType["typeKey"])
		}
		if actType["parentTypeId"] != float64(17) {
			t.Errorf("parentTypeId = %v, want 17", actType["parentTypeId"])
		}

		_, _ = w.Write([]byte(`{"activityId":12345}`))
	})

	_, client := testServer(t, handler)
	_, err := client.RetypeActivity(context.Background(), "12345", 1, "running", 17)
	if err != nil {
		t.Fatalf("RetypeActivity: %v", err)
	}
}

func TestRetypeActivity_InvalidID(t *testing.T) {
	_, client := testServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	_, err := client.RetypeActivity(context.Background(), "abc", 1, "running", 17)
	if err == nil {
		t.Fatal("expected error for invalid ID")
	}
}

// --- DeleteActivity tests ---

func TestDeleteActivity_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/activity-service/activity/12345" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	_, client := testServer(t, handler)
	err := client.DeleteActivity(context.Background(), "12345")
	if err != nil {
		t.Fatalf("DeleteActivity: %v", err)
	}
}

func TestDeleteActivity_ServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	})

	_, client := testServer(t, handler)
	err := client.DeleteActivity(context.Background(), "99999")
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *GarminAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected GarminAPIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
}
