package cmd

import (
	"fmt"
)

var profileGroup = Group{Name: "profile", Help: "View user profile.", Children: []Node{
	Custom{Name: "view", Help: "Show user profile.", Default: true, Cmd: &ProfileViewCmd{}},
	Custom{Name: "settings", Help: "Show user settings.", Cmd: &ProfileSettingsCmd{}},
}}

// ProfileViewCmd shows the user profile.
type ProfileViewCmd struct{}

func (c *ProfileViewCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "profile.get", nil, nil)
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

	data, err := client.Call(g.Context, "profile.user-settings", nil, nil)
	if err != nil {
		return fmt.Errorf("get user settings: %w", err)
	}

	return writeHealthJSON(g, data)
}
