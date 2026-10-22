package garminapi

import (
	"context"
	"encoding/json"
	"net/http"
)

var calendarEndpoints = []Endpoint{
	{Name: "calendar.note.get", Method: http.MethodGet, Host: ConnectAPI, Path: "/calendar-service/note/{noteID}"},
	{Name: "calendar.note.create", Method: http.MethodPost, Host: ConnectAPI, Path: "/calendar-service/note", Body: true},
	{Name: "calendar.note.update", Method: http.MethodPut, Host: ConnectAPI, Path: "/calendar-service/note/{noteID}", Body: true},
	{Name: "calendar.note.move", Method: http.MethodPut, Host: ConnectAPI, Path: "/calendar-service/calendarItem/note/{noteID}/date/{date}"},
	{Name: "calendar.note.delete", Method: http.MethodDelete, Host: ConnectAPI, Path: "/calendar-service/note/{noteID}"},
	// The month is 0-based: 0 is January.
	{Name: "calendar.month", Method: http.MethodGet, Host: ConnectAPI, Path: "/calendar-service/year/{year}/month/{month}"},
	{Name: "calendar.export.get", Method: http.MethodGet, Host: ConnectAPI, Path: "/calendar-service/export"},
	{Name: "calendar.export.create", Method: http.MethodPost, Host: ConnectAPI, Path: "/calendar-service/export", Body: true},
	{Name: "calendar.export.delete", Method: http.MethodDelete, Host: ConnectAPI, Path: "/calendar-service/export/{exportID}"},
	{Name: "calendar.preferences.get", Method: http.MethodGet, Host: ConnectAPI, Path: "/calendar-service/preferences"},
	{Name: "calendar.preferences.update", Method: http.MethodPut, Host: ConnectAPI, Path: "/calendar-service/preferences", Body: true},
}

// SetCalendarPreference reads the calendar preferences, sets key to on and
// sends the full object back, because Garmin answers 500 to a partial
// object. It sends nothing when key is not a true or false value in the
// object, and the error then lists the keys that are.
func (c *Client) SetCalendarPreference(ctx context.Context, key string, on bool) (json.RawMessage, error) {
	data, err := c.Call(ctx, "calendar.preferences.get", nil, nil)
	if err != nil {
		return nil, err
	}
	body, err := setBoolPreference(data, "calendar preference", key, on)
	if err != nil {
		return nil, err
	}
	return c.Call(ctx, "calendar.preferences.update", nil, body)
}

// NoteEdit holds the new values of a note update. A nil field keeps the
// saved value, and an empty Content clears the text.
type NoteEdit struct {
	Name    *string
	Content *string
}

// UpdateNote reads the calendar note, sets the fields of e and sends the full
// object back, because Garmin rejects a body without the date.
func (c *Client) UpdateNote(ctx context.Context, id string, e NoteEdit) (json.RawMessage, error) {
	params := map[string]string{"noteID": id}
	data, err := c.Call(ctx, "calendar.note.get", params, nil)
	if err != nil {
		return nil, err
	}
	fields := map[string]any{}
	if e.Name != nil {
		fields["noteName"] = *e.Name
	}
	if e.Content != nil {
		fields["content"] = *e.Content
	}
	body, err := setFields(data, "note "+id, fields)
	if err != nil {
		return nil, err
	}
	return c.Call(ctx, "calendar.note.update", params, body)
}
