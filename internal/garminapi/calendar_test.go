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
		{"calendar.month", map[string]string{"year": "2026", "month": "9"}, "GET /calendar-service/year/2026/month/9"},
		{"calendar.export.get", nil, "GET /calendar-service/export"},
		{"calendar.export.create", nil, "POST /calendar-service/export"},
		{"calendar.export.delete", map[string]string{"exportID": "0123456789abcdef0123456789abcdef"}, "DELETE /calendar-service/export/0123456789abcdef0123456789abcdef"},
		{"calendar.preferences.get", nil, "GET /calendar-service/preferences"},
		{"calendar.preferences.update", nil, "PUT /calendar-service/preferences"},
	})
}
