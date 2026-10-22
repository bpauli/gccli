package garminapi

import (
	"context"
	"encoding/json"
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
	body, err := setBoolPreference(data, "notification preference", key, on)
	if err != nil {
		return nil, err
	}
	return c.Call(ctx, "notifications.set-preferences", nil, body)
}

// setBoolPreference returns the preferences object data with key set to on.
// It fails when key is not a true or false value in data, and the error then
// lists the keys that are. what names one preference in the errors.
func setBoolPreference(data json.RawMessage, what, key string, on bool) ([]byte, error) {
	var prefs map[string]json.RawMessage
	if err := json.Unmarshal(data, &prefs); err != nil || prefs == nil {
		return nil, fmt.Errorf("parse %ss: not a JSON object", what)
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
		return nil, fmt.Errorf("no %s %q; the keys are: %s", what, key, strings.Join(keys, ", "))
	case !isJSONBool(v):
		return nil, fmt.Errorf("%s %q is not true or false; the keys are: %s", what, key, strings.Join(keys, ", "))
	}
	return setFields(data, what+"s", map[string]any{key: on})
}

func isJSONBool(v json.RawMessage) bool {
	s := string(v)
	return s == "true" || s == "false"
}
