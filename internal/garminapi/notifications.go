package garminapi

import "net/http"

var notificationsEndpoints = []Endpoint{
	{Name: "notifications.list", Method: http.MethodGet, Host: ConnectAPI, Path: "/notification-service/notification/notification/cursor"},
}
