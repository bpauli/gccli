package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bpauli/gccli/internal/outfmt"
)

func devicesTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/device-service/deviceregistration/devices", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"deviceId":12345,"displayName":"Forerunner 265","deviceTypePk":123}]`))
	})

	mux.HandleFunc("/device-service/deviceservice/device-info/settings/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"alarms":[{"time":"07:00"}],"deviceId":12345}`))
	})

	mux.HandleFunc("/web-gateway/device-info/primary-training-device", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"primaryTrainingDevice":{"deviceId":12345}}`))
	})

	mux.HandleFunc("/web-gateway/solar/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"date":"2024-01-01","solarIntensity":85}]`))
	})

	mux.HandleFunc("/device-service/deviceservice/mylastused", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"deviceId":12345,"lastUsedDeviceName":"Forerunner 265"}`))
	})

	return httptest.NewServer(mux)
}

// --- DeviceAlarmsCmd tests ---

func TestDeviceAlarms_Success(t *testing.T) {
	server := devicesTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &DeviceAlarmsCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}
