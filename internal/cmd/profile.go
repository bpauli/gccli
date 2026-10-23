package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/bpauli/gccli/internal/garminapi"
)

var profileGroup = Group{Name: "profile", Help: "View user profile.", Children: []Node{
	Leaf{Name: "view", Help: "Show user profile.", Default: true, Endpoint: "profile.get", Verb: "get profile", Output: JSON{}},
	Leaf{Name: "settings", Help: "Show user settings.", Endpoint: "profile.user-settings", Verb: "get user settings", Output: JSON{}},
	profilePrivacyGroup,
	profilePrivacyZonesGroup,
	Leaf{Name: "hr-zones", Help: "Show heart rate zones.", Endpoint: "profile.hr-zones", Verb: "get heart rate zones", Output: JSON{List: true}},
	profilePowerZonesGroup,
}}

var profilePowerZonesGroup = Group{Name: "power-zones", Help: "Show power zones.", Children: []Node{
	Leaf{Name: "list", Help: "Show power zones for all sports.", Default: true, Endpoint: "profile.power-zones", Verb: "get power zones", Output: JSON{List: true}},
	Leaf{
		Name: "sport", Help: "Show power zones for one sport.",
		Spec: argsOf(&struct {
			Sport Upper `arg:"" help:"Sport: cycling, running, rowing or cross-country-skiing." enum:"cycling,running,rowing,cross-country-skiing"`
		}{}),
		Endpoint: "profile.sport-power-zones",
		Params:   Params{"sport": In("Sport")},
		Verb:     "get sport power zones", Output: JSON{},
	},
}}

var profilePrivacyGroup = Group{Name: "privacy", Help: "View and change privacy settings.", Children: []Node{
	Leaf{Name: "view", Help: "Show privacy settings.", Default: true, Endpoint: "profile.privacy-settings", Verb: "get privacy settings", Output: JSON{}},
	Custom{Name: "set", Help: "Change who can see your profile, activities, wellness data and badges.", Cmd: &ProfilePrivacySetCmd{}},
}}

// Garmin calls connections "subscribers" in privacy settings and "following" for badges.
var privacyLevels = map[string]struct{ accessRule, badgeVisibility string }{
	"public":      {"public", "public"},
	"private":     {"private", "private"},
	"connections": {"subscribers", "following"},
	"groups":      {"groups", "groups"},
}

// ProfilePrivacySetCmd changes the privacy of the profile, the activities,
// the wellness data and the badges. A nil flag is not given.
type ProfilePrivacySetCmd struct {
	Profile    *string `help:"Who can see your profile." enum:"public,private,connections,groups" placeholder:"LEVEL"`
	Activities *string `help:"Who can see your activities by default. Past activities keep their privacy; change one with activity privacy." enum:"public,private,connections,groups" placeholder:"LEVEL"`
	Wellness   *string `help:"Who can see your wellness data, such as steps." enum:"public,private,connections,groups" placeholder:"LEVEL"`
	Badges     *string `help:"Who can see your badges." enum:"public,private,connections,groups" placeholder:"LEVEL"`
}

type privacyCategory struct {
	flag, key string
	level     *string
}

func (c *ProfilePrivacySetCmd) categories() []privacyCategory {
	return []privacyCategory{
		{"profile", "profilePrivacySettings", c.Profile},
		{"activities", "activityPrivacySettings", c.Activities},
		{"wellness", "wellnessPrivacySettings", c.Wellness},
	}
}

func (c *ProfilePrivacySetCmd) Validate() error {
	if c.Profile == nil && c.Activities == nil && c.Wellness == nil && c.Badges == nil {
		return errors.New("give at least one of --profile, --activities, --wellness or --badges")
	}
	return nil
}

func (c *ProfilePrivacySetCmd) Run(g *Globals) error {
	var given []privacyCategory
	var changed []string
	for _, cat := range c.categories() {
		if cat.level != nil {
			given = append(given, cat)
			changed = append(changed, cat.flag+" "+*cat.level)
		}
	}
	if c.Badges != nil {
		changed = append(changed, "badges "+*c.Badges)
	}

	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	var data json.RawMessage
	if len(given) > 0 {
		if data, err = setPrivacySettings(g, client, given); err != nil {
			return err
		}
	}
	if c.Badges != nil {
		displayName, err := resolveDisplayName(g, client)
		if err != nil {
			return err
		}
		body, err := json.Marshal(map[string]string{"badgeVisibility": privacyLevels[*c.Badges].badgeVisibility})
		if err != nil {
			return err
		}
		data, err = client.Call(g.Context, "profile.privacy.badges", map[string]string{"displayName": displayName}, body)
		if err != nil {
			return fmt.Errorf("change badge privacy: %w", err)
		}
	}
	return reportWrite(g, data, "Changed privacy: %s", strings.Join(changed, ", "))
}

// setPrivacySettings sends back the whole settings answer, as the Garmin web
// settings page does, with the rule of each given category and of all its
// subcategories changed.
func setPrivacySettings(g *Globals, client *garminapi.Client, given []privacyCategory) (json.RawMessage, error) {
	data, err := client.Call(g.Context, "profile.privacy-settings", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("get privacy settings: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var settings map[string]any
	if err := dec.Decode(&settings); err != nil {
		return nil, fmt.Errorf("parse privacy settings: %w", err)
	}
	for _, cat := range given {
		category, ok := settings[cat.key].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("garmin returned no %s in its privacy settings", cat.key)
		}
		rule := privacyLevels[*cat.level].accessRule
		objectAt(category, "categoryDetails")["categoryAccessRule"] = rule
		subs, _ := category["subcategoryPrivacySettings"].([]any)
		for _, item := range subs {
			if sub, ok := item.(map[string]any); ok {
				objectAt(sub, "subcategoryDetails")["accessRule"] = rule
			}
		}
	}
	body, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}
	data, err = client.Call(g.Context, "profile.privacy-settings.update", nil, body)
	if err != nil {
		return nil, fmt.Errorf("change privacy settings: %w", err)
	}
	return data, nil
}

func objectAt(m map[string]any, key string) map[string]any {
	obj, ok := m[key].(map[string]any)
	if !ok {
		obj = map[string]any{}
		m[key] = obj
	}
	return obj
}

var profilePrivacyZonesGroup = Group{Name: "privacy-zones", Help: "List, add and delete privacy zones.", Children: []Node{
	Leaf{Name: "list", Help: "List privacy zones.", Default: true, Endpoint: "profile.privacy-zones", Verb: "get privacy zones", Output: JSON{List: true}},
	Custom{Name: "add", Help: "Add a privacy zone around a point.", Cmd: &ProfilePrivacyZonesAddCmd{}},
	Leaf{
		Name: "delete", Help: "Delete a privacy zone.",
		Spec: argsOf(&struct {
			ID ID `arg:"" help:"Privacy zone ID."`
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
