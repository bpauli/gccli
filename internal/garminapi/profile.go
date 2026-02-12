package garminapi

import (
	"context"
	"encoding/json"
	"net/http"
)

// GetProfile returns the authenticated user's profile.
func (c *Client) GetProfile(ctx context.Context) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodGet, "/userprofile-service/usersocialprofile", nil)
}

// GetUserSettings returns the authenticated user's settings.
func (c *Client) GetUserSettings(ctx context.Context) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodGet, "/userprofile-service/userprofile/user-settings", nil)
}
