package garminapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	// LoadLocation needs a zone database, and Windows has none.
	_ "time/tzdata"

	"github.com/bpauli/gccli/internal/units"
)

var profileEndpoints = []Endpoint{
	{Name: "profile.get", Method: http.MethodGet, Host: ConnectAPI, Path: "/userprofile-service/userprofile/settings"},
	{Name: "profile.user-settings", Method: http.MethodGet, Host: ConnectAPI, Path: "/userprofile-service/userprofile/user-settings"},
	{Name: "profile.social", Method: http.MethodGet, Host: ConnectAPI, Path: "/userprofile-service/socialProfile/{displayName}"},
	{Name: "profile.privacy-settings", Method: http.MethodGet, Host: ConnectAPI, Path: "/privacysettings-service/privacy-settings"},
	{Name: "profile.privacy-zones", Method: http.MethodGet, Host: ConnectAPI, Path: "/privacyzone-service/privacyZones"},
	{Name: "profile.privacy-zone.create", Method: http.MethodPost, Host: ConnectAPI, Path: "/privacyzone-service/privacyZones", Body: true},
	{Name: "profile.privacy-zone.delete", Method: http.MethodDelete, Host: ConnectAPI, Path: "/privacyzone-service/privacyZones/{privacyZoneID}"},
	{Name: "profile.hr-zones", Method: http.MethodGet, Host: ConnectAPI, Path: "/biometric-service/heartRateZones/v2"},
	{Name: "profile.power-zones", Method: http.MethodGet, Host: ConnectAPI, Path: "/biometric-service/powerZones/sports/all"},
}

// GetDisplayName returns the authenticated user's display name from profile settings.
func (c *Client) GetDisplayName(ctx context.Context) (string, error) {
	data, err := c.Call(ctx, "profile.get", nil, nil)
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

// GetSocialProfile returns the authenticated user's social profile.
func (c *Client) GetSocialProfile(ctx context.Context) (json.RawMessage, error) {
	displayName, err := c.GetDisplayName(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve display name: %w", err)
	}
	return c.Call(ctx, "profile.social", map[string]string{"displayName": displayName}, nil)
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

// GetTimeZone returns the time zone of the profile settings. Garmin names
// UTC "gmt", which is not an IANA name.
func (c *Client) GetTimeZone(ctx context.Context) (*time.Location, error) {
	data, err := c.Call(ctx, "profile.get", nil, nil)
	if err != nil {
		return nil, err
	}

	var settings struct {
		TimeZone string `json:"timeZone"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("parse profile settings: %w", err)
	}
	switch {
	case settings.TimeZone == "":
		return nil, fmt.Errorf("profile settings missing timeZone")
	case strings.EqualFold(settings.TimeZone, "gmt"):
		return time.UTC, nil
	}
	loc, err := time.LoadLocation(settings.TimeZone)
	if err != nil {
		return nil, fmt.Errorf("unknown time zone %q in profile settings", settings.TimeZone)
	}
	return loc, nil
}

// GetMeasurementSystem returns the unit system from userData.measurementSystem
// in the user settings. A missing or unknown value is an error.
func (c *Client) GetMeasurementSystem(ctx context.Context) (units.System, error) {
	data, err := c.Call(ctx, "profile.user-settings", nil, nil)
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
