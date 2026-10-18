package garminapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

var hydrationEndpoints = []Endpoint{
	{Name: "hydration.get", Method: http.MethodGet, Host: ConnectAPI, Path: "/usersummary-service/usersummary/hydration/daily/{date}"},
	{Name: "hydration.log", Method: http.MethodPut, Host: ConnectAPI, Path: "/usersummary-service/usersummary/hydration/log", Body: true},
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
	return c.Call(ctx, "hydration.log", nil, body)
}
