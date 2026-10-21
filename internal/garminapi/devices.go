package garminapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

var devicesEndpoints = []Endpoint{
	{Name: "devices.list", Method: http.MethodGet, Host: ConnectAPI, Path: "/device-service/deviceregistration/devices"},
	{Name: "devices.settings", Method: http.MethodGet, Host: ConnectAPI, Path: "/device-service/deviceservice/device-info/settings/{deviceID}"},
	{Name: "devices.primary-training", Method: http.MethodGet, Host: ConnectAPI, Path: "/web-gateway/device-info/primary-training-device"},
	{Name: "devices.solar", Method: http.MethodGet, Host: ConnectAPI, Path: "/web-gateway/solar/{deviceID}/{startDate}/{endDate}"},
	{Name: "devices.last-used", Method: http.MethodGet, Host: ConnectAPI, Path: "/device-service/deviceservice/mylastused"},
	{Name: "devices.sensors", Method: http.MethodGet, Host: ConnectAPI, Path: "/device-service/sensors"},
	{Name: "devices.backup-summary", Method: http.MethodGet, Host: ConnectAPI, Path: "/devicebackup-service/backup/summary"},
	{Name: "devices.historical", Method: http.MethodGet, Host: ConnectAPI, Path: "/device-service/deviceregistration/devices/historical"},
	{Name: "devices.messages", Method: http.MethodGet, Host: ConnectAPI, Path: "/device-service/devicemessage/messages"},
	{Name: "devices.send-message", Method: http.MethodPost, Host: ConnectAPI, Path: "/device-service/devicemessage/messages", Body: true},
	{Name: "devices.cancel-message", Method: http.MethodDelete, Host: ConnectAPI, Path: "/device-service/devicemessage/message/{messageID}"},
}

// GetDeviceAlarms returns the alarms of all devices in one list, from the
// settings of each registered device. An empty devices body is an empty
// list, and so is a device without alarms.
func (c *Client) GetDeviceAlarms(ctx context.Context) (json.RawMessage, error) {
	devicesData, err := c.Call(ctx, "devices.list", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("get devices: %w", err)
	}

	var devices []struct {
		DeviceID json.Number `json:"deviceId"`
	}
	if len(devicesData) > 0 {
		if err := json.Unmarshal(devicesData, &devices); err != nil {
			return nil, fmt.Errorf("parse devices: %w", err)
		}
	}

	allAlarms := []any{}
	for _, device := range devices {
		if device.DeviceID == "" {
			continue
		}

		settings, err := c.Call(ctx, "devices.settings", map[string]string{"deviceID": device.DeviceID.String()}, nil)
		if err != nil {
			continue // skip devices that fail
		}

		var settingsMap map[string]any
		if err := json.Unmarshal(settings, &settingsMap); err != nil {
			continue
		}

		if alarms, ok := settingsMap["alarms"]; ok {
			if alarmList, ok := alarms.([]any); ok {
				allAlarms = append(allAlarms, alarmList...)
			}
		}
	}

	result, err := json.Marshal(allAlarms)
	if err != nil {
		return nil, fmt.Errorf("marshal alarms: %w", err)
	}
	return json.RawMessage(result), nil
}

// Device is a registered device from the devices.list answer.
type Device struct {
	ID              json.Number
	Name            string
	WorkoutFileType string
	// flags holds every boolean field of the answer, such as workoutCapable.
	flags map[string]bool
}

func (d *Device) UnmarshalJSON(b []byte) error {
	var named struct {
		ID              json.Number `json:"deviceId"`
		Name            string      `json:"displayName"`
		WorkoutFileType string      `json:"workoutFileType"`
	}
	if err := json.Unmarshal(b, &named); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(b, &fields); err != nil {
		return err
	}
	*d = Device{ID: named.ID, Name: named.Name, WorkoutFileType: named.WorkoutFileType, flags: map[string]bool{}}
	for key, raw := range fields {
		var on bool
		if json.Unmarshal(raw, &on) == nil {
			d.flags[key] = on
		}
	}
	return nil
}

// workoutSportCapability is the device flag that the Garmin Connect web app
// requires, on top of workoutCapable, for a workout of each sport.
var workoutSportCapability = map[string]string{
	"running":           "runningWorkoutCapable",
	"cycling":           "cyclingWorkoutCapable",
	"other":             "customWorkoutCapable",
	"swimming":          "swimWorkoutCapable",
	"strength_training": "strengthWorkoutCapable",
	"cardio_training":   "cardioWorkoutCapable",
	"yoga":              "yogaWorkoutCapable",
	"pilates":           "pilatesWorkoutCapable",
	"mobility":          "mobilityWorkoutCapable",
	"hiit":              "hiitWorkoutCapable",
	"multi_sport":       "multisportWorkoutCapable",
	"rucking":           "ruckingWorkoutsCapable",
	"walking":           "walkWorkoutCapable",
}

// CanReceive reports whether the device takes the workout. A multisport
// workout also needs the flag of the sport of each segment.
func (d Device) CanReceive(w Workout) bool {
	if !d.flags["workoutCapable"] {
		return false
	}
	sports := []string{w.SportType.Key}
	if w.SportType.Key == "multi_sport" {
		for _, s := range w.Segments {
			sports = append(sports, s.SportType.Key)
		}
	}
	for _, sport := range sports {
		if flag, ok := workoutSportCapability[sport]; ok && !d.flags[flag] {
			return false
		}
	}
	return true
}
