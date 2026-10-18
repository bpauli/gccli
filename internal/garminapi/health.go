package garminapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

var healthEndpoints = []Endpoint{
	{Name: "health.daily-summary", Method: http.MethodGet, Host: ConnectAPI, Path: "/usersummary-service/usersummary/daily/{displayName}?calendarDate={date}"},
	{Name: "health.steps", Method: http.MethodGet, Host: ConnectAPI, Path: "/wellness-service/wellness/dailySummaryChart/{displayName}?date={date}"},
	{Name: "health.daily-steps", Method: http.MethodGet, Host: ConnectAPI, Path: "/usersummary-service/stats/steps/daily/{startDate}/{endDate}"},
	{Name: "health.weekly-steps", Method: http.MethodGet, Host: ConnectAPI, Path: "/usersummary-service/stats/steps/weekly/{endDate}/{weeks}"},
	{Name: "health.heart-rate", Method: http.MethodGet, Host: ConnectAPI, Path: "/wellness-service/wellness/dailyHeartRate/{displayName}?date={date}"},
	{Name: "health.resting-heart-rate", Method: http.MethodGet, Host: ConnectAPI, Path: "/userstats-service/wellness/daily/{displayName}?fromDate={date}&untilDate={date}&metricId=60"},
	{Name: "health.floors", Method: http.MethodGet, Host: ConnectAPI, Path: "/wellness-service/wellness/floorsChartData/daily/{date}"},
	{Name: "health.sleep", Method: http.MethodGet, Host: ConnectAPI, Path: "/wellness-service/wellness/dailySleepData/{displayName}?date={date}&nonSleepBufferMinutes=60"},
	{Name: "health.stress", Method: http.MethodGet, Host: ConnectAPI, Path: "/wellness-service/wellness/dailyStress/{date}"},
	{Name: "health.weekly-stress", Method: http.MethodGet, Host: ConnectAPI, Path: "/usersummary-service/stats/stress/weekly/{endDate}/{weeks}"},
	{Name: "health.respiration", Method: http.MethodGet, Host: ConnectAPI, Path: "/wellness-service/wellness/daily/respiration/{date}"},
	{Name: "health.spo2", Method: http.MethodGet, Host: ConnectAPI, Path: "/wellness-service/wellness/daily/spo2/{date}"},
	{Name: "health.hrv", Method: http.MethodGet, Host: ConnectAPI, Path: "/hrv-service/hrv/{date}"},
	{Name: "health.body-battery", Method: http.MethodGet, Host: ConnectAPI, Path: "/wellness-service/wellness/bodyBattery/reports/daily?startDate={startDate}&endDate={endDate}"},
	{Name: "health.intensity-minutes", Method: http.MethodGet, Host: ConnectAPI, Path: "/wellness-service/wellness/daily/im/{date}"},
	{Name: "health.weekly-intensity-minutes", Method: http.MethodGet, Host: ConnectAPI, Path: "/usersummary-service/stats/im/weekly/{startDate}/{endDate}"},
	{Name: "health.training-readiness", Method: http.MethodGet, Host: ConnectAPI, Path: "/metrics-service/metrics/trainingreadiness/{date}"},
	{Name: "health.training-status", Method: http.MethodGet, Host: ConnectAPI, Path: "/metrics-service/metrics/trainingstatus/aggregated/{date}"},
	{Name: "health.fitness-age", Method: http.MethodGet, Host: ConnectAPI, Path: "/fitnessage-service/fitnessage/{date}"},
	{Name: "health.max-metrics", Method: http.MethodGet, Host: ConnectAPI, Path: "/metrics-service/metrics/maxmet/daily/{date}/{date}"},
	{Name: "health.lactate-threshold", Method: http.MethodGet, Host: ConnectAPI, Path: "/biometric-service/biometric/latestLactateThreshold"},
	{Name: "health.cycling-ftp", Method: http.MethodGet, Host: ConnectAPI, Path: "/biometric-service/biometric/latestFunctionalThresholdPower/CYCLING"},
	{Name: "health.endurance-score", Method: http.MethodGet, Host: ConnectAPI, Path: "/metrics-service/metrics/endurancescore?calendarDate={date}"},
	{Name: "health.hill-score", Method: http.MethodGet, Host: ConnectAPI, Path: "/metrics-service/metrics/hillscore?calendarDate={date}"},
	{Name: "health.all-day-events", Method: http.MethodGet, Host: ConnectAPI, Path: "/wellness-service/wellness/dailyEvents?calendarDate={date}"},
	{Name: "health.lifestyle-logging", Method: http.MethodGet, Host: ConnectAPI, Path: "/lifestylelogging-service/dailyLog/{date}"},
	{Name: "health.power-curve", Method: http.MethodGet, Host: ConnectAPI, Path: "/fitnessstats-service/powerCurve?startDate={startDate}&endDate={endDate}"},
	{Name: "health.running-economy", Method: http.MethodGet, Host: ConnectAPI, Path: "/metrics-service/metrics/runningeconomy/latest/{date}"},
}

// GetRacePredictions returns race prediction data.
// If startDate and endDate are empty, returns current predictions.
// Date format: YYYY-MM-DD.
func (c *Client) GetRacePredictions(ctx context.Context, startDate, endDate string) (json.RawMessage, error) {
	if startDate == "" {
		return c.ConnectAPI(ctx, http.MethodGet, "/metrics-service/metrics/racepredictions", nil)
	}
	path := fmt.Sprintf("/metrics-service/metrics/racepredictions/range/%s/%s?type=daily", startDate, endDate)
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}
