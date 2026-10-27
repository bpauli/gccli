package garminapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestReportExportEndpoints(t *testing.T) {
	testEndpoints(t, reportExportEndpoints, []endpointCase{
		{"reportexport.init", map[string]string{"category": "bloodPressure", "start": "2026-10-01", "end": "2026-10-07", "locale": "en"}, "POST /reportexport-service/export/init/bloodPressure/2026-10-01/2026-10-07?locale=en"},
		{"reportexport.status", map[string]string{"id": "abc="}, "GET /reportexport-service/export/reportStatus/abc="},
	})
}

type reportServer struct {
	statuses []string
	estimate string
	reportID string
	initBody map[string]any
	initLine string
	paths    []string
	accepts  []string
}

func (s *reportServer) handler(t *testing.T) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.paths = append(s.paths, r.Method+" "+r.URL.EscapedPath())
		switch {
		case r.Method == http.MethodPost:
			s.initLine = r.Method + " " + r.URL.RequestURI()
			raw, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(raw, &s.initBody); err != nil {
				t.Errorf("init body %q is not JSON: %v", raw, err)
			}
			_, _ = w.Write([]byte(`{"estimatedTimeForStatusCheck":` + s.estimate + `,"reportId":"` + s.reportID + `"}`))
		case strings.Contains(r.URL.Path, "/reportStatus/"):
			status := s.statuses[0]
			if len(s.statuses) > 1 {
				s.statuses = s.statuses[1:]
			}
			_, _ = w.Write([]byte(`{"progress":{"status":"` + status + `","responseMessage":null},"reportId":"x"}`))
		default:
			s.accepts = append(s.accepts, r.Header.Get("Accept"))
			w.Header().Set("Content-Type", "application/pdf")
			_, _ = w.Write([]byte("%PDF-1.4 body"))
		}
	})
}

func recordWaits(client *Client, waits *[]time.Duration) {
	WithReportWait(func(_ context.Context, d time.Duration) error {
		*waits = append(*waits, d)
		return nil
	})(client)
}

func TestExportReportBloodPressure(t *testing.T) {
	srv := &reportServer{statuses: []string{"CREATING", "READY"}, estimate: "5000", reportID: "id/1+2="}
	_, client := testServer(t, srv.handler(t))
	var waits []time.Duration
	recordWaits(client, &waits)

	got, err := client.ExportReport(context.Background(),
		BloodPressureReport("2026-10-27", "2026-10-27", PeriodDaily, ReportPDF, ReportPreferences{Locale: "de", DateFormatKey: "ddmmyyyy"}))
	if err != nil || string(got) != "%PDF-1.4 body" {
		t.Fatalf("ExportReport = %q, %v", got, err)
	}

	if want := "POST /reportexport-service/export/init/bloodPressure/2026-10-27/2026-10-27?locale=de"; srv.initLine != want {
		t.Errorf("init request = %q, want %q", srv.initLine, want)
	}
	wantBody := map[string]any{"contentToSkip": []any{}, "reportSubCategory": "Daily", "reportFormat": "PDF", "requestedDateFormatKey": "ddmmyyyy"}
	if !jsonEqual(srv.initBody, wantBody) {
		t.Errorf("init body = %v, want %v", srv.initBody, wantBody)
	}
	wantPaths := []string{
		"POST /reportexport-service/export/init/bloodPressure/2026-10-27/2026-10-27",
		"GET /reportexport-service/export/reportStatus/id%2F1+2=",
		"GET /reportexport-service/export/reportStatus/id%2F1+2=",
		"GET /reportexport-service/export/reportContent/id%2F1+2=",
	}
	if !slices.Equal(srv.paths, wantPaths) {
		t.Errorf("requests = %q, want %q", srv.paths, wantPaths)
	}
	if want := []time.Duration{5 * time.Second, 5 * time.Second}; !slices.Equal(waits, want) {
		t.Errorf("waits = %v, want %v", waits, want)
	}
	if len(srv.accepts) != 1 || srv.accepts[0] == "application/json" {
		t.Errorf("content request Accept = %q: Garmin answers 406 to application/json", srv.accepts)
	}
}

func TestExportReportHealthSnapshot(t *testing.T) {
	srv := &reportServer{statuses: []string{"READY"}, estimate: "3000", reportID: "r"}
	_, client := testServer(t, srv.handler(t))
	recordWaits(client, new([]time.Duration))

	if _, err := client.ExportReport(context.Background(), HealthSnapshotReport("2026-10-01", "uuid-1", ReportPreferences{Locale: "en"})); err != nil {
		t.Fatal(err)
	}
	if want := "POST /reportexport-service/export/init/wellnessactivity/2026-10-01/2026-10-01?locale=en"; srv.initLine != want {
		t.Errorf("init request = %q, want %q", srv.initLine, want)
	}
	if !jsonEqual(srv.initBody, map[string]any{"supplementalId": "uuid-1"}) {
		t.Errorf("init body = %v", srv.initBody)
	}
}

