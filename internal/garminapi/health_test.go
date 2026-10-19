package garminapi

import (
	"context"
	"net/http"
	"testing"
)

func TestHealthEndpoints(t *testing.T) {
	day := map[string]string{"date": "2024-01-15"}
	user := map[string]string{"displayName": "jo", "date": "2024-01-15"}
	span := map[string]string{"startDate": "2024-01-01", "endDate": "2024-01-31"}
	testEndpoints(t, healthEndpoints, []endpointCase{
		{"health.daily-summary", user, "GET /usersummary-service/usersummary/daily/jo?calendarDate=2024-01-15"},
		{"health.steps", user, "GET /wellness-service/wellness/dailySummaryChart/jo?date=2024-01-15"},
		{"health.daily-steps", span, "GET /usersummary-service/stats/steps/daily/2024-01-01/2024-01-31"},
		{"health.weekly-steps", map[string]string{"endDate": "2024-01-31", "weeks": "4"}, "GET /usersummary-service/stats/steps/weekly/2024-01-31/4"},
		{"health.monthly-steps", map[string]string{"endDate": "2024-01-31", "months": "6"}, "GET /usersummary-service/stats/steps/monthly/2024-01-31/6"},
		{"health.weekly-calories", map[string]string{"endDate": "2024-01-31", "weeks": "4"}, "GET /usersummary-service/stats/calories/weekly/2024-01-31/4"},
		{"health.weekly-pushes", map[string]string{"endDate": "2024-01-31", "weeks": "4"}, "GET /usersummary-service/stats/pushes/weekly/2024-01-31/4"},
		{"health.monthly-pushes", map[string]string{"endDate": "2024-01-31", "months": "6"}, "GET /usersummary-service/stats/pushes/monthly/2024-01-31/6"},
		{"health.lifetime-totals", map[string]string{"displayName": "jo"}, "GET /usersummary-service/stats/connectLifetimeTotals/jo"},
		{"health.heart-rate", user, "GET /wellness-service/wellness/dailyHeartRate/jo?date=2024-01-15"},
		{"health.daily-heart-rate", span, "GET /usersummary-service/stats/heartRate/daily/2024-01-01/2024-01-31"},
		{"health.weekly-heart-rate", map[string]string{"endDate": "2024-01-31", "weeks": "4"}, "GET /usersummary-service/stats/heartRate/weekly/2024-01-31/4"},
		{"health.resting-heart-rate", user, "GET /userstats-service/wellness/daily/jo?fromDate=2024-01-15&untilDate=2024-01-15&metricId=60"},
		{"health.floors", day, "GET /wellness-service/wellness/floorsChartData/daily/2024-01-15"},
		{"health.daily-floors", span, "GET /usersummary-service/stats/floors/daily/2024-01-01/2024-01-31"},
		{"health.weekly-floors", map[string]string{"endDate": "2024-01-31", "weeks": "4"}, "GET /usersummary-service/stats/floors/weekly/2024-01-31/4"},
		{"health.sleep", user, "GET /wellness-service/wellness/dailySleepData/jo?date=2024-01-15&nonSleepBufferMinutes=60"},
		{"health.stress", day, "GET /wellness-service/wellness/dailyStress/2024-01-15"},
		{"health.daily-stress", span, "GET /usersummary-service/stats/stress/daily/2024-01-01/2024-01-31"},
		{"health.weekly-stress", map[string]string{"endDate": "2024-01-31", "weeks": "12"}, "GET /usersummary-service/stats/stress/weekly/2024-01-31/12"},
		{"health.respiration", day, "GET /wellness-service/wellness/daily/respiration/2024-01-15"},
		{"health.daily-respiration", span, "GET /usersummary-service/stats/respiration/daily/2024-01-01/2024-01-31"},
		{"health.spo2", day, "GET /wellness-service/wellness/daily/spo2/2024-01-15"},
		{"health.hrv", day, "GET /hrv-service/hrv/2024-01-15"},
		{"health.body-battery", span, "GET /wellness-service/wellness/bodyBattery/reports/daily?startDate=2024-01-01&endDate=2024-01-31"},
		{"health.intensity-minutes", day, "GET /wellness-service/wellness/daily/im/2024-01-15"},
		{"health.weekly-intensity-minutes", span, "GET /usersummary-service/stats/im/weekly/2024-01-01/2024-01-31"},
		{"health.training-readiness", day, "GET /metrics-service/metrics/trainingreadiness/2024-01-15"},
		{"health.training-status", day, "GET /metrics-service/metrics/trainingstatus/aggregated/2024-01-15"},
		{"health.fitness-age", day, "GET /fitnessage-service/fitnessage/2024-01-15"},
		{"health.max-metrics", day, "GET /metrics-service/metrics/maxmet/daily/2024-01-15/2024-01-15"},
		{"health.lactate-threshold", nil, "GET /biometric-service/biometric/latestLactateThreshold"},
		{"health.cycling-ftp", nil, "GET /biometric-service/biometric/latestFunctionalThresholdPower/CYCLING"},
		{"health.critical-swim-speed", day, "GET /biometric-service/criticalSwimSpeed/latest/2024-01-15"},
		{"health.endurance-score", day, "GET /metrics-service/metrics/endurancescore?calendarDate=2024-01-15"},
		{"health.hill-score", day, "GET /metrics-service/metrics/hillscore?calendarDate=2024-01-15"},
		{"health.all-day-events", day, "GET /wellness-service/wellness/dailyEvents?calendarDate=2024-01-15"},
		{"health.lifestyle-logging", day, "GET /lifestylelogging-service/dailyLog/2024-01-15"},
		{"health.power-curve", span, "GET /fitnessstats-service/powerCurve?startDate=2024-01-01&endDate=2024-01-31"},
		{"health.running-economy", day, "GET /metrics-service/metrics/runningeconomy/latest/2024-01-15"},
	})
}

func TestGetRacePredictions_Current(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metrics-service/metrics/racepredictions" {
			t.Errorf("path = %s, want /metrics-service/metrics/racepredictions", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"5k":1200,"10k":2500,"halfMarathon":5400,"marathon":11200}`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetRacePredictions(context.Background(), "", "")
	if err != nil {
		t.Fatalf("GetRacePredictions: %v", err)
	}
	if data == nil {
		t.Fatal("expected data")
	}
}

func TestGetRacePredictions_DateRange(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metrics-service/metrics/racepredictions/range/2024-01-01/2024-01-31" {
			t.Errorf("path = %s, want /metrics-service/metrics/racepredictions/range/2024-01-01/2024-01-31", r.URL.Path)
		}
		if got := r.URL.Query().Get("type"); got != "daily" {
			t.Errorf("type = %s, want daily", got)
		}
		_, _ = w.Write([]byte(`[{"date":"2024-01-01"}]`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetRacePredictions(context.Background(), "2024-01-01", "2024-01-31")
	if err != nil {
		t.Fatalf("GetRacePredictions: %v", err)
	}
	if data == nil {
		t.Fatal("expected data")
	}
}
