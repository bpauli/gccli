package garminapi

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestWellnessEndpoints(t *testing.T) {
	span := map[string]string{"startDate": "2024-01-01", "endDate": "2024-01-31"}
	day := map[string]string{"date": "2026-10-01"}
	testEndpoints(t, wellnessEndpoints, []endpointCase{
		{"wellness.menstrual-cycle-data", map[string]string{"date": "2026-10-01", "todayDate": "2026-10-20"}, "GET /periodichealth-service/menstrualcycle/dayview/2026-10-01?todayDate=2026-10-20"},
		{"wellness.menstrual-cycle-summary", day, "GET /periodichealth-service/menstrualcycle/summary/2026-10-01"},
		{"wellness.pregnancy-summary", nil, "GET /periodichealth-service/menstrualcycle/pregnancysnapshot"},
		{"wellness.jet-lag-trips", nil, "GET /jetlag-service/jetlag/trip/all"},
		{"wellness.jet-lag-status", map[string]string{"date": "2024-01-15"}, "GET /jetlag-service/jetlag/status/daily/2024-01-15"},
		{"wellness.menstrual-calendar", span, "GET /periodichealth-service/menstrualcycle/calendar/2024-01-01/2024-01-31"},
		{"wellness.menstrual-log", day, "GET /periodichealth-service/menstrualcycle/dailylog/2026-10-01"},
		{"wellness.set-menstrual-log", day, "POST /periodichealth-service/menstrualcycle/dailylog/2026-10-01"},
		{"wellness.delete-menstrual-log", day, "DELETE /periodichealth-service/menstrualcycle/dailylog/2026-10-01"},
		{"wellness.request-reload", map[string]string{"date": "2024-01-15"}, "POST /wellness-service/wellness/epoch/request/2024-01-15"},
	})
}

// The clock of the user is not in UTC, and Garmin takes reportTimestamp only
// in UTC with milliseconds and no zone suffix.
func TestSetMenstrualLogReportTimestamp(t *testing.T) {
	var body map[string]any
	_, client := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	flow := "LIGHT"
	now := time.Date(2026, 10, 15, 11, 30, 5, 123_000_000, time.FixedZone("CEST", 2*60*60))
	if _, err := client.SetMenstrualLog(context.Background(), "2026-10-01", MenstrualLogEdit{Flow: &flow}, now); err != nil {
		t.Fatal(err)
	}
	if got, want := body["reportTimestamp"], "2026-10-15T09:30:05.123"; got != want {
		t.Errorf("reportTimestamp = %v, want %q", got, want)
	}
}
