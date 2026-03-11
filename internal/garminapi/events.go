package garminapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// GetEvents returns calendar events starting from a date with pagination.
func (c *Client) GetEvents(ctx context.Context, startDate string, pageIndex, limit int, sortOrder string) (json.RawMessage, error) {
	path := fmt.Sprintf("/calendar-service/events?startDate=%s&pageIndex=%d&limit=%d&sortOrder=%s",
		startDate, pageIndex, limit, sortOrder)
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// AddEvent creates a new calendar event.
func (c *Client) AddEvent(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodPost, "/calendar-service/event", bytes.NewReader(payload))
}

// DeleteEvent deletes a calendar event by ID.
func (c *Client) DeleteEvent(ctx context.Context, eventID string) error {
	_, err := c.ConnectAPI(ctx, http.MethodDelete, "/calendar-service/event/"+eventID, nil)
	return err
}
