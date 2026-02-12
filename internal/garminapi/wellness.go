package garminapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// GetMenstrualCycleData returns menstrual cycle data for a given date range.
// Date format: YYYY-MM-DD.
func (c *Client) GetMenstrualCycleData(ctx context.Context, startDate, endDate string) (json.RawMessage, error) {
	path := fmt.Sprintf("/periodichealth-service/menstrualcycle/dayview/%s/%s", startDate, endDate)
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// GetMenstrualCycleSummary returns menstrual cycle summary data for a given date range.
// Date format: YYYY-MM-DD.
func (c *Client) GetMenstrualCycleSummary(ctx context.Context, startDate, endDate string) (json.RawMessage, error) {
	path := fmt.Sprintf("/periodichealth-service/menstrualcycle/summary/%s/%s", startDate, endDate)
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// GetPregnancySummary returns pregnancy summary data.
func (c *Client) GetPregnancySummary(ctx context.Context) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodGet, "/periodichealth-service/pregnancy/summary", nil)
}

// RequestReload requests Garmin to reload data for a given date.
// Date format: YYYY-MM-DD.
func (c *Client) RequestReload(ctx context.Context, date string) (json.RawMessage, error) {
	path := fmt.Sprintf("/wellness-service/wellness/epoch/request/%s", date)
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}
