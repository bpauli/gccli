package cmd

import (
	"fmt"
)

var hydrationGroup = Group{Name: "hydration", Help: "Track hydration.", Children: []Node{
	Leaf{
		Name: "view", Help: "Show hydration data for a date.", Default: true,
		Spec: argsOf(&struct {
			Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
		}{}),
		Endpoint: "hydration.get",
		Params:   Params{"date": In("Date")},
		Verb:     "get hydration", Output: JSON{},
		LateDates: []string{"Date"},
	},
	Leaf{
		Name: "daily", Help: "Show daily hydration statistics for a date range.",
		Spec: argsOf(&struct {
			StartDate Date `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"start"`
			EndDate   Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"end"`
		}{}),
		Endpoint: "hydration.daily",
		Params:   Params{"startDate": In("StartDate"), "endDate": In("EndDate")},
		Verb:     "get daily hydration", Output: JSON{List: true},
	},
	Leaf{
		Name: "weekly", Help: "Show weekly hydration statistics.",
		Spec: argsOf(&struct {
			EndDate Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today." name:"end"`
			Weeks   int  `help:"Number of weeks." default:"4" name:"weeks"`
		}{}),
		Endpoint: "hydration.weekly",
		Params:   Params{"endDate": In("EndDate").Or(Today), "weeks": In("Weeks")},
		Verb:     "get weekly hydration", Output: JSON{List: true},
	},
	Custom{Name: "add", Help: "Add a hydration log entry.", Cmd: &HydrationAddCmd{}},
}}

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
