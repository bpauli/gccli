package cmd

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/bpauli/gccli/internal/config"
	"github.com/bpauli/gccli/internal/outfmt"
	"github.com/bpauli/gccli/internal/ui"
	"github.com/bpauli/gccli/internal/units"
)

const userSettingsPath = "/userprofile-service/userprofile/user-settings"

func unitsTestServer(t *testing.T, settings http.HandlerFunc) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/activitylist-service/activities/search/activities", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(sampleActivitiesJSON()))
	})
	if settings != nil {
		mux.HandleFunc(userSettingsPath, settings)
	}
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())
}

func serveMeasurementSystem(value string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"userData":{"measurementSystem":"` + value + `"}}`))
	}
}

func failOnSettingsRequest(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		t.Errorf("unexpected user settings request")
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	done := make(chan string)
	go func() {
		data, _ := io.ReadAll(r)
		done <- string(data)
	}()
	fn()
	_ = w.Close()
	return <-done
}

func runActivitiesList(t *testing.T, mode outfmt.Mode, pref units.Preference) (stdout, stderr string) {
	t.Helper()
	var buf bytes.Buffer
	g := testGlobals(t, &buf, mode, "test@example.com")
	g.Units = pref
	stdout = captureStdout(t, func() {
		if err := (&ActivitiesListCmd{Limit: 20}).Run(g); err != nil {
			t.Fatalf("Run: %v", err)
		}
	})
	return stdout, buf.String()
}

func TestActivitiesList_AutoUsesGarminProfile(t *testing.T) {
	unitsTestServer(t, serveMeasurementSystem("statute_us"))

	stdout, stderr := runActivitiesList(t, outfmt.Plain, units.Auto)
	if !strings.Contains(stdout, "3.18 mi") {
		t.Errorf("stdout missing statute distance:\n%s", stdout)
	}
	if strings.Contains(stdout, " km") {
		t.Errorf("stdout still has km:\n%s", stdout)
	}
	if stderr != "" {
		t.Errorf("unexpected stderr: %q", stderr)
	}
}

func TestActivitiesList_JSONSkipsUnitsLookup(t *testing.T) {
	unitsTestServer(t, failOnSettingsRequest(t))

	stdout, _ := runActivitiesList(t, outfmt.JSON, units.Auto)
	if !strings.Contains(stdout, `"distance": 5123.45`) {
		t.Errorf("JSON output changed:\n%s", stdout)
	}
}

func TestActivitiesList_PinnedSkipsUnitsLookup(t *testing.T) {
	unitsTestServer(t, failOnSettingsRequest(t))

	stdout, _ := runActivitiesList(t, outfmt.Plain, units.Pin(units.StatuteUK))
	if !strings.Contains(stdout, "3.18 mi") {
		t.Errorf("stdout missing pinned statute distance:\n%s", stdout)
	}
}

func TestActivitiesList_UnitsLookupFallsBackToMetric(t *testing.T) {
	tests := []struct {
		name     string
		settings http.HandlerFunc
	}{
		{"not found", nil},
		{"unknown value", serveMeasurementSystem("furlongs")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unitsTestServer(t, tt.settings)

			stdout, stderr := runActivitiesList(t, outfmt.Plain, units.Auto)
			if !strings.Contains(stdout, "5.12 km") {
				t.Errorf("stdout missing metric distance:\n%s", stdout)
			}
			if !strings.Contains(stderr, "could not read Garmin unit preference") ||
				!strings.Contains(stderr, "using metric") {
				t.Errorf("stderr missing fallback warning: %q", stderr)
			}
		})
	}
}

func TestUnitsPreference(t *testing.T) {
	tests := []struct {
		name     string
		flag     string
		cfgUnits string
		want     units.Preference
		wantWarn bool
	}{
		{"default", "auto", "", units.Auto, false},
		{"flag pins", "statute_us", "", units.Pin(units.StatuteUS), false},
		{"flag beats config", "metric", "statute_uk", units.Pin(units.Metric), false},
		{"auto flag falls through to config", "auto", "statute_uk", units.Pin(units.StatuteUK), false},
		{"config auto", "auto", "auto", units.Auto, false},
		{"invalid config warns", "auto", "imperial", units.Auto, true},
		{"flag hides invalid config", "metric", "imperial", units.Pin(units.Metric), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			u := ui.NewWithWriter(&buf, "never")
			got := unitsPreference(tt.flag, &config.File{Units: tt.cfgUnits}, u)
			if got != tt.want {
				t.Errorf("unitsPreference(%q, %q) = %v, want %v", tt.flag, tt.cfgUnits, got, tt.want)
			}
			if warned := strings.Contains(buf.String(), "units"); warned != tt.wantWarn {
				t.Errorf("warning = %q, wantWarn %v", buf.String(), tt.wantWarn)
			}
		})
	}
}
