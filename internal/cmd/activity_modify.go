package cmd

import (
	"encoding/json"
	"fmt"
)

// ActivityRenameCmd renames an activity.
type ActivityRenameCmd struct {
	ID   string `arg:"" help:"Activity ID."`
	Name string `arg:"" help:"New activity name."`
}

func (c *ActivityRenameCmd) Run(g *Globals) error {
	data, err := updateActivity(g, c.ID, "rename activity", map[string]any{"activityName": c.Name})
	if err != nil {
		return err
	}
	return reportWrite(g, data, "Renamed activity %s to %q", c.ID, c.Name)
}

// ActivityRetypeCmd changes the type of an activity.
type ActivityRetypeCmd struct {
	ID           string `arg:"" help:"Activity ID."`
	TypeID       int    `help:"Activity type ID." required:"" name:"type-id"`
	TypeKey      string `help:"Activity type key (e.g. running, cycling)." required:"" name:"type-key"`
	ParentTypeID int    `help:"Parent activity type ID." default:"0" name:"parent-type-id"`
}

func (c *ActivityRetypeCmd) Run(g *Globals) error {
	fields := map[string]any{"activityTypeDTO": map[string]any{
		"typeId":       c.TypeID,
		"typeKey":      c.TypeKey,
		"parentTypeId": c.ParentTypeID,
	}}
	data, err := updateActivity(g, c.ID, "retype activity", fields)
	if err != nil {
		return err
	}
	return reportWrite(g, data, "Changed activity %s type to %s", c.ID, c.TypeKey)
}

// ActivityDescribeCmd sets or clears the description of an activity.
type ActivityDescribeCmd struct {
	ID   string `arg:"" help:"Activity ID."`
	Text string `arg:"" help:"New description. An empty text (\"\") clears it."`
}

func (c *ActivityDescribeCmd) Run(g *Globals) error {
	var description any = c.Text
	if c.Text == "" {
		description = nil
	}
	data, err := updateActivity(g, c.ID, "describe activity", map[string]any{"description": description})
	if err != nil {
		return err
	}
	if c.Text == "" {
		return reportWrite(g, data, "Cleared activity %s description", c.ID)
	}
	return reportWrite(g, data, "Changed activity %s description to %q", c.ID, c.Text)
}

// activityPrivacy maps each privacy word of the CLI to Garmin's access rule.
// Garmin calls the connections level "subscribers".
var activityPrivacy = map[string]map[string]any{
	"public":      {"typeId": 1, "typeKey": "public"},
	"private":     {"typeId": 2, "typeKey": "private"},
	"connections": {"typeId": 3, "typeKey": "subscribers"},
}

// ActivityPrivacyCmd changes who can see an activity.
type ActivityPrivacyCmd struct {
	ID    string `arg:"" help:"Activity ID."`
	Level string `arg:"" help:"Who can see the activity: public, private or connections." enum:"public,private,connections"`
}

func (c *ActivityPrivacyCmd) Run(g *Globals) error {
	data, err := updateActivity(g, c.ID, "change activity privacy", map[string]any{"accessControlRuleDTO": activityPrivacy[c.Level]})
	if err != nil {
		return err
	}
	return reportWrite(g, data, "Changed activity %s privacy to %s", c.ID, c.Level)
}

// ActivityFavoriteCmd marks an activity as a favorite.
type ActivityFavoriteCmd struct {
	ID string `arg:"" help:"Activity ID."`
}

func (c *ActivityFavoriteCmd) Run(g *Globals) error {
	data, err := updateActivity(g, c.ID, "favorite activity", map[string]any{"metadataDTO": map[string]any{"favorite": true}})
	if err != nil {
		return err
	}
	return reportWrite(g, data, "Added activity %s to favorites", c.ID)
}

// ActivityUnfavoriteCmd removes the favorite mark from an activity.
type ActivityUnfavoriteCmd struct {
	ID string `arg:"" help:"Activity ID."`
}

func (c *ActivityUnfavoriteCmd) Run(g *Globals) error {
	data, err := updateActivity(g, c.ID, "unfavorite activity", map[string]any{"metadataDTO": map[string]any{"favorite": false}})
	if err != nil {
		return err
	}
	return reportWrite(g, data, "Removed activity %s from favorites", c.ID)
}

func updateActivity(g *Globals, id, verb string, fields map[string]any) (json.RawMessage, error) {
	client, err := resolveClient(g)
	if err != nil {
		return nil, err
	}

	data, err := client.UpdateActivity(g.Context, id, fields)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", verb, err)
	}
	return data, nil
}
