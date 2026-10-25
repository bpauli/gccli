package garminapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bpauli/gccli/internal/garminauth"
)

func TestActivitiesEndpoints(t *testing.T) {
	id := map[string]string{"activityID": "123"}
	page := func(kv ...string) map[string]string {
		args := map[string]string{"start": "0", "limit": "20"}
		for i := 0; i < len(kv); i += 2 {
			args[kv[i]] = kv[i+1]
		}
		return args
	}
	const list = "GET /activitylist-service/activities/search/activities?start=0&limit=20"
	testEndpoints(t, activitiesEndpoints, []endpointCase{
		{"activities.list", page("activityType", ""), list},
		{"activities.list", page("activityType", "running"), list + "&activityType=running"},
		{"activities.list", page("activityType", "a b&c"), list + "&activityType=a+b%26c"},
		{"activities.get", id, "GET /activity-service/activity/123"},
		{"activities.details", map[string]string{"activityID": "123", "maxChart": "2000", "maxPoly": "4000"}, "GET /activity-service/activity/123/details?maxChartSize=2000&maxPolylineSize=4000"},
		{"activities.splits", id, "GET /activity-service/activity/123/splits"},
		{"activities.typed-splits", id, "GET /activity-service/activity/123/typedsplits"},
		{"activities.split-summaries", id, "GET /activity-service/activity/123/split_summaries"},
		{"activities.weather", id, "GET /activity-service/activity/123/weather"},
		{"activities.hr-zones", id, "GET /activity-service/activity/123/hrTimeInZones"},
		{"activities.power-zones", id, "GET /activity-service/activity/123/powerTimeInZones"},
		{"activities.exercise-sets", id, "GET /activity-service/activity/123/exerciseSets"},
		{"activities.gear", id, "GET /gear-service/gear?activityId=123"},
		{"activities.search", page("startDate", "2024-01-01", "endDate", "2024-01-31"), list + "&startDate=2024-01-01&endDate=2024-01-31"},
		{"activities.search", page("startDate", "2024-01-01", "endDate", ""), list + "&startDate=2024-01-01"},
		{"activities.set-exercise-sets", id, "PUT /activity-service/activity/123/exerciseSets"},
		{"activities.create", nil, "POST /activity-service/activity"},
		{"activities.update", id, "PUT /activity-service/activity/123"},
		{"activities.delete", id, "DELETE /activity-service/activity/123"},
		{"activities.favorites", map[string]string{"displayName": "jo", "limit": "20"}, "GET /activitylist-service/activities/favorite/jo?limit=20"},
		{"activities.types", nil, "GET /activity-service/activity/activityTypes"},
		{"activities.pending", nil, "GET /pendingactivity-service/pending-activities"},
		{"activities.statistics", map[string]string{"displayName": "jo"}, "GET /userstats-service/statistics/jo"},
		{"activities.monthly-statistics", map[string]string{"displayName": "jo"}, "GET /userstats-service/statistics/monthly/jo?ByParentType=true"},
		{"activities.comments", id, "GET /conversation-service/conversation/comments/ACTIVITY/123?start=0&limit=20"},
		{"activities.likes", id, "GET /conversation-service/conversation/likes/ACTIVITY/123"},
		{"activities.conversation", id, "GET /conversation-service/conversation/ACTIVITY/123?likerLimit=3"},
		{"activities.like", id, "POST /conversation-service/conversation/like/ACTIVITY/123"},
		{"activities.unlike", map[string]string{"conversationUUID": "9f0c-4e"}, "DELETE /conversation-service/conversation/like/9f0c-4e/0"},
		{"activities.unlike", map[string]string{"conversationUUID": "a/b"}, "DELETE /conversation-service/conversation/like/a%2Fb/0"},
		{"activities.add-comment", id, "POST /conversation-service/conversation/comment/ACTIVITY/123/null"},
		{"activities.edit-comment", map[string]string{"conversationUUID": "c1", "commentUUID": "k1"}, "PUT /conversation-service/conversation/comment/c1/k1"},
		{"activities.delete-comment", map[string]string{"conversationUUID": "c1", "commentUUID": "k1"}, "DELETE /conversation-service/conversation/comment/c1/k1/0"},
		{"activities.comment-likes", map[string]string{"commentUUID": "k1"}, "GET /conversation-service/conversation/comment/likes/k1"},
		{"activities.like-comment", map[string]string{"commentUUID": "k1"}, "POST /conversation-service/conversation/comment/like/k1"},
		{"activities.unlike-comment", map[string]string{"commentUUID": "k1"}, "DELETE /conversation-service/conversation/comment/like/k1/0"},
	})
}

func TestActivitiesRefuseAnEmptyID(t *testing.T) {
	_, client := testServer(t, http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("request sent: %s %s", r.Method, r.URL.RequestURI())
	}))
	for _, name := range []string{"activities.get", "activities.gear", "activities.delete"} {
		_, err := client.Call(context.Background(), name, map[string]string{"activityID": ""}, nil)
		if want := "endpoint " + name + ": activityID: required"; err == nil || err.Error() != want {
			t.Errorf("%s: err = %v, want %q", name, err, want)
		}
	}
}

// --- CountActivities tests ---

func TestCountActivities_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/activitylist-service/activities/count" {
			t.Errorf("path = %s, want /activitylist-service/activities/count", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		_, _ = w.Write([]byte(`{"totalCount":42}`))
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

// An empty body (a 204 or an empty 200) counts as 0, not as an error.
func TestCountActivities_EmptyBody(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusNoContent} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
			})

			_, client := testServer(t, handler)
			count, err := client.CountActivities(context.Background())
			if err != nil {
				t.Fatalf("CountActivities: %v", err)
			}
			if count != 0 {
				t.Errorf("count = %d, want 0", count)
			}
		})
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
		format DownloadFormat
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
		"Morning Run", "running", "America/New_York", 5000, 1800, "2024-01-15T08:00:00")
	if err != nil {
		t.Fatalf("CreateManualActivity: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
}

// --- UpdateActivity tests ---

func TestUpdateActivity_SendsFieldsWithNumericID(t *testing.T) {
	var got []byte
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/activity-service/activity/12345" {
			t.Errorf("request = %s %s, want PUT /activity-service/activity/12345", r.Method, r.URL.Path)
		}
		got, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNoContent)
	})

	_, client := testServer(t, handler)
	fields := map[string]any{"description": nil, "activityId": "999"}
	if _, err := client.UpdateActivity(context.Background(), 12345, fields); err != nil {
		t.Fatalf("UpdateActivity: %v", err)
	}
	if want := `{"activityId":12345,"description":null}`; string(got) != want {
		t.Errorf("body = %s, want %s", got, want)
	}
}
