package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var activitiesPrivacyGroup = Group{Name: "privacy", Help: "Show or change the privacy of all past activities.", Children: []Node{
	Leaf{
		Name: "status", Help: "Show the last privacy change of past activities.", Default: true,
		Endpoint: "activities.bulk-privacy",
		Verb:     "get privacy change", Output: JSON{},
	},
	Custom{Name: "set", Help: "Change the privacy of all past activities, or of past activities of some types.", Cmd: &ActivitiesPrivacySetCmd{}},
}}

// ActivitiesPrivacySetCmd changes the privacy of past activities. A nil Type
// means all activity types. Kong leaves Type nil when --type is not given
// and makes it an empty list for --type "", which Validate rejects, so an
// empty value never widens the change to all activities.
type ActivitiesPrivacySetCmd struct {
	Level string   `arg:"" help:"Who can see the activities: public, private or connections." enum:"public,private,connections"`
	Type  []string `help:"Top-level activity type key (e.g. running, cycling). Repeat the flag or separate with commas. Default: all activity types." placeholder:"KEY"`
	Force bool     `help:"Skip confirmation prompt." short:"f"`
}

func (c *ActivitiesPrivacySetCmd) Help() string {
	return "Garmin applies the change within minutes. activities privacy status shows the progress. To change one activity, use activity privacy."
}

func (c *ActivitiesPrivacySetCmd) Validate() error {
	if c.Type != nil && len(c.Type) == 0 {
		return errors.New("--type must not be empty")
	}
	for _, t := range c.Type {
		if strings.TrimSpace(t) == "" {
			return errors.New("--type must not be empty")
		}
	}
	return nil
}

func (c *ActivitiesPrivacySetCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	scope := "all past activities"
	if len(c.Type) > 0 {
		scope = "past " + strings.Join(c.Type, ", ") + " activities"
	}
	ok, err := confirm(g.Stderr, fmt.Sprintf("Change the privacy of %s to %s?", scope, c.Level), c.Force)
	if err != nil {
		return err
	}
	if !ok {
		g.UI.Infof("Cancelled")
		return nil
	}

	rule := activityPrivacy[c.Level]["typeKey"]
	updates := map[string]any{}
	if len(c.Type) == 0 {
		updates["all"] = rule
	}
	for _, t := range c.Type {
		updates[t] = rule
	}
	body, err := json.Marshal(map[string]any{
		"createDate": nowFn().UTC().Format("2006-01-02T15:04:05.000Z"),
		"updates":    updates,
	})
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "activities.set-bulk-privacy", nil, body)
	if err != nil {
		return fmt.Errorf("change privacy of past activities: %w", err)
	}
	return reportWrite(g, data, "Requested privacy %s for %s", c.Level, scope)
}
