package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/bpauli/gccli/internal/garminapi"
)

// GearCmd groups gear subcommands.
type GearCmd struct {
	List       GearListCmd       `cmd:"" default:"withargs" help:"List all gear."`
	Stats      GearStatsCmd      `cmd:"" help:"Show usage statistics for a gear item."`
	Activities GearActivitiesCmd `cmd:"" help:"Show activities linked to a gear item."`
	Defaults   GearDefaultsCmd   `cmd:"" help:"Show default gear per activity type."`
	Link       GearLinkCmd       `cmd:"" help:"Link gear to an activity."`
	Unlink     GearUnlinkCmd     `cmd:"" help:"Unlink gear from an activity."`
}

// GearListCmd lists all gear for the authenticated user.
type GearListCmd struct{}

func (c *GearListCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	profilePK, err := getUserProfilePK(g, client)
	if err != nil {
		return err
	}

	data, err := client.GetGear(g.Context, profilePK)
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

	data, err := client.GetGearStats(g.Context, c.UUID)
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

	data, err := client.GetGearActivities(g.Context, c.UUID, c.Limit)
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

	profilePK, err := getUserProfilePK(g, client)
	if err != nil {
		return err
	}

	data, err := client.GetGearDefaults(g.Context, profilePK)
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

	_, err = client.LinkGear(g.Context, c.UUID, c.ActivityID)
	if err != nil {
		return fmt.Errorf("link gear: %w", err)
	}

	g.UI.Successf("Linked gear %s to activity %s", c.UUID, c.ActivityID)
	return nil
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

	_, err = client.UnlinkGear(g.Context, c.UUID, c.ActivityID)
	if err != nil {
		return fmt.Errorf("unlink gear: %w", err)
	}

	g.UI.Successf("Unlinked gear %s from activity %s", c.UUID, c.ActivityID)
	return nil
}

// getUserProfilePK fetches the authenticated user's social profile and extracts
// the userProfileNumber needed by gear API calls.
func getUserProfilePK(g *Globals, client *garminapi.Client) (string, error) {
	data, err := client.GetSocialProfile(g.Context)
	if err != nil {
		return "", fmt.Errorf("get user profile: %w", err)
	}

	// Use json.Decoder with UseNumber to preserve numeric precision.
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()

	var profile map[string]any
	if err := dec.Decode(&profile); err != nil {
		return "", fmt.Errorf("parse user profile: %w", err)
	}

	pk, ok := profile["userProfileNumber"]
	if !ok {
		return "", fmt.Errorf("user profile missing userProfileNumber field")
	}

	return fmt.Sprintf("%v", pk), nil
}
