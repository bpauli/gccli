package cmd

import (
	"fmt"
)

var gearGroup = Group{Name: "gear", Help: "Manage gear.", Children: []Node{
	Leaf{
		Name: "list", Help: "List all gear.", Default: true,
		Endpoint: "gear.list",
		Params:   Params{"userProfilePK": ProfilePK},
		Verb:     "get gear", Output: JSON{},
	},
	Leaf{
		Name: "stats", Help: "Show usage statistics for a gear item.",
		Spec: argsOf(&struct {
			UUID string `arg:"" help:"Gear UUID."`
		}{}),
		Endpoint: "gear.stats",
		Params:   Params{"gearUUID": In("UUID")},
		Verb:     "get gear stats", Output: JSON{},
	},
	Leaf{
		Name: "activities", Help: "Show activities linked to a gear item.",
		Spec: argsOf(&struct {
			UUID  string `arg:"" help:"Gear UUID."`
			Limit int    `help:"Maximum number of activities to return." default:"20"`
		}{}),
		Endpoint: "gear.activities",
		Params:   Params{"gearUUID": In("UUID"), "limit": In("Limit")},
		Verb:     "get gear activities", Output: JSON{},
	},
	Leaf{
		Name: "defaults", Help: "Show default gear per activity type.",
		Endpoint: "gear.defaults",
		Params:   Params{"userProfileNumber": ProfilePK},
		Verb:     "get gear defaults", Output: JSON{},
	},
	Custom{Name: "link", Help: "Link gear to an activity.", Cmd: &GearLinkCmd{}},
	Custom{Name: "unlink", Help: "Unlink gear from an activity.", Cmd: &GearUnlinkCmd{}},
}}

// GearLinkCmd links a gear item to an activity.
type GearLinkCmd struct {
	UUID       string `arg:"" help:"Gear UUID."`
	ActivityID string `arg:"" help:"Activity ID." name:"activity-id"`
}

func (c *GearLinkCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "gear.link", map[string]string{"gearUUID": c.UUID, "activityID": c.ActivityID}, nil)
	if err != nil {
		return fmt.Errorf("link gear: %w", err)
	}
	return reportWrite(g, data, "Linked gear %s to activity %s", c.UUID, c.ActivityID)
}

// GearUnlinkCmd unlinks a gear item from an activity.
type GearUnlinkCmd struct {
	UUID       string `arg:"" help:"Gear UUID."`
	ActivityID string `arg:"" help:"Activity ID." name:"activity-id"`
}

func (c *GearUnlinkCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "gear.unlink", map[string]string{"gearUUID": c.UUID, "activityID": c.ActivityID}, nil)
	if err != nil {
		return fmt.Errorf("unlink gear: %w", err)
	}
	return reportWrite(g, data, "Unlinked gear %s from activity %s", c.UUID, c.ActivityID)
}
