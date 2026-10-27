package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

type reportFlow struct {
	mu       sync.Mutex
	status   string
	initCode int
	requests []string
	initBody map[string]any
}

func (f *reportFlow) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, r.Method+" "+r.URL.RequestURI())
	switch {
	case r.URL.Path == "/userprofile-service/userprofile/settings":
		_, _ = w.Write([]byte(`{"preferredLocale":"de","dateFormat":{"formatKey":"ddmmyyyy"}}`))
	case r.Method == http.MethodPost:
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &f.initBody)
		if f.initCode != 0 {
			w.WriteHeader(f.initCode)
			return
		}
		_, _ = w.Write([]byte(`{"estimatedTimeForStatusCheck":3000,"reportId":"rep=="}`))
	case strings.Contains(r.URL.Path, "/reportStatus/"):
		_, _ = w.Write([]byte(`{"progress":{"status":"` + f.status + `","responseMessage":null}}`))
	default:
		_, _ = w.Write([]byte("REPORT-BYTES"))
	}
}

func (f *reportFlow) initRequest() string {
	for _, r := range f.requests {
		if strings.HasPrefix(r, "POST ") {
			return r
		}
	}
	return ""
}

func (f *reportFlow) sent() int {
	n := 0
	for _, r := range f.requests {
		if strings.Contains(r, "reportexport-service") {
			n++
		}
	}
	return n
}

func runReportCommand(t *testing.T, flow *reportFlow, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	server := httptest.NewServer(flow)
	t.Cleanup(server.Close)
	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	storeTestTokens(t, store, "test@example.com", testTokens())
	overrideNewClient(t, server)

	args = append([]string{"--color", "never", "--account", "test@example.com"}, args...)
	stdout, stderr = captureOutput(t, func() {
		code = Execute(args, "1.0.0", "abc123", "2024-01-01")
	})
	return code, stdout, stderr
}

func TestBodyExportBloodPressureSavesTheReport(t *testing.T) {
	flow := &reportFlow{status: "READY"}
	out := filepath.Join(t.TempDir(), "bp.pdf")

	code, stdout, stderr := runReportCommand(t, flow,
		"body", "export-blood-pressure", "--period", "seven-days", "--end", "2026-10-27", "-o", out)

	if code != 0 || stdout != "" {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, stdout, stderr)
	}
	if got, _ := os.ReadFile(out); string(got) != "REPORT-BYTES" {
		t.Errorf("file = %q", got)
	}
	if want := "Downloaded " + out + " (12 bytes)"; !strings.Contains(stderr, want) {
		t.Errorf("stderr = %q, want %q", stderr, want)
	}
	if want := "POST /reportexport-service/export/init/bloodPressure/2026-10-21/2026-10-27?locale=de"; flow.initRequest() != want {
		t.Errorf("init = %q, want %q", flow.initRequest(), want)
	}
	want := `{"contentToSkip":[],"reportFormat":"PDF","reportSubCategory":"SevenDays","requestedDateFormatKey":"ddmmyyyy"}`
	if got, _ := json.Marshal(flow.initBody); string(got) != want {
		t.Errorf("init body = %s, want %s", got, want)
	}
}

func TestBodyExportBloodPressureDefaultsToTodayAndNamesTheFile(t *testing.T) {
	flow := &reportFlow{status: "READY"}
	dir := t.TempDir()
	t.Chdir(dir)

	code, _, stderr := runReportCommand(t, flow, "body", "export-blood-pressure", "--end", "2026-10-27")

	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "blood_pressure_daily_2026-10-27.pdf")); string(got) != "REPORT-BYTES" {
		t.Errorf("default file = %q", got)
	}
	if flow.initBody["reportSubCategory"] != "Daily" {
		t.Errorf("reportSubCategory = %v, want Daily", flow.initBody["reportSubCategory"])
	}
}

func TestBodyExportBloodPressureRange(t *testing.T) {
	tests := []struct {
		period, sub, start string
	}{
		{"daily", "Daily", "2026-10-27"},
		{"seven-days", "SevenDays", "2026-10-21"},
		{"four-weeks", "FourWeeks", "2026-09-30"},
		{"yearly", "Yearly", "2025-10-29"},
	}
	for _, tc := range tests {
		t.Run(tc.period, func(t *testing.T) {
			flow := &reportFlow{status: "READY"}
			out := filepath.Join(t.TempDir(), "x")
			if code, _, stderr := runReportCommand(t, flow, "body", "export-blood-pressure", "--period", tc.period, "--end", "2026-10-27", "-o", out); code != 0 {
				t.Fatalf("exit %d, stderr %q", code, stderr)
			}
			if want := "POST /reportexport-service/export/init/bloodPressure/" + tc.start + "/2026-10-27?locale=de"; flow.initRequest() != want {
				t.Errorf("init = %q, want %q", flow.initRequest(), want)
			}
			if flow.initBody["reportSubCategory"] != tc.sub {
				t.Errorf("reportSubCategory = %v, want %s", flow.initBody["reportSubCategory"], tc.sub)
			}
		})
	}
}

