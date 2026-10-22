package garminapi

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestNotificationsEndpoints(t *testing.T) {
	testEndpoints(t, notificationsEndpoints, []endpointCase{
		{"notifications.list", map[string]string{"limit": "50"}, "GET /notification-service/notification/notification/cursor?limit=50"},
		{"notifications.list", map[string]string{"limit": "3", "cursor": "abc"}, "GET /notification-service/notification/notification/cursor?limit=3&cursor=abc"},
		{"notifications.list", map[string]string{"limit": "3", "cursor": ""}, "GET /notification-service/notification/notification/cursor?limit=3"},
		{"notifications.dismiss", map[string]string{"notificationID": "777", "dismiss": "true"}, "PUT /notification-service/notification/777/dismiss/true"},
		{"notifications.dismiss", map[string]string{"notificationID": "777", "dismiss": "false"}, "PUT /notification-service/notification/777/dismiss/false"},
		{"notifications.unread", nil, "GET /web-gateway/inbox/unreadBubble"},
		{"notifications.preferences", nil, "GET /notification-service/preference/userPreference"},
		{"notifications.set-preferences", nil, "PUT /notification-service/preference/userPreference"},
	})
}

func TestNotificationsDismissRefusesOtherValues(t *testing.T) {
	_, client := testServer(t, http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("request sent: %s %s", r.Method, r.URL.RequestURI())
	}))
	_, err := client.Call(context.Background(), "notifications.dismiss", map[string]string{"notificationID": "777", "dismiss": "yes"}, nil)
	var epErr *EndpointError
	if !errors.As(err, &epErr) || epErr.Param != "dismiss" {
		t.Errorf("err = %v, want EndpointError for dismiss", err)
	}
}
