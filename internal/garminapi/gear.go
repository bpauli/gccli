package garminapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

var gearEndpoints = []Endpoint{
	{Name: "gear.list", Method: http.MethodGet, Host: ConnectAPI, Path: "/gear-service/gear/filterGear?userProfilePk={userProfilePK}"},
	{Name: "gear.stats", Method: http.MethodGet, Host: ConnectAPI, Path: "/gear-service/gear/stats/{gearUUID}"},
	{Name: "gear.activities", Method: http.MethodGet, Host: ConnectAPI, Path: "/activitylist-service/activities/{gearUUID}/gear?limit={limit}"},
	{Name: "gear.defaults", Method: http.MethodGet, Host: ConnectAPI, Path: "/gear-service/gear/user/{userProfileNumber}/activityTypes"},
	{Name: "gear.link", Method: http.MethodPut, Host: ConnectAPI, Path: "/gear-service/gear/link/{gearUUID}/activity/{activityID}"},
	{Name: "gear.unlink", Method: http.MethodPut, Host: ConnectAPI, Path: "/gear-service/gear/unlink/{gearUUID}/activity/{activityID}"},
	{Name: "gear.types", Method: http.MethodGet, Host: ConnectAPI, Path: "/gear-service/gear/types"},
	{Name: "gear.collections", Method: http.MethodGet, Host: ConnectAPI, Path: "/gear-service/collection"},
	{Name: "gear.get", Method: http.MethodGet, Host: ConnectAPI, Path: "/gear-service/gear/v2/{gearUUID}"},
	{Name: "gear.create", Method: http.MethodPost, Host: ConnectAPI, Path: "/gear-service/gear/v2", Body: true},
	{Name: "gear.update", Method: http.MethodPut, Host: ConnectAPI, Path: "/gear-service/gear/v2/{gearUUID}", Body: true},
	{Name: "gear.retire", Method: http.MethodPut, Host: ConnectAPI, Path: "/gear-service/gear/v2/{gearUUID}/status/RETIRED"},
	{Name: "gear.unretire", Method: http.MethodPut, Host: ConnectAPI, Path: "/gear-service/gear/v2/{gearUUID}/status/ACTIVE"},
	{Name: "gear.delete", Method: http.MethodDelete, Host: ConnectAPI, Path: "/gear-service/gear/v2/{gearUUID}"},
	{Name: "gear.collection.get", Method: http.MethodGet, Host: ConnectAPI, Path: "/gear-service/collection/{collectionUUID}"},
	{Name: "gear.collection.create", Method: http.MethodPost, Host: ConnectAPI, Path: "/gear-service/collection", Body: true},
	{Name: "gear.collection.update", Method: http.MethodPut, Host: ConnectAPI, Path: "/gear-service/collection/{collectionUUID}", Body: true},
	{Name: "gear.collection.add-gear", Method: http.MethodPost, Host: ConnectAPI, Path: "/gear-service/collection/{collectionUUID}/gear/{gearUUID}"},
	{Name: "gear.collection.delete", Method: http.MethodDelete, Host: ConnectAPI, Path: "/gear-service/collection/{collectionUUID}"},
}

// NewGear is the input of a new gear item. Type is a Garmin gear type such as
// SHOES, and FirstUseDate is YYYY-MM-DD.
type NewGear struct {
	Name         string
	Type         string
	Brand        string
	Model        string
	Notes        string
	FirstUseDate string
}

// CreateGear adds a gear item. Garmin requires notes of 1 to 2000 characters,
// so the name is sent as the notes when Notes is empty.
func (c *Client) CreateGear(ctx context.Context, g NewGear) (json.RawMessage, error) {
	notes := g.Notes
	if notes == "" {
		notes = g.Name
	}
	body, err := json.Marshal(map[string]any{
		"uuid":                    nil,
		"gearType":                g.Type,
		"brand":                   g.Brand,
		"model":                   g.Model,
		"name":                    g.Name,
		"firstUseDate":            g.FirstUseDate,
		"maxUsageDate":            nil,
		"maxUsageDistanceMeters":  0,
		"maxUsageDurationSeconds": 0,
		"usageType":               "DISTANCE",
		"notes":                   notes,
		"associatedActivityTypes": []any{},
	})
	if err != nil {
		return nil, fmt.Errorf("marshal gear: %w", err)
	}
	return c.Call(ctx, "gear.create", nil, body)
}

// GearEdit holds the new values of a gear update. An empty field keeps the
// saved value.
type GearEdit struct {
	Name  string
	Brand string
	Model string
	Notes string
}

// UpdateGear reads the gear item, sets the fields of e and sends the full
// object back, because Garmin rejects a partial body.
func (c *Client) UpdateGear(ctx context.Context, uuid string, e GearEdit) (json.RawMessage, error) {
	params := map[string]string{"gearUUID": uuid}
	data, err := c.Call(ctx, "gear.get", params, nil)
	if err != nil {
		return nil, err
	}
	body, err := withFields(data, "gear "+uuid, map[string]string{"name": e.Name, "brand": e.Brand, "model": e.Model, "notes": e.Notes})
	if err != nil {
		return nil, err
	}
	return c.Call(ctx, "gear.update", params, body)
}

// RenameGearCollection reads the gear collection, sets its name and sends the
// full object back, because Garmin rejects a partial body.
func (c *Client) RenameGearCollection(ctx context.Context, uuid, name string) (json.RawMessage, error) {
	params := map[string]string{"collectionUUID": uuid}
	data, err := c.Call(ctx, "gear.collection.get", params, nil)
	if err != nil {
		return nil, err
	}
	body, err := withFields(data, "gear collection "+uuid, map[string]string{"name": name})
	if err != nil {
		return nil, err
	}
	return c.Call(ctx, "gear.collection.update", params, body)
}

// withFields sets each non-empty value of fields in the JSON object data.
// The other values go back as Garmin sent them, so unknown keys and number
// text such as 0.0 stay the same.
func withFields(data json.RawMessage, what string, fields map[string]string) ([]byte, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil || obj == nil {
		return nil, fmt.Errorf("parse %s: not a JSON object", what)
	}
	for key, value := range fields {
		if value == "" {
			continue
		}
		v, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		obj[key] = v
	}
	body, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("marshal %s: %w", what, err)
	}
	return body, nil
}