func TestBodyExportBloodPressureWithoutDataKeepsTheOldFile(t *testing.T) {
	flow := &reportFlow{status: "DATA_NOT_FOUND"}
	out := filepath.Join(t.TempDir(), "bp.pdf")
	if err := os.WriteFile(out, []byte("OLD"), 0o644); err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := runReportCommand(t, flow,
		"body", "export-blood-pressure", "--end", "2020-01-01", "-o", out)

	if code != 1 || stdout != "" {
		t.Fatalf("exit %d, stdout %q, want 1 and no stdout", code, stdout)
	}
	if want := "export blood pressure report: Garmin has no data for this report (status DATA_NOT_FOUND)"; !strings.Contains(stderr, want) {
		t.Errorf("stderr = %q, want %q", stderr, want)
	}
	if got, _ := os.ReadFile(out); string(got) != "OLD" {
		t.Errorf("the failed export changed the output file to %q", got)
	}
}

func TestBodyExportBloodPressureJSONError(t *testing.T) {
	flow := &reportFlow{status: "DATA_NOT_FOUND"}

	code, stdout, stderr := runReportCommand(t, flow, "--json",
		"body", "export-blood-pressure", "--end", "2020-01-01", "-o", filepath.Join(t.TempDir(), "x"))

	want := `{"error":{"code":"error","message":"export blood pressure report: Garmin has no data for this report (status DATA_NOT_FOUND)"}}`
	if code != 1 || stdout != "" || strings.TrimSpace(stderr) != want {
		t.Errorf("exit %d, stdout %q, stderr %q, want exit 1 and %s", code, stdout, stderr, want)
	}
}

func TestBodyExportBloodPressureRateLimited(t *testing.T) {
	flow := &reportFlow{initCode: http.StatusTooManyRequests}

	code, _, stderr := runReportCommand(t, flow,
		"body", "export-blood-pressure", "-o", filepath.Join(t.TempDir(), "x"))

	if code != 5 {
		t.Errorf("exit %d, want 5; stderr %q", code, stderr)
	}
}

func TestBodyExportBloodPressureUsageErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"unknown period", []string{"--period", "weekly"}, "--period must be one of"},
		{"no format flag, Garmin only makes PDF", []string{"--format", "csv"}, "unknown flag --format"},
		{"no start flag, the period sets the range", []string{"--start", "2026-10-01"}, "unknown flag --start"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			flow := &reportFlow{status: "READY"}
			code, _, stderr := runReportCommand(t, flow, append([]string{"body", "export-blood-pressure"}, tc.args...)...)
			if code != 2 || !strings.Contains(stderr, tc.want) {
				t.Errorf("exit %d, stderr %q, want exit 2 and %q", code, stderr, tc.want)
			}
			if flow.sent() != 0 {
				t.Errorf("a usage error sent %d report requests", flow.sent())
			}
		})
	}
}

func TestHealthSnapshotsReport(t *testing.T) {
	flow := &reportFlow{status: "READY"}
	dir := t.TempDir()
	t.Chdir(dir)

	code, stdout, stderr := runReportCommand(t, flow, "health", "snapshots", "report", "2026-10-01", "uuid-1")

	if code != 0 || stdout != "" {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, stdout, stderr)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "health_snapshot_uuid-1.pdf")); string(got) != "REPORT-BYTES" {
		t.Errorf("file = %q", got)
	}
	if want := "POST /reportexport-service/export/init/wellnessactivity/2026-10-01/2026-10-01?locale=de"; flow.initRequest() != want {
		t.Errorf("init = %q, want %q", flow.initRequest(), want)
	}
	if got, _ := json.Marshal(flow.initBody); string(got) != `{"supplementalId":"uuid-1"}` {
		t.Errorf("init body = %s", got)
	}
}

func TestHealthSnapshotsReportWithoutData(t *testing.T) {
	flow := &reportFlow{status: "DATA_NOT_FOUND"}
	dir := t.TempDir()
	t.Chdir(dir)

	code, _, stderr := runReportCommand(t, flow, "health", "snapshots", "report", "2026-10-01", "uuid-1")

	if want := "export health snapshot report: Garmin has no data for this report"; code != 1 || !strings.Contains(stderr, want) {
		t.Errorf("exit %d, stderr %q, want exit 1 and %q", code, stderr, want)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("a failed export left %d files", len(entries))
	}
}
