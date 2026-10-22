package garminapi

import (
	"context"
	"encoding/json"
	"fmt"
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
		{"devices.backups", nil, "GET /devicebackup-service/backup"},
		{"devices.set-primary-tracker", map[string]string{"deviceID": "12345"}, "PUT /device-service/deviceservice/device-info/12345/primary-activity-tracker"},
		{"devices.historical", nil, "GET /device-service/deviceregistration/devices/historical"},
		{"devices.messages", nil, "GET /device-service/devicemessage/messages"},
		{"devices.send-message", nil, "POST /device-service/devicemessage/messages"},
		{"devices.cancel-message", map[string]string{"messageID": "777"}, "DELETE /device-service/devicemessage/message/777"},
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
	if string(data) != "[]" {
		t.Fatalf("alarms = %s, want [] (an empty list, not null)", data)
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
	if string(data) != "[]" {
		t.Fatalf("alarms = %s, want [] (an empty list, not null)", data)
	}
}

// An empty devices body (a 204 or an empty 200) is an empty list, not an
// error.
func TestGetDeviceAlarms_EmptyDevicesBody(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusNoContent} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/device-service/deviceregistration/devices" {
					t.Errorf("unexpected request %s", r.URL.Path)
				}
				w.WriteHeader(status)
			})

			_, client := testServer(t, handler)
			data, err := client.GetDeviceAlarms(context.Background())
			if err != nil {
				t.Fatalf("GetDeviceAlarms: %v", err)
			}
			if string(data) != "[]" {
				t.Fatalf("alarms = %s, want []", data)
			}
		})
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

func TestDeviceCanReceive(t *testing.T) {
	workout := func(raw string) Workout {
		var w Workout
		if err := json.Unmarshal([]byte(raw), &w); err != nil {
			t.Fatal(err)
		}
		return w
	}
	run := workout(`{"sportType":{"sportTypeKey":"running"}}`)
	brick := workout(`{"sportType":{"sportTypeKey":"multi_sport"},"workoutSegments":[{"sportType":{"sportTypeKey":"cycling"}},{"sportType":{"sportTypeKey":"running"}}]}`)
	unknown := workout(`{"sportType":{"sportTypeKey":"new_sport"}}`)

	tests := []struct {
		name    string
		device  string
		workout Workout
		want    bool
	}{
		{"workout and sport capable", `{"workoutCapable":true,"runningWorkoutCapable":true}`, run, true},
		{"not workout capable", `{"workoutCapable":false,"runningWorkoutCapable":true}`, run, false},
		{"sport flag false", `{"workoutCapable":true,"runningWorkoutCapable":false}`, run, false},
		{"sport flag missing", `{"workoutCapable":true}`, run, false},
		{"sport without a flag needs only workoutCapable", `{"workoutCapable":true}`, unknown, true},
		{"multisport with every segment flag", `{"workoutCapable":true,"multisportWorkoutCapable":true,"cyclingWorkoutCapable":true,"runningWorkoutCapable":true}`, brick, true},
		{"multisport without a segment flag", `{"workoutCapable":true,"multisportWorkoutCapable":true,"cyclingWorkoutCapable":true}`, brick, false},
		{"multisport without the multisport flag", `{"workoutCapable":true,"cyclingWorkoutCapable":true,"runningWorkoutCapable":true}`, brick, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d Device
			if err := json.Unmarshal([]byte(tt.device), &d); err != nil {
				t.Fatal(err)
			}
			if got := d.CanReceive(tt.workout); got != tt.want {
				t.Errorf("CanReceive = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeviceUnmarshal(t *testing.T) {
	var d Device
	if err := json.Unmarshal([]byte(`{"deviceId":3632535288,"displayName":"Forerunner 965","workoutFileType":null,"workoutCapable":true}`), &d); err != nil {
		t.Fatal(err)
	}
	if d.ID.String() != "3632535288" || d.Name != "Forerunner 965" || d.WorkoutFileType != "" {
		t.Errorf("Device = %+v", d)
	}
}
