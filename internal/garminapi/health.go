package garminapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// GetDailySummary returns the user's daily summary for a given date.
// Date format: YYYY-MM-DD.
func (c *Client) GetDailySummary(ctx context.Context, displayName, date string) (json.RawMessage, error) {
	path := fmt.Sprintf("/usersummary-service/usersummary/daily/%s?calendarDate=%s", displayName, date)
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// GetSteps returns step chart data for a given date.
// Date format: YYYY-MM-DD.
func (c *Client) GetSteps(ctx context.Context, displayName, date string) (json.RawMessage, error) {
	path := fmt.Sprintf("/wellness-service/wellness/dailySummaryChart/%s?date=%s", displayName, date)
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// GetDailySteps returns daily step data for a date range.
// Date format: YYYY-MM-DD.
func (c *Client) GetDailySteps(ctx context.Context, startDate, endDate string) (json.RawMessage, error) {
	path := fmt.Sprintf("/usersummary-service/stats/steps/daily/%s/%s", startDate, endDate)
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// GetWeeklySteps returns weekly step data ending at the given date.
// Date format: YYYY-MM-DD.
func (c *Client) GetWeeklySteps(ctx context.Context, endDate string, weeks int) (json.RawMessage, error) {
	path := fmt.Sprintf("/usersummary-service/stats/steps/weekly/%s/%d", endDate, weeks)
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// GetHeartRate returns heart rate data for a given date.
// Date format: YYYY-MM-DD.
func (c *Client) GetHeartRate(ctx context.Context, displayName, date string) (json.RawMessage, error) {
	path := fmt.Sprintf("/wellness-service/wellness/dailyHeartRate/%s?date=%s", displayName, date)
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// GetRestingHeartRate returns resting heart rate data for a given date.
// Date format: YYYY-MM-DD.
func (c *Client) GetRestingHeartRate(ctx context.Context, displayName, date string) (json.RawMessage, error) {
	path := fmt.Sprintf("/userstats-service/wellness/daily/%s?fromDate=%s&untilDate=%s&metricId=60",
		displayName, date, date)
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// GetFloors returns floors climbed data for a given date.
// Date format: YYYY-MM-DD.
func (c *Client) GetFloors(ctx context.Context, date string) (json.RawMessage, error) {
	path := "/wellness-service/wellness/floorsChartData/daily/" + date
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// GetSleep returns sleep data for a given date.
// Date format: YYYY-MM-DD.
func (c *Client) GetSleep(ctx context.Context, displayName, date string) (json.RawMessage, error) {
	path := fmt.Sprintf("/wellness-service/wellness/dailySleepData/%s?date=%s&nonSleepBufferMinutes=60",
		displayName, date)
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// GetStress returns stress data for a given date.
// Date format: YYYY-MM-DD.
func (c *Client) GetStress(ctx context.Context, date string) (json.RawMessage, error) {
	path := "/wellness-service/wellness/dailyStress/" + date
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// GetWeeklyStress returns weekly stress data ending at the given date.
// Date format: YYYY-MM-DD.
func (c *Client) GetWeeklyStress(ctx context.Context, endDate string, weeks int) (json.RawMessage, error) {
	path := fmt.Sprintf("/usersummary-service/stats/stress/weekly/%s/%d", endDate, weeks)
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// GetRespiration returns respiration data for a given date.
// Date format: YYYY-MM-DD.
func (c *Client) GetRespiration(ctx context.Context, date string) (json.RawMessage, error) {
	path := "/wellness-service/wellness/daily/respiration/" + date
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// GetSPO2 returns SpO2 (blood oxygen) data for a given date.
// Date format: YYYY-MM-DD.
func (c *Client) GetSPO2(ctx context.Context, date string) (json.RawMessage, error) {
	path := "/wellness-service/wellness/daily/spo2/" + date
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// GetHRV returns heart rate variability data for a given date.
// Date format: YYYY-MM-DD.
func (c *Client) GetHRV(ctx context.Context, date string) (json.RawMessage, error) {
	path := "/hrv-service/hrv/" + date
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// GetBodyBattery returns body battery data for a date range.
// Date format: YYYY-MM-DD.
func (c *Client) GetBodyBattery(ctx context.Context, startDate, endDate string) (json.RawMessage, error) {
	path := fmt.Sprintf("/wellness-service/wellness/bodyBattery/reports/daily?startDate=%s&endDate=%s",
		startDate, endDate)
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// GetIntensityMinutes returns intensity minutes data for a given date.
// Date format: YYYY-MM-DD.
func (c *Client) GetIntensityMinutes(ctx context.Context, date string) (json.RawMessage, error) {
	path := "/wellness-service/wellness/daily/im/" + date
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// GetWeeklyIntensityMinutes returns weekly intensity minutes data for a date range.
// Date format: YYYY-MM-DD.
func (c *Client) GetWeeklyIntensityMinutes(ctx context.Context, startDate, endDate string) (json.RawMessage, error) {
	path := fmt.Sprintf("/usersummary-service/stats/im/weekly/%s/%s", startDate, endDate)
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// GetTrainingReadiness returns training readiness data for a given date.
// Date format: YYYY-MM-DD.
func (c *Client) GetTrainingReadiness(ctx context.Context, date string) (json.RawMessage, error) {
	path := "/metrics-service/metrics/trainingreadiness/" + date
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// GetTrainingStatus returns training status data for a given date.
// Date format: YYYY-MM-DD.
func (c *Client) GetTrainingStatus(ctx context.Context, date string) (json.RawMessage, error) {
	path := "/metrics-service/metrics/trainingstatus/aggregated/" + date
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// GetFitnessAge returns fitness age data for a given date.
// Date format: YYYY-MM-DD.
func (c *Client) GetFitnessAge(ctx context.Context, date string) (json.RawMessage, error) {
	path := "/fitnessage-service/fitnessage/" + date
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// GetMaxMetrics returns VO2max and other max metrics for a given date.
// Date format: YYYY-MM-DD.
func (c *Client) GetMaxMetrics(ctx context.Context, date string) (json.RawMessage, error) {
	path := fmt.Sprintf("/metrics-service/metrics/maxmet/daily/%s/%s", date, date)
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// GetLactateThreshold returns the latest lactate threshold data.
func (c *Client) GetLactateThreshold(ctx context.Context) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodGet, "/biometric-service/biometric/latestLactateThreshold", nil)
}

// GetCyclingFTP returns the latest cycling functional threshold power.
func (c *Client) GetCyclingFTP(ctx context.Context) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodGet, "/biometric-service/biometric/latestFunctionalThresholdPower/CYCLING", nil)
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

// GetEnduranceScore returns endurance score data for a given date.
// Date format: YYYY-MM-DD.
func (c *Client) GetEnduranceScore(ctx context.Context, date string) (json.RawMessage, error) {
	path := fmt.Sprintf("/metrics-service/metrics/endurancescore?calendarDate=%s", date)
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// GetHillScore returns hill score data for a given date.
// Date format: YYYY-MM-DD.
func (c *Client) GetHillScore(ctx context.Context, date string) (json.RawMessage, error) {
	path := fmt.Sprintf("/metrics-service/metrics/hillscore?calendarDate=%s", date)
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// GetAllDayEvents returns daily wellness events for a given date.
// Date format: YYYY-MM-DD.
func (c *Client) GetAllDayEvents(ctx context.Context, date string) (json.RawMessage, error) {
	path := fmt.Sprintf("/wellness-service/wellness/dailyEvents?calendarDate=%s", date)
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// GetLifestyleLogging returns lifestyle logging data for a given date.
// Date format: YYYY-MM-DD.
func (c *Client) GetLifestyleLogging(ctx context.Context, date string) (json.RawMessage, error) {
	path := "/lifestylelogging-service/dailyLog/" + date
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}
