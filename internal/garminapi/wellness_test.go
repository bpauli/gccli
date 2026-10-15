package garminapi

import "testing"

func TestWellnessEndpoints(t *testing.T) {
	span := map[string]string{"startDate": "2024-01-01", "endDate": "2024-01-31"}
	testEndpoints(t, wellnessEndpoints, []endpointCase{
		{"wellness.menstrual-cycle-data", span, "GET /periodichealth-service/menstrualcycle/dayview/2024-01-01/2024-01-31"},
		{"wellness.menstrual-cycle-summary", span, "GET /periodichealth-service/menstrualcycle/summary/2024-01-01/2024-01-31"},
		{"wellness.pregnancy-summary", nil, "GET /periodichealth-service/pregnancy/summary"},
		{"wellness.request-reload", map[string]string{"date": "2024-01-15"}, "GET /wellness-service/wellness/epoch/request/2024-01-15"},
	})
}
