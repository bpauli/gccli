package garminapi

import "net/http"

var eventsEndpoints = []Endpoint{
	{Name: "events.list", Method: http.MethodGet, Host: ConnectAPI, Path: "/calendar-service/events?startDate={startDate}&pageIndex={pageIndex}&limit={limit}&sortOrder={sortOrder}"},
	{Name: "events.upcoming", Method: http.MethodGet, Host: ConnectAPI, Path: "/calendar-service/events/upcoming"},
	{Name: "events.add", Method: http.MethodPost, Host: ConnectAPI, Path: "/calendar-service/event", Body: true},
	{Name: "events.delete", Method: http.MethodDelete, Host: ConnectAPI, Path: "/calendar-service/event/{eventID}"},
}
