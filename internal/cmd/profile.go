package cmd

import (
	"fmt"
)

// ProfileCmd groups profile subcommands.
type ProfileCmd struct {
	View     ProfileViewCmd     `cmd:"" default:"withargs" help:"Show user profile."`
	Settings ProfileSettingsCmd `cmd:"" help:"Show user settings."`
}

// ProfileViewCmd shows the user profile.
type ProfileViewCmd struct{}

func (c *ProfileViewCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.GetProfile(g.Context)
	if err != nil {
		return fmt.Errorf("get profile: %w", err)
	}

	return writeHealthJSON(g, data)
}

// ProfileSettingsCmd shows user settings.
type ProfileSettingsCmd struct{}

func (c *ProfileSettingsCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.GetUserSettings(g.Context)
	if err != nil {
		return fmt.Errorf("get user settings: %w", err)
	}

	return writeHealthJSON(g, data)
}
