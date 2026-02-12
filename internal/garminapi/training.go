package garminapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// GetTrainingPlans returns the user's training plans.
func (c *Client) GetTrainingPlans(ctx context.Context, locale string) (json.RawMessage, error) {
	if locale == "" {
		locale = "en"
	}
	path := fmt.Sprintf("/trainingplan-service/trainingplan/plans?locale=%s", locale)
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// GetTrainingPlan returns a specific training plan by ID.
func (c *Client) GetTrainingPlan(ctx context.Context, planID string) (json.RawMessage, error) {
	path := fmt.Sprintf("/trainingplan-service/trainingplan/plan/%s", planID)
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}
