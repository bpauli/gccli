package garminapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

var gearEndpoints = []Endpoint{
	{Name: "gear.list", Method: http.MethodGet, Host: ConnectAPI, Path: "/gear-service/gear/filterGear?userProfilePk={userProfilePK}"},
	{Name: "gear.stats", Method: http.MethodGet, Host: ConnectAPI, Path: "/gear-service/gear/stats/{gearUUID}"},
	{Name: "gear.activities", Method: http.MethodGet, Host: ConnectAPI, Path: "/activitylist-service/activities/{gearUUID}/gear?limit={limit}"},
	{Name: "gear.defaults", Method: http.MethodGet, Host: ConnectAPI, Path: "/gear-service/gear/user/{userProfileNumber}/activityTypes"},
	{Name: "gear.link", Method: http.MethodPut, Host: ConnectAPI, Path: "/gear-service/gear/link/{gearUUID}/activity/{activityID}"},
	{Name: "gear.unlink", Method: http.MethodPut, Host: ConnectAPI, Path: "/gear-service/gear/unlink/{gearUUID}/activity/{activityID}"},
}

// GetSocialProfile returns the authenticated user's social profile.
// It first resolves the display name via profile settings, then fetches the
// social profile. The response includes profileId needed by other API calls.
func (c *Client) GetSocialProfile(ctx context.Context) (json.RawMessage, error) {
	displayName, err := c.GetDisplayName(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve display name: %w", err)
	}
	return c.ConnectAPI(ctx, http.MethodGet, "/userprofile-service/socialProfile/"+displayName, nil)
}
