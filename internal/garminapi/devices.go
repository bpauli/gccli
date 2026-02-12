package garminapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// GetDevices returns all registered devices for the current user.
func (c *Client) GetDevices(ctx context.Context) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodGet, "/device-service/deviceregistration/devices", nil)
}

// GetDeviceSettings returns settings for a specific device.
func (c *Client) GetDeviceSettings(ctx context.Context, deviceID string) (json.RawMessage, error) {
	path := fmt.Sprintf("/device-service/deviceservice/device-info/settings/%s", deviceID)
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// GetPrimaryTrainingDevice returns information about the primary training device.
func (c *Client) GetPrimaryTrainingDevice(ctx context.Context) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodGet, "/web-gateway/device-info/primary-training-device", nil)
}

// GetDeviceSolar returns solar charging data for a compatible device.
// Date format: YYYY-MM-DD.
func (c *Client) GetDeviceSolar(ctx context.Context, deviceID, startDate, endDate string) (json.RawMessage, error) {
	path := fmt.Sprintf("/web-gateway/solar/%s/%s/%s", deviceID, startDate, endDate)
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// GetDeviceAlarms returns alarms from all devices by fetching device settings
// for each registered device.
func (c *Client) GetDeviceAlarms(ctx context.Context) (json.RawMessage, error) {
	devicesData, err := c.GetDevices(ctx)
	if err != nil {
		return nil, fmt.Errorf("get devices: %w", err)
	}

	var devices []map[string]any
	if err := json.Unmarshal(devicesData, &devices); err != nil {
		return nil, fmt.Errorf("parse devices: %w", err)
	}

	var allAlarms []any
	for _, device := range devices {
		deviceID, ok := device["deviceId"]
		if !ok {
			continue
		}
		idStr := fmt.Sprintf("%v", deviceID)

		settings, err := c.GetDeviceSettings(ctx, idStr)
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

// GetLastUsedDevice returns the most recently used device.
func (c *Client) GetLastUsedDevice(ctx context.Context) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodGet, "/device-service/deviceservice/mylastused", nil)
}
