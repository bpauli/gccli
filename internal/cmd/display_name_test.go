package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bpauli/gccli/internal/outfmt"
)

func serveDisplayName(mux *http.ServeMux) {
	mux.HandleFunc("/userprofile-service/userprofile/settings", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"displayName":"Test User"}`))
	})
}

var displayNameCommands = []struct {
	name     string
	cmd      interface{ Run(*Globals) error }
	wantPath string
}{
	{"health summary", &HealthSummaryCmd{}, "/usersummary-service/usersummary/daily/Test User"},
	{"health steps", &HealthStepsViewCmd{}, "/wellness-service/wellness/dailySummaryChart/Test User"},
	{"health hr", &HealthHRCmd{}, "/wellness-service/wellness/dailyHeartRate/Test User"},
	{"health rhr", &HealthRHRCmd{}, "/userstats-service/wellness/daily/Test User"},
	{"health sleep", &HealthSleepCmd{}, "/wellness-service/wellness/dailySleepData/Test User"},
}

func TestDisplayName_ResolvedFromProfile(t *testing.T) {
	for _, tc := range displayNameCommands {
		t.Run(tc.name, func(t *testing.T) {
			var dataPaths []string
			mux := http.NewServeMux()
			serveDisplayName(mux)
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				dataPaths = append(dataPaths, r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{}`))
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			store := newTestSecretsStore(t)
			overrideLoadSecrets(t, store)
			overrideNewClient(t, server)
			storeTestTokens(t, store, "test@example.com", testTokens())

			var buf bytes.Buffer
			g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
			if err := tc.cmd.Run(g); err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if len(dataPaths) != 1 || dataPaths[0] != tc.wantPath {
				t.Errorf("data requests = %q, want exactly [%q]", dataPaths, tc.wantPath)
			}
		})
	}
}

func TestDisplayName_ProfileErrorStopsCommand(t *testing.T) {
	for _, tc := range displayNameCommands {
		t.Run(tc.name, func(t *testing.T) {
			var dataPaths []string
			mux := http.NewServeMux()
			mux.HandleFunc("/userprofile-service/userprofile/settings", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			})
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				dataPaths = append(dataPaths, r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{}`))
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			store := newTestSecretsStore(t)
			overrideLoadSecrets(t, store)
			overrideNewClient(t, server)
			storeTestTokens(t, store, "test@example.com", testTokens())

			var buf bytes.Buffer
			g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
			if err := tc.cmd.Run(g); err == nil || !strings.HasPrefix(err.Error(), "resolve display name: ") {
				t.Fatalf("err = %v, want prefix %q", err, "resolve display name: ")
			}
			if len(dataPaths) != 0 {
				t.Errorf("data requests = %q, want none", dataPaths)
			}
		})
	}
}
