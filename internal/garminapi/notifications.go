package garminapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
)

var notificationsEndpoints = []Endpoint{
	{Name: "notifications.list", Method: http.MethodGet, Host: ConnectAPI, Path: "/notification-service/notification/notification/cursor"},
	{Name: "notifications.preferences", Method: http.MethodGet, Host: ConnectAPI, Path: "/notification-service/preference/userPreference"},
	{Name: "notifications.set-preferences", Method: http.MethodPut, Host: ConnectAPI, Path: "/notification-service/preference/userPreference", Body: true},
}

// SetNotificationPreference reads the notification preferences, sets key to
// on and sends the full object back. It sends nothing when key is not a true
// or false value in the object, and the error then lists the keys that are.
func (c *Client) SetNotificationPreference(ctx context.Context, key string, on bool) (json.RawMessage, error) {
	data, err := c.Call(ctx, "notifications.preferences", nil, nil)
	if err != nil {
		return nil, err
	}
	var prefs map[string]json.RawMessage
	if err := json.Unmarshal(data, &prefs); err != nil || prefs == nil {
		return nil, errors.New("parse notification preferences: not a JSON object")
	}
	var keys []string
	for k, v := range prefs {
		if isJSONBool(v) {
			keys = append(keys, k)
		}
	}
	slices.Sort(keys)
	switch v, ok := prefs[key]; {
	case !ok:
		return nil, fmt.Errorf("no notification preference %q; the keys are: %s", key, strings.Join(keys, ", "))
	case !isJSONBool(v):
		return nil, fmt.Errorf("notification preference %q is not true or false; the keys are: %s", key, strings.Join(keys, ", "))
	}
	body, err := setFields(data, "notification preferences", map[string]any{key: on})
	if err != nil {
		return nil, err
	}
	return c.Call(ctx, "notifications.set-preferences", nil, body)
}

func isJSONBool(v json.RawMessage) bool {
	s := string(v)
	return s == "true" || s == "false"
}
