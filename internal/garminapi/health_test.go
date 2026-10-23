package garminapi

import (
	"maps"
	"testing"
)

func TestHealthEndpoints(t *testing.T) {
	day := map[string]string{"date": "2024-01-15"}
	user := map[string]string{"displayName": "jo", "date": "2024-01-15"}
	span := map[string]string{"startDate": "2024-01-01", "endDate": "2024-01-31"}
	withAggregation := func(args map[string]string, aggregation string) map[string]string {
		out := map[string]string{"aggregation": aggregation}
		maps.Copy(out, args)
		return out
	}
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
		{"health.sleep-naps", day, "GET /sleep-service/sleep/naps/2024-01-15"},
		{"health.add-nap", nil, "POST /sleep-service/sleep/nap"},
		{"health.delete-nap", map[string]string{"date": "2024-01-15", "startMillis": "1705320000000"}, "DELETE /sleep-service/sleep/nap/2024-01-15/1705320000000"},
		{"health.sleep-note", day, "PUT /sleep-service/sleep/dailySleep/2024-01-15/note"},
		{"health.daily-sleep", span, "GET /sleep-service/stats/sleep/daily/2024-01-01/2024-01-31"},
		{"health.weekly-sleep", map[string]string{"endDate": "2024-01-31"}, "GET /sleep-service/stats/sleep/weekly/2024-01-31/52"},
		{"health.overall-sleep", span, "GET /sleep-service/stats/sleep/overall/2024-01-01/2024-01-31"},
		{"health.stress", day, "GET /wellness-service/wellness/dailyStress/2024-01-15"},
		{"health.daily-stress", span, "GET /usersummary-service/stats/stress/daily/2024-01-01/2024-01-31"},
		{"health.weekly-stress", map[string]string{"endDate": "2024-01-31", "weeks": "12"}, "GET /usersummary-service/stats/stress/weekly/2024-01-31/12"},
		{"health.respiration", day, "GET /wellness-service/wellness/daily/respiration/2024-01-15"},
		{"health.daily-respiration", span, "GET /usersummary-service/stats/respiration/daily/2024-01-01/2024-01-31"},
		{"health.spo2", day, "GET /wellness-service/wellness/daily/spo2/2024-01-15"},
		{"health.hrv", day, "GET /hrv-service/hrv/2024-01-15"},
		{"health.body-battery", span, "GET /wellness-service/wellness/bodyBattery/reports/daily?startDate=2024-01-01&endDate=2024-01-31"},
		{"health.daily-body-battery", span, "GET /usersummary-service/stats/bodybattery/daily/2024-01-01/2024-01-31"},
		{"health.intensity-minutes", day, "GET /wellness-service/wellness/daily/im/2024-01-15"},
		{"health.weekly-intensity-minutes", span, "GET /usersummary-service/stats/im/weekly/2024-01-01/2024-01-31"},
		{"health.training-readiness", day, "GET /metrics-service/metrics/trainingreadiness/2024-01-15"},
		{"health.training-status", day, "GET /metrics-service/metrics/trainingstatus/aggregated/2024-01-15"},
		{"health.daily-training-status", day, "GET /metrics-service/metrics/trainingstatus/daily/2024-01-15"},
		{"health.weekly-training-status", map[string]string{"displayName": "jo", "startDate": "2024-01-01", "endDate": "2024-01-31"}, "GET /metrics-service/metrics/trainingstatus/weekly/jo?fromCalendarDate=2024-01-01&toCalendarDate=2024-01-31"},
		{"health.fitness-age", day, "GET /fitnessage-service/fitnessage/2024-01-15"},
		{"health.max-metrics", day, "GET /metrics-service/metrics/maxmet/daily/2024-01-15/2024-01-15"},
		{"health.lactate-threshold", nil, "GET /biometric-service/biometric/latestLactateThreshold"},
		{"health.cycling-ftp", nil, "GET /biometric-service/biometric/latestFunctionalThresholdPower/CYCLING"},
		{"health.critical-swim-speed", day, "GET /biometric-service/criticalSwimSpeed/latest/2024-01-15"},
		{"health.critical-swim-speed-range", span, "GET /biometric-service/criticalSwimSpeed/range/2024-01-01/2024-01-31"},
		{"health.power-to-weight", day, "GET /biometric-service/biometric/powerToWeight/latest/2024-01-15"},
		{"health.cycling-ftp-range", span, "GET /biometric-service/stats/functionalThresholdPower/range/2024-01-01/2024-01-31?sport=CYCLING"},
		{"health.lactate-heart-rate-range", span, "GET /biometric-service/stats/lactateThresholdHeartRate/range/2024-01-01/2024-01-31"},
		{"health.lactate-speed-range", span, "GET /biometric-service/stats/lactateThresholdSpeed/range/2024-01-01/2024-01-31"},
		{"health.power-to-weight-range", span, "GET /biometric-service/stats/powerToWeight/range/2024-01-01/2024-01-31"},
		{"health.endurance-score", day, "GET /metrics-service/metrics/endurancescore?calendarDate=2024-01-15"},
		{"health.endurance-score-stats", withAggregation(span, "monthly"), "GET /metrics-service/metrics/endurancescore/stats?startDate=2024-01-01&endDate=2024-01-31&aggregation=monthly"},
		{"health.hill-score", day, "GET /metrics-service/metrics/hillscore?calendarDate=2024-01-15"},
		{"health.hill-score-stats", withAggregation(span, "daily"), "GET /metrics-service/metrics/hillscore/stats?startDate=2024-01-01&endDate=2024-01-31&aggregation=daily"},
		{"health.all-day-events", day, "GET /wellness-service/wellness/dailyEvents?calendarDate=2024-01-15"},
		{"health.lifestyle-logging", day, "GET /lifestylelogging-service/dailyLog/2024-01-15"},
		{"health.power-curve", span, "GET /fitnessstats-service/powerCurve?startDate=2024-01-01&endDate=2024-01-31"},
		{
			"health.progress",
			map[string]string{"aggregation": "weekly", "activityType": "running,cycling", "startDate": "2024-01-01", "endDate": "2024-01-31", "metric": "distance,duration"},
			"GET /fitnessstats-service/activity?aggregation=weekly&activityType=running&activityType=cycling&startDate=2024-01-01&endDate=2024-01-31&metric=distance&metric=duration&standardizedUnits=true",
		},
		{
			"health.progress-activities",
			map[string]string{"startDate": "2024-01-01", "endDate": "2024-01-31", "metric": "distance,avgHr"},
			"GET /fitnessstats-service/activity/all?startDate=2024-01-01&endDate=2024-01-31&metric=distance&metric=avgHr&standardizedUnits=true",
		},
		{"health.progress-metrics", nil, "GET /fitnessstats-service/activity/availableMetrics"},
		{"health.running-economy", day, "GET /metrics-service/metrics/runningeconomy/latest/2024-01-15"},
		{"health.running-tolerance", withAggregation(span, "weekly"), "GET /metrics-service/metrics/runningtolerance/stats?startDate=2024-01-01&endDate=2024-01-31&aggregation=weekly"},
		{"health.cycling-ability", withAggregation(span, "WEEKLY"), "GET /metrics-service/metrics/cycling-ability?startDate=2024-01-01&endDate=2024-01-31&aggregation=WEEKLY"},
		{"health.acclimation", span, "GET /wellness-service/stats/daily/acclimation?fromDate=2024-01-01&untilDate=2024-01-31"},
		{"health.daily-acclimation", day, "GET /wellness-service/wellness/daily/spo2acclimation/2024-01-15"},
		{"health.race-predictions", map[string]string{"displayName": "jo"}, "GET /metrics-service/metrics/racepredictions/latest/jo"},
		{"health.race-predictions-range", map[string]string{"displayName": "jo", "startDate": "2024-01-01", "endDate": "2024-01-31"}, "GET /metrics-service/metrics/racepredictions/daily/jo?fromCalendarDate=2024-01-01&toCalendarDate=2024-01-31"},
		{"health.one-rep-max", day, "GET /fitnessstats-service/vbt-stats/one-rep-max/current/2024-01-15"},
	})
}
