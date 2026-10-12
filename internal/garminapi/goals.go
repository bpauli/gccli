package garminapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// GetGoals returns the user's goals with the given status ("active", "future" or "past").
// The goal-service requires a status.
func (c *Client) GetGoals(ctx context.Context, status string) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodGet, "/goal-service/goal/goals?status="+status, nil)
}

// GetBadgesEarned returns all badges earned by the user.
func (c *Client) GetBadgesEarned(ctx context.Context) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodGet, "/badge-service/badge/earned", nil)
}

// GetBadgesAvailable returns all available badges.
func (c *Client) GetBadgesAvailable(ctx context.Context) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodGet, "/badge-service/badge/available", nil)
}

// GetBadgesInProgress returns badges currently in progress.
func (c *Client) GetBadgesInProgress(ctx context.Context) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodGet, "/badge-service/badge/in-progress", nil)
}

// GetChallenges returns active challenges the user is participating in.
func (c *Client) GetChallenges(ctx context.Context) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodGet, "/challenge-service/challenge/joined", nil)
}

// GetBadgeChallenges returns badge challenges.
func (c *Client) GetBadgeChallenges(ctx context.Context) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodGet, "/badge-service/badge/challenges", nil)
}

// GetPersonalRecords returns the user's personal records for a given owner display name.
func (c *Client) GetPersonalRecords(ctx context.Context, ownerDisplayName string) (json.RawMessage, error) {
	path := fmt.Sprintf("/personalrecord-service/personalrecord/prs/%s", ownerDisplayName)
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}
