package cmd

import (
	"fmt"
)

var hydrationGroup = Group{Name: "hydration", Help: "Track hydration.", Children: []Node{
	Custom{Name: "view", Help: "Show hydration data for a date.", Default: true, Cmd: &HydrationViewCmd{}},
	Custom{Name: "add", Help: "Add a hydration log entry.", Cmd: &HydrationAddCmd{}},
}}

// HydrationViewCmd shows hydration data for a date.
type HydrationViewCmd struct {
	Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
}

func (c *HydrationViewCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	date, err := resolveDate(c.Date)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "hydration.get", map[string]string{"date": date}, nil)
	if err != nil {
		return fmt.Errorf("get hydration: %w", err)
	}

	return writeHealthJSON(g, data)
}

// HydrationAddCmd adds a hydration log entry.
type HydrationAddCmd struct {
	Amount float64 `arg:"" help:"Amount of water in milliliters."`
	Date   string  `help:"Date to log for (YYYY-MM-DD). Defaults to today." default:""`
}

func (c *HydrationAddCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	date, err := resolveDate(c.Date)
	if err != nil {
		return err
	}

	data, err := client.AddHydration(g.Context, date, c.Amount)
	if err != nil {
		return fmt.Errorf("add hydration: %w", err)
	}
	return reportWrite(g, data, "Added %g ml of water on %s", c.Amount, date)
}
