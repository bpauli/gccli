package garminapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// GetHydration returns hydration data for a given date.
// Date format: YYYY-MM-DD.
func (c *Client) GetHydration(ctx context.Context, date string) (json.RawMessage, error) {
	path := fmt.Sprintf("/usersummary-service/usersummary/hydration/daily/%s", date)
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// AddHydration adds a hydration log entry for the given date.
// Amount is in milliliters.
func (c *Client) AddHydration(ctx context.Context, date string, amountML float64) (json.RawMessage, error) {
	payload := map[string]any{
		"valueInML":    amountML,
		"calendarDate": date,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal hydration payload: %w", err)
	}
	return c.ConnectAPI(ctx, http.MethodPut, "/usersummary-service/usersummary/hydration/log", bytes.NewReader(body))
}
