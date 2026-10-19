package garminapi

import "testing"

func TestNotificationsEndpoints(t *testing.T) {
	testEndpoints(t, notificationsEndpoints, []endpointCase{
		{"notifications.list", nil, "GET /notification-service/notification/notification/cursor"},
		{"notifications.preferences", nil, "GET /notification-service/preference/userPreference"},
		{"notifications.set-preferences", nil, "PUT /notification-service/preference/userPreference"},
	})
}
