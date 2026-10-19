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
	fields := map[string]string{}
	if e.Name != nil {
		fields["noteName"] = *e.Name
	}
	if e.Content != nil {
		fields["content"] = *e.Content
	}
	body, err := withFields(data, "note "+id, fields)
	if err != nil {
		return nil, err
	}
	return c.Call(ctx, "calendar.note.update", params, body)
}
