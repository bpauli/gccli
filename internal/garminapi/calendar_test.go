package garminapi

import "testing"

func TestCalendarEndpoints(t *testing.T) {
	note := map[string]string{"noteID": "12345"}
	testEndpoints(t, calendarEndpoints, []endpointCase{
		{"calendar.note.get", note, "GET /calendar-service/note/12345"},
		{"calendar.note.create", nil, "POST /calendar-service/note"},
		{"calendar.note.update", note, "PUT /calendar-service/note/12345"},
		{"calendar.note.move", map[string]string{"noteID": "12345", "date": "2026-10-20"}, "PUT /calendar-service/calendarItem/note/12345/date/2026-10-20"},
		{"calendar.note.delete", note, "DELETE /calendar-service/note/12345"},
	})
}
