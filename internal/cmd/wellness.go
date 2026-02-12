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
	StartDate string `help:"Start date (YYYY-MM-DD)." required:""`
	EndDate   string `help:"End date (YYYY-MM-DD)." required:""`
}

func (c *WellnessMenstrualCycleCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.GetMenstrualCycleData(g.Context, c.StartDate, c.EndDate)
	if err != nil {
		return fmt.Errorf("get menstrual cycle data: %w", err)
	}

	return writeHealthJSON(g, data)
}

// WellnessMenstrualSummaryCmd shows menstrual cycle summary.
type WellnessMenstrualSummaryCmd struct {
	StartDate string `help:"Start date (YYYY-MM-DD)." required:""`
	EndDate   string `help:"End date (YYYY-MM-DD)." required:""`
}

func (c *WellnessMenstrualSummaryCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.GetMenstrualCycleSummary(g.Context, c.StartDate, c.EndDate)
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

	data, err := client.GetPregnancySummary(g.Context)
	if err != nil {
		return fmt.Errorf("get pregnancy summary: %w", err)
	}

	return writeHealthJSON(g, data)
}