func TestExportReportPollInterval(t *testing.T) {
	tests := []struct {
		estimate string
		want     time.Duration
	}{
		{"0", 3 * time.Second},
		{"null", 3 * time.Second},
		{"100", time.Second},
		{"4000", 4 * time.Second},
		{"60000", 10 * time.Second},
	}
	for _, tc := range tests {
		t.Run(tc.estimate, func(t *testing.T) {
			srv := &reportServer{statuses: []string{"READY"}, estimate: tc.estimate, reportID: "r"}
			_, client := testServer(t, srv.handler(t))
			var waits []time.Duration
			recordWaits(client, &waits)
			if _, err := client.ExportReport(context.Background(), HealthSnapshotReport("2026-10-01", "u", ReportPreferences{Locale: "en"})); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(waits, []time.Duration{tc.want}) {
				t.Errorf("waits = %v, want [%v]", waits, tc.want)
			}
		})
	}
}

func TestExportReportEndsWithoutContentOnNotReady(t *testing.T) {
	for _, status := range []string{"DATA_NOT_FOUND", "ERROR", "OFF"} {
		t.Run(status, func(t *testing.T) {
			srv := &reportServer{statuses: []string{status}, estimate: "3000", reportID: "r"}
			_, client := testServer(t, srv.handler(t))
			recordWaits(client, new([]time.Duration))

			_, err := client.ExportReport(context.Background(), HealthSnapshotReport("2026-10-01", "u", ReportPreferences{Locale: "en"}))
			var unavailable *ReportUnavailableError
			if !errors.As(err, &unavailable) || unavailable.Status != status {
				t.Fatalf("err = %v, want ReportUnavailableError with status %s", err, status)
			}
			if len(srv.accepts) != 0 {
				t.Errorf("content was requested after status %s", status)
			}
		})
	}
}

func TestExportReportGivesUpWhileStillCreating(t *testing.T) {
	srv := &reportServer{statuses: []string{"CREATING"}, estimate: "5000", reportID: "r"}
	_, client := testServer(t, srv.handler(t))
	var waits []time.Duration
	recordWaits(client, &waits)

	_, err := client.ExportReport(context.Background(), HealthSnapshotReport("2026-10-01", "u", ReportPreferences{Locale: "en"}))
	var timeout *ReportTimeoutError
	if !errors.As(err, &timeout) {
		t.Fatalf("err = %v, want ReportTimeoutError", err)
	}
	if len(waits) != 24 || timeout.Waited != 2*time.Minute {
		t.Errorf("waited %d times for %v, want 24 times for 2m", len(waits), timeout.Waited)
	}
	if len(srv.accepts) != 0 {
		t.Error("content was requested for a report that never became READY")
	}
}

func TestExportReportStopsWhenTheWaitIsCancelled(t *testing.T) {
	srv := &reportServer{statuses: []string{"READY"}, estimate: "3000", reportID: "r"}
	_, client := testServer(t, srv.handler(t))
	WithReportWait(func(context.Context, time.Duration) error { return context.Canceled })(client)

	_, err := client.ExportReport(context.Background(), HealthSnapshotReport("2026-10-01", "u", ReportPreferences{Locale: "en"}))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if len(srv.paths) != 1 {
		t.Errorf("requests after the cancel: %q", srv.paths)
	}
}

func TestExportReportInitWithoutReportID(t *testing.T) {
	_, client := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"estimatedTimeForStatusCheck":3000}`))
	}))
	recordWaits(client, new([]time.Duration))

	if _, err := client.ExportReport(context.Background(), HealthSnapshotReport("2026-10-01", "u", ReportPreferences{Locale: "en"})); err == nil {
		t.Fatal("ExportReport succeeded without a reportId")
	}
}

func jsonEqual(a, b any) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}

func TestGetReportPreferences(t *testing.T) {
	tests := []struct {
		name    string
		reply   string
		want    ReportPreferences
		wantErr bool
	}{
		{"both set", `{"preferredLocale":"de","dateFormat":{"formatKey":"ddmmyyyy"}}`, ReportPreferences{Locale: "de", DateFormatKey: "ddmmyyyy"}, false},
		{"no locale falls back to en", `{"dateFormat":{"formatKey":"mmddyyyy"}}`, ReportPreferences{Locale: "en", DateFormatKey: "mmddyyyy"}, false},
		{"no date format key", `{"preferredLocale":"de"}`, ReportPreferences{}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, client := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/userprofile-service/userprofile/settings" {
					t.Errorf("path = %s", r.URL.Path)
				}
				_, _ = w.Write([]byte(tc.reply))
			}))
			got, err := client.GetReportPreferences(context.Background())
			if (err != nil) != tc.wantErr || got != tc.want {
				t.Errorf("GetReportPreferences = %+v, %v; want %+v, error %v", got, err, tc.want, tc.wantErr)
			}
		})
	}
}
