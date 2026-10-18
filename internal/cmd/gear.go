package cmd

import (
	"fmt"
	"strconv"
)

var gearGroup = Group{Name: "gear", Help: "Manage gear.", Children: []Node{
	Custom{Name: "list", Help: "List all gear.", Default: true, Cmd: &GearListCmd{}},
	Custom{Name: "stats", Help: "Show usage statistics for a gear item.", Cmd: &GearStatsCmd{}},
	Custom{Name: "activities", Help: "Show activities linked to a gear item.", Cmd: &GearActivitiesCmd{}},
	Custom{Name: "defaults", Help: "Show default gear per activity type.", Cmd: &GearDefaultsCmd{}},
	Custom{Name: "link", Help: "Link gear to an activity.", Cmd: &GearLinkCmd{}},
	Custom{Name: "unlink", Help: "Unlink gear from an activity.", Cmd: &GearUnlinkCmd{}},
}}

// GearListCmd lists all gear for the authenticated user.
type GearListCmd struct{}

func (c *GearListCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	profilePK, err := client.GetProfilePK(g.Context)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "gear.list", map[string]string{"userProfilePK": profilePK}, nil)
	if err != nil {
		return fmt.Errorf("get gear: %w", err)
	}

	return writeHealthJSON(g, data)
}

// GearStatsCmd shows usage statistics for a gear item.
type GearStatsCmd struct {
	UUID string `arg:"" help:"Gear UUID."`
}

func (c *GearStatsCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "gear.stats", map[string]string{"gearUUID": c.UUID}, nil)
	if err != nil {
		return fmt.Errorf("get gear stats: %w", err)
	}

	return writeHealthJSON(g, data)
}

// GearActivitiesCmd shows activities linked to a gear item.
type GearActivitiesCmd struct {
	UUID  string `arg:"" help:"Gear UUID."`
	Limit int    `help:"Maximum number of activities to return." default:"20"`
}

func (c *GearActivitiesCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "gear.activities", map[string]string{"gearUUID": c.UUID, "limit": strconv.Itoa(c.Limit)}, nil)
	if err != nil {
		return fmt.Errorf("get gear activities: %w", err)
	}

	return writeHealthJSON(g, data)
}

// GearDefaultsCmd shows default gear per activity type.
type GearDefaultsCmd struct{}

func (c *GearDefaultsCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	profilePK, err := client.GetProfilePK(g.Context)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "gear.defaults", map[string]string{"userProfileNumber": profilePK}, nil)
	if err != nil {
		return fmt.Errorf("get gear defaults: %w", err)
	}

	return writeHealthJSON(g, data)
}

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
