package garminapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// GetSocialProfile returns the authenticated user's social profile.
// The response includes userProfileNumber and profileId needed by other API calls.
func (c *Client) GetSocialProfile(ctx context.Context) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodGet, "/userprofile-service/usersocialprofile", nil)
}

// GetGear returns all gear for a user profile.
func (c *Client) GetGear(ctx context.Context, userProfilePK string) (json.RawMessage, error) {
	path := "/gear-service/gear/filterGear?userProfilePk=" + userProfilePK
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// GetGearStats returns usage statistics for a specific gear item.
func (c *Client) GetGearStats(ctx context.Context, gearUUID string) (json.RawMessage, error) {
	path := fmt.Sprintf("/gear-service/gear/stats/%s", gearUUID)
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// GetGearActivities returns activities linked to a specific gear item.
func (c *Client) GetGearActivities(ctx context.Context, gearUUID string, limit int) (json.RawMessage, error) {
	path := fmt.Sprintf("/activitylist-service/activities/%s/gear?limit=%d", gearUUID, limit)
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// GetGearDefaults returns default gear per activity type for a user.
func (c *Client) GetGearDefaults(ctx context.Context, userProfileNumber string) (json.RawMessage, error) {
	path := fmt.Sprintf("/gear-service/gear/user/%s/activityTypes", userProfileNumber)
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// LinkGear links a gear item to an activity.
func (c *Client) LinkGear(ctx context.Context, gearUUID, activityID string) (json.RawMessage, error) {
	path := fmt.Sprintf("/gear-service/gear/link/%s/activity/%s", gearUUID, activityID)
	return c.ConnectAPI(ctx, http.MethodPut, path, nil)
}

// UnlinkGear unlinks a gear item from an activity.
func (c *Client) UnlinkGear(ctx context.Context, gearUUID, activityID string) (json.RawMessage, error) {
	path := fmt.Sprintf("/gear-service/gear/unlink/%s/activity/%s", gearUUID, activityID)
	return c.ConnectAPI(ctx, http.MethodPut, path, nil)
}
