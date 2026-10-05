package garminapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/bpauli/gccli/internal/units"
)

// GetProfile returns the authenticated user's profile settings.
func (c *Client) GetProfile(ctx context.Context) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodGet, "/userprofile-service/userprofile/settings", nil)
}

// GetUserSettings returns the authenticated user's settings.
func (c *Client) GetUserSettings(ctx context.Context) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodGet, "/userprofile-service/userprofile/user-settings", nil)
}

// GetDisplayName returns the authenticated user's display name from profile settings.
func (c *Client) GetDisplayName(ctx context.Context) (string, error) {
	data, err := c.ConnectAPI(ctx, http.MethodGet, "/userprofile-service/userprofile/settings", nil)
	if err != nil {
		return "", err
	}

	var settings struct {
		DisplayName string `json:"displayName"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return "", fmt.Errorf("parse profile settings: %w", err)
	}
	if settings.DisplayName == "" {
		return "", fmt.Errorf("profile settings missing displayName")
	}
	return settings.DisplayName, nil
}

// GetMeasurementSystem returns the unit system from userData.measurementSystem
// in the user settings. A missing or unknown value is an error.
func (c *Client) GetMeasurementSystem(ctx context.Context) (units.System, error) {
	data, err := c.GetUserSettings(ctx)
	if err != nil {
		return units.Metric, err
	}

	var settings struct {
		UserData struct {
			MeasurementSystem string `json:"measurementSystem"`
		} `json:"userData"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return units.Metric, fmt.Errorf("parse user settings: %w", err)
	}
	if settings.UserData.MeasurementSystem == "" {
		return units.Metric, fmt.Errorf("user settings missing userData.measurementSystem")
	}
	return units.ParseSystem(settings.UserData.MeasurementSystem)
}
