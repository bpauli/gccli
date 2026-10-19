package garminapi

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"
)

func TestDevicesEndpoints(t *testing.T) {
	testEndpoints(t, devicesEndpoints, []endpointCase{
		{"devices.list", nil, "GET /device-service/deviceregistration/devices"},
		{"devices.settings", map[string]string{"deviceID": "12345"}, "GET /device-service/deviceservice/device-info/settings/12345"},
		{"devices.primary-training", nil, "GET /web-gateway/device-info/primary-training-device"},
		{"devices.solar", map[string]string{"deviceID": "12345", "startDate": "2024-01-01", "endDate": "2024-01-07"}, "GET /web-gateway/solar/12345/2024-01-01/2024-01-07"},
		{"devices.last-used", nil, "GET /device-service/deviceservice/mylastused"},
		{"devices.sensors", nil, "GET /device-service/sensors"},
		{"devices.backup-summary", nil, "GET /devicebackup-service/backup/summary"},
	})
}

// --- GetDeviceAlarms tests ---

func TestGetDeviceAlarms_Success(t *testing.T) {
	callCount := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/device-service/deviceregistration/devices":
			_, _ = w.Write([]byte(`[{"deviceId":111},{"deviceId":222}]`))
		case strings.HasPrefix(r.URL.Path, "/device-service/deviceservice/device-info/settings/111"):
			callCount++
			_, _ = w.Write([]byte(`{"alarms":[{"time":"07:00"},{"time":"08:00"}]}`))
		case strings.HasPrefix(r.URL.Path, "/device-service/deviceservice/device-info/settings/222"):
			callCount++
			_, _ = w.Write([]byte(`{"alarms":[{"time":"06:30"}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	_, client := testServer(t, handler)
	data, err := client.GetDeviceAlarms(context.Background())
	if err != nil {
		t.Fatalf("GetDeviceAlarms: %v", err)
	}

	var result []any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 alarms, got %d", len(result))
	}
	if callCount != 2 {
		t.Errorf("expected 2 settings calls, got %d", callCount)
	}
}

func TestGetDeviceAlarms_TenDigitDeviceID(t *testing.T) {
	var settingsPaths []string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/device-service/deviceregistration/devices":
			_, _ = w.Write([]byte(`[{"deviceId":3442032430}]`))
		case "/device-service/deviceservice/device-info/settings/3442032430":
			settingsPaths = append(settingsPaths, r.URL.Path)
			_, _ = w.Write([]byte(`{"alarms":[{"time":"07:00"}]}`))
		default:
			settingsPaths = append(settingsPaths, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})

	_, client := testServer(t, handler)
	data, err := client.GetDeviceAlarms(context.Background())
	if err != nil {
		t.Fatalf("GetDeviceAlarms: %v", err)
	}

	want := []string{"/device-service/deviceservice/device-info/settings/3442032430"}
	if !slices.Equal(settingsPaths, want) {
		t.Errorf("settings paths = %q, want %q", settingsPaths, want)
	}

	var result []any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 alarm, got %d", len(result))
	}
}

func TestGetDeviceAlarms_NoDevices(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/device-service/deviceregistration/devices" {
			_, _ = w.Write([]byte(`[]`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	_, client := testServer(t, handler)
	data, err := client.GetDeviceAlarms(context.Background())
	if err != nil {
		t.Fatalf("GetDeviceAlarms: %v", err)
	}

	var result []any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0 alarms, got %d", len(result))
	}
}

func TestGetDeviceAlarms_NoAlarms(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/device-service/deviceregistration/devices" {
			_, _ = w.Write([]byte(`[{"deviceId":111}]`))
			return
		}
		_, _ = w.Write([]byte(`{"someOtherSetting":"value"}`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetDeviceAlarms(context.Background())
	if err != nil {
		t.Fatalf("GetDeviceAlarms: %v", err)
	}

	var result []any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0 alarms, got %d", len(result))
	}
}

func TestGetDeviceAlarms_SettingsError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/device-service/deviceregistration/devices" {
			_, _ = w.Write([]byte(`[{"deviceId":111}]`))
			return
		}
		// Settings call fails — alarms should still return empty, not error.
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})

	_, client := testServer(t, handler)
	data, err := client.GetDeviceAlarms(context.Background())
	if err != nil {
		t.Fatalf("GetDeviceAlarms: %v", err)
	}

	var result []any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0 alarms (settings failed), got %d", len(result))
	}
}
