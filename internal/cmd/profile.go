package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

var profileGroup = Group{Name: "profile", Help: "View user profile.", Children: []Node{
	Leaf{Name: "view", Help: "Show user profile.", Default: true, Endpoint: "profile.get", Verb: "get profile", Output: JSON{}},
	Leaf{Name: "settings", Help: "Show user settings.", Endpoint: "profile.user-settings", Verb: "get user settings", Output: JSON{}},
	profilePrivacyGroup,
	profilePrivacyZonesGroup,
	Leaf{Name: "hr-zones", Help: "Show heart rate zones.", Endpoint: "profile.hr-zones", Verb: "get heart rate zones", Output: JSON{}},
	Leaf{Name: "power-zones", Help: "Show power zones for all sports.", Endpoint: "profile.power-zones", Verb: "get power zones", Output: JSON{}},
}}

var profilePrivacyGroup = Group{Name: "privacy", Help: "View privacy settings.", Children: []Node{
	Leaf{Name: "view", Help: "Show privacy settings.", Default: true, Endpoint: "profile.privacy-settings", Verb: "get privacy settings", Output: JSON{}},
}}

var profilePrivacyZonesGroup = Group{Name: "privacy-zones", Help: "List, add and delete privacy zones.", Children: []Node{
	Leaf{Name: "list", Help: "List privacy zones.", Default: true, Endpoint: "profile.privacy-zones", Verb: "get privacy zones", Output: JSON{List: true}},
	Custom{Name: "add", Help: "Add a privacy zone around a point.", Cmd: &ProfilePrivacyZonesAddCmd{}},
	Leaf{
		Name: "delete", Help: "Delete a privacy zone.",
		Spec: argsOf(&struct {
			ID string `arg:"" help:"Privacy zone ID."`
		}{}),
		Endpoint: "profile.privacy-zone.delete",
		Params:   Params{"privacyZoneID": In("ID")},
		Verb:     "delete privacy zone",
		Output:   Write{Done: "Deleted privacy zone {ID}", Confirm: "Delete privacy zone {ID}?"},
	},
}}

// ProfilePrivacyZonesAddCmd adds a privacy zone. Garmin accepts a radius of
// 500 to 1500 meters.
type ProfilePrivacyZonesAddCmd struct {
	Lat    float64 `required:"" placeholder:"DEGREES" help:"Latitude of the center, -90 to 90. Give a negative value as --lat=-33.9."`
	Lon    float64 `required:"" placeholder:"DEGREES" help:"Longitude of the center, -180 to 180. Give a negative value as --lon=-30."`
	Radius int     `required:"" placeholder:"METERS" help:"Radius in meters, 500 to 1500."`
}

func (c *ProfilePrivacyZonesAddCmd) Validate() error {
	// The negated range checks also refuse NaN.
	switch {
	case !(c.Lat >= -90 && c.Lat <= 90):
		return errors.New("--lat must be between -90 and 90")
	case !(c.Lon >= -180 && c.Lon <= 180):
		return errors.New("--lon must be between -180 and 180")
	case c.Radius < 500 || c.Radius > 1500:
		return errors.New("--radius must be between 500 and 1500 meters")
	}
	return nil
}

func (c *ProfilePrivacyZonesAddCmd) Run(g *Globals) error {
	body, err := json.Marshal(map[string]any{"centerLatitude": c.Lat, "centerLongitude": c.Lon, "radius": c.Radius})
	if err != nil {
		return err
	}
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "profile.privacy-zone.create", nil, body)
	if err != nil {
		return fmt.Errorf("add privacy zone: %w", err)
	}
	var resp map[string]any
	_ = json.Unmarshal(data, &resp)
	center := strconv.FormatFloat(c.Lat, 'f', -1, 64) + ", " + strconv.FormatFloat(c.Lon, 'f', -1, 64)
	if id := jsonString(resp, "privacyZoneId"); id != "" {
		return reportWrite(g, data, "Added privacy zone of %d m around %s (ID: %s)", c.Radius, center, id)
	}
	return reportWrite(g, data, "Added privacy zone of %d m around %s", c.Radius, center)
}
