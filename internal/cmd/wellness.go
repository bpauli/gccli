package cmd

import (
	"fmt"
)

// WellnessCmd groups wellness subcommands.
type WellnessCmd struct {
	MenstrualCycle   WellnessMenstrualCycleCmd   `cmd:"" name:"menstrual-cycle" help:"Show menstrual cycle data."`
	MenstrualSummary WellnessMenstrualSummaryCmd `cmd:"" name:"menstrual-summary" help:"Show menstrual cycle summary."`
	PregnancySummary WellnessPregnancySummaryCmd `cmd:"" name:"pregnancy-summary" help:"Show pregnancy summary."`
}

// WellnessMenstrualCycleCmd shows menstrual cycle data.
type WellnessMenstrualCycleCmd struct {
	StartDate Date `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"start"`
	EndDate   Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"end"`
}

func (c *WellnessMenstrualCycleCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "wellness.menstrual-cycle-data", map[string]string{"startDate": string(c.StartDate), "endDate": string(c.EndDate)}, nil)
	if err != nil {
		return fmt.Errorf("get menstrual cycle data: %w", err)
	}

	return writeHealthJSON(g, data)
}

// WellnessMenstrualSummaryCmd shows menstrual cycle summary.
type WellnessMenstrualSummaryCmd struct {
	StartDate Date `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"start"`
	EndDate   Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"end"`
}

func (c *WellnessMenstrualSummaryCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "wellness.menstrual-cycle-summary", map[string]string{"startDate": string(c.StartDate), "endDate": string(c.EndDate)}, nil)
	if err != nil {
		return fmt.Errorf("get menstrual cycle summary: %w", err)
	}

	return writeHealthJSON(g, data)
}

// WellnessPregnancySummaryCmd shows pregnancy summary.
type WellnessPregnancySummaryCmd struct{}

func (c *WellnessPregnancySummaryCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "wellness.pregnancy-summary", nil, nil)
	if err != nil {
		return fmt.Errorf("get pregnancy summary: %w", err)
	}

	return writeHealthJSON(g, data)
}
