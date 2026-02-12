package cmd

import (
	"fmt"
)

// ReloadCmd requests a data reload for a given date.
type ReloadCmd struct {
	Date string `arg:"" optional:"" help:"Date to reload (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
}

func (c *ReloadCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	date, err := resolveDate(c.Date)
	if err != nil {
		return err
	}

	data, err := client.RequestReload(g.Context, date)
	if err != nil {
		return fmt.Errorf("request reload: %w", err)
	}

	return writeHealthJSON(g, data)
}
