package garminapi

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"time"
)

var wellnessEndpoints = []Endpoint{
	{Name: "wellness.menstrual-cycle-data", Method: http.MethodGet, Host: ConnectAPI, Path: "/periodichealth-service/menstrualcycle/dayview/{startDate}/{endDate}"},
	{Name: "wellness.menstrual-cycle-summary", Method: http.MethodGet, Host: ConnectAPI, Path: "/periodichealth-service/menstrualcycle/summary/{startDate}/{endDate}"},
	{Name: "wellness.pregnancy-summary", Method: http.MethodGet, Host: ConnectAPI, Path: "/periodichealth-service/pregnancy/summary"},
	{Name: "wellness.jet-lag-trips", Method: http.MethodGet, Host: ConnectAPI, Path: "/jetlag-service/jetlag/trip/all"},
	{Name: "wellness.jet-lag-status", Method: http.MethodGet, Host: ConnectAPI, Path: "/jetlag-service/jetlag/status/daily/{date}"},
	{Name: "wellness.menstrual-calendar", Method: http.MethodGet, Host: ConnectAPI, Path: "/periodichealth-service/menstrualcycle/calendar/{startDate}/{endDate}"},
	{Name: "wellness.menstrual-log", Method: http.MethodGet, Host: ConnectAPI, Path: "/periodichealth-service/menstrualcycle/dailylog/{date}"},
	{Name: "wellness.set-menstrual-log", Method: http.MethodPost, Host: ConnectAPI, Path: "/periodichealth-service/menstrualcycle/dailylog/{date}", Body: true},
	{Name: "wellness.delete-menstrual-log", Method: http.MethodDelete, Host: ConnectAPI, Path: "/periodichealth-service/menstrualcycle/dailylog/{date}"},
	{Name: "wellness.request-reload", Method: http.MethodPost, Host: ConnectAPI, Path: "/wellness-service/wellness/epoch/request/{date}"},
}

// MenstrualLogEdit holds the changes to the menstrual daily log of a day, in
// Garmin's values. A nil field keeps the saved value. An empty string or an
// empty list clears it.
type MenstrualLogEdit struct {
	Flow           *string   `json:"flow,omitempty"`
	Symptoms       *[]string `json:"symptoms,omitempty"`
	Moods          *[]string `json:"moods,omitempty"`
	Discharge      *[]string `json:"discharge,omitempty"`
	SexDrive       *string   `json:"sexDrive,omitempty"`
	SexualActivity *string   `json:"sexualActivity,omitempty"`
	Notes          *string   `json:"notes,omitempty"`
	OvulationDay   *bool     `json:"ovulationDay,omitempty"`
}

// menstrualLogTimestamp is the format of reportTimestamp: UTC without a zone
// suffix, because Garmin answers 500 to a "Z".
const menstrualLogTimestamp = "2006-01-02T15:04:05.000"

// SetMenstrualLog reads the menstrual daily log of date, sets the fields of e
// and saves the full log. Garmin replaces the whole day, so a field that e
// does not change must be sent again to keep it.
func (c *Client) SetMenstrualLog(ctx context.Context, date string, e MenstrualLogEdit, now time.Time) (json.RawMessage, error) {
	params := map[string]string{"date": date}
	saved, err := c.Call(ctx, "wellness.menstrual-log", params, nil)
	if err != nil {
		return nil, err
	}
	body, err := menstrualLogBody(saved, date, e, now)
	if err != nil {
		return nil, err
	}
	return c.Call(ctx, "wellness.set-menstrual-log", params, body)
}

// menstrualLogBody returns the saved log with the fields of e set. Garmin
// ignores a write whose reportTimestamp is older than the saved one, so the
// body gets now and never the saved timestamp.
func menstrualLogBody(saved json.RawMessage, date string, e MenstrualLogEdit, now time.Time) ([]byte, error) {
	log := map[string]json.RawMessage{}
	if len(saved) > 0 {
		if err := json.Unmarshal(saved, &log); err != nil || log == nil {
			return nil, fmt.Errorf("parse menstrual log of %s: not a JSON object", date)
		}
	}
	changes, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(changes, &log); err != nil {
		return nil, err
	}
	for key, value := range maps.Clone(log) {
		// An omitted note keeps the saved one, so a cleared note must be sent.
		if isEmptyJSON(value) && (key != "notes" || e.Notes == nil) {
			delete(log, key)
		}
	}
	delete(log, "userProfilePk")
	log["calendarDate"], _ = json.Marshal(date)
	log["reportTimestamp"], _ = json.Marshal(now.UTC().Format(menstrualLogTimestamp))
	return json.Marshal(log)
}

func isEmptyJSON(value json.RawMessage) bool {
	var v any
	if err := json.Unmarshal(value, &v); err != nil {
		return false
	}
	switch v := v.(type) {
	case nil:
		return true
	case string:
		return v == ""
	case []any:
		return len(v) == 0
	}
	return false
}
