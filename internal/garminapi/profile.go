package garminapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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

// GetProfilePK returns the profileId of the authenticated user's social
// profile, which the gear endpoints take as the user profile PK.
func (c *Client) GetProfilePK(ctx context.Context) (string, error) {
	data, err := c.GetSocialProfile(ctx)
	if err != nil {
		return "", fmt.Errorf("get user profile: %w", err)
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()

	var profile map[string]any
	if err := dec.Decode(&profile); err != nil {
		return "", fmt.Errorf("parse user profile: %w", err)
	}

	pk, ok := profile["profileId"]
	if !ok {
		return "", fmt.Errorf("user profile missing profileId field")
	}

	var s string
	switch v := pk.(type) {
	case json.Number:
		s = v.String()
	case string:
		s = v
	}
	if s == "" || strings.Trim(s, "0123456789") != "" {
		return "", fmt.Errorf("user profile profileId field is not an unsigned integer")
	}
	return s, nil
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
