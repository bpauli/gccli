package garminapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
)

// --- GetDevices tests ---

func TestGetDevices_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/device-service/deviceregistration/devices" {
			t.Errorf("path = %s, want /device-service/deviceregistration/devices", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"deviceId":12345,"displayName":"Forerunner 265"}]`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetDevices(context.Background())
	if err != nil {
		t.Fatalf("GetDevices: %v", err)
	}

	var result []map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 device, got %d", len(result))
	}
	if result[0]["displayName"] != "Forerunner 265" {
		t.Errorf("displayName = %v, want Forerunner 265", result[0]["displayName"])
	}
}

func TestGetDevices_ServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})

	_, client := testServer(t, handler)
	_, err := client.GetDevices(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *GarminAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected GarminAPIError, got %T: %v", err, err)
	}
}

// --- GetDeviceSettings tests ---

func TestGetDeviceSettings_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/device-service/deviceservice/device-info/settings/12345" {
			t.Errorf("path = %s, want /device-service/deviceservice/device-info/settings/12345", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"alarms":[{"time":"07:00"}],"deviceId":12345}`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetDeviceSettings(context.Background(), "12345")
	if err != nil {
		t.Fatalf("GetDeviceSettings: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["deviceId"] == nil {
		t.Error("expected deviceId field")
	}
}

func TestGetDeviceSettings_NotFound(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	})

	_, client := testServer(t, handler)
	_, err := client.GetDeviceSettings(context.Background(), "99999")
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *GarminAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected GarminAPIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", apiErr.StatusCode, http.StatusNotFound)
	}
}

// --- GetPrimaryTrainingDevice tests ---

func TestGetPrimaryTrainingDevice_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/web-gateway/device-info/primary-training-device" {
			t.Errorf("path = %s, want /web-gateway/device-info/primary-training-device", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"primaryTrainingDevice":{"deviceId":12345},"devicePriority":[{"deviceId":12345}]}`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetPrimaryTrainingDevice(context.Background())
	if err != nil {
		t.Fatalf("GetPrimaryTrainingDevice: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["primaryTrainingDevice"] == nil {
		t.Error("expected primaryTrainingDevice field")
	}
}

// --- GetDeviceSolar tests ---

func TestGetDeviceSolar_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/web-gateway/solar/12345/2024-01-01/2024-01-01" {
			t.Errorf("path = %s, want /web-gateway/solar/12345/2024-01-01/2024-01-01", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"date":"2024-01-01","solarIntensity":85}]`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetDeviceSolar(context.Background(), "12345", "2024-01-01", "2024-01-01")
	if err != nil {
		t.Fatalf("GetDeviceSolar: %v", err)
	}

	var result []map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result))
	}
}

func TestGetDeviceSolar_DateRange(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/web-gateway/solar/12345/2024-01-01/2024-01-07" {
			t.Errorf("path = %s, want /web-gateway/solar/12345/2024-01-01/2024-01-07", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"date":"2024-01-01"},{"date":"2024-01-02"}]`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetDeviceSolar(context.Background(), "12345", "2024-01-01", "2024-01-07")
	if err != nil {
		t.Fatalf("GetDeviceSolar: %v", err)
	}

	var result []map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}
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

// --- GetLastUsedDevice tests ---

func TestGetLastUsedDevice_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/device-service/deviceservice/mylastused" {
			t.Errorf("path = %s, want /device-service/deviceservice/mylastused", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"deviceId":12345,"deviceTypePk":123,"lastUsedDeviceApplicationKey":"abc","lastUsedDeviceName":"Forerunner 265"}`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetLastUsedDevice(context.Background())
	if err != nil {
		t.Fatalf("GetLastUsedDevice: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["lastUsedDeviceName"] != "Forerunner 265" {
		t.Errorf("lastUsedDeviceName = %v, want Forerunner 265", result["lastUsedDeviceName"])
	}
}

func TestGetLastUsedDevice_ServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})

	_, client := testServer(t, handler)
	_, err := client.GetLastUsedDevice(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *GarminAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected GarminAPIError, got %T: %v", err, err)
	}
}
