package cmd

import (
	"fmt"
)

var wellnessGroup = Group{Name: "wellness", Help: "View wellness data.", Children: []Node{
	Custom{Name: "menstrual-cycle", Help: "Show menstrual cycle data.", Cmd: &WellnessMenstrualCycleCmd{}},
	Custom{Name: "menstrual-summary", Help: "Show menstrual cycle summary.", Cmd: &WellnessMenstrualSummaryCmd{}},
	Custom{Name: "pregnancy-summary", Help: "Show pregnancy summary.", Cmd: &WellnessPregnancySummaryCmd{}},
}}

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
