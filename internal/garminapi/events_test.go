package garminapi

import "testing"

func TestEventsEndpoints(t *testing.T) {
	testEndpoints(t, eventsEndpoints, []endpointCase{
		{
			"events.list",
			map[string]string{"startDate": "2026-04-01", "pageIndex": "1", "limit": "20", "sortOrder": "eventDate_asc"},
			"GET /calendar-service/events?startDate=2026-04-01&pageIndex=1&limit=20&sortOrder=eventDate_asc",
		},
		{"events.upcoming", nil, "GET /calendar-service/events/upcoming"},
		{"events.add", nil, "POST /calendar-service/event"},
		{"events.delete", map[string]string{"eventID": "12345"}, "DELETE /calendar-service/event/12345"},
	})
}
