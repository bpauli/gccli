package cmd

import "fmt"

var healthBodyBatteryGroup = Group{Name: "body-battery", Help: "Show body battery data.", Children: []Node{
	Leaf{
		Name: "view", Help: "Show body battery data for a date.", Default: true,
		Spec: argsOf(&struct {
			Date Date `arg:"" optional:"" default:"today" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
		}{}),
		Endpoint: "health.body-battery",
		Params:   Params{"startDate": In("Date"), "endDate": In("Date")},
		Verb:     "get body battery", Output: JSON{},
	},
	Leaf{
		Name: "range", Help: "Show body battery data for a date range.",
		Spec: argsOf(&struct {
			StartDate Date `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"start"`
			EndDate   Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"end"`
		}{}),
		Endpoint: "health.body-battery",
		Params:   Params{"startDate": In("StartDate"), "endDate": In("EndDate")},
		Verb:     "get body battery range", Output: JSON{},
	},
}}

var healthIntensityMinutesGroup = Group{Name: "intensity-minutes", Help: "Show intensity minutes data.", Children: []Node{
	Leaf{
		Name: "view", Help: "Show intensity minutes for a date.", Default: true,
		Spec: argsOf(&struct {
			Date Date `arg:"" optional:"" default:"today" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
		}{}),
		Endpoint: "health.intensity-minutes",
		Params:   Params{"date": In("Date")},
		Verb:     "get intensity minutes", Output: JSON{},
	},
	Leaf{
		Name: "weekly", Help: "Show weekly intensity minutes for a date range.",
		Spec: argsOf(&struct {
			StartDate Date `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"start"`
			EndDate   Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"end"`
		}{}),
		Endpoint: "health.weekly-intensity-minutes",
		Params:   Params{"startDate": In("StartDate"), "endDate": In("EndDate")},
		Verb:     "get weekly intensity minutes", Output: JSON{},
	},
}}

var healthRacePredictionsGroup = Group{Name: "race-predictions", Help: "Show race predictions.", Children: []Node{
	Custom{Name: "view", Help: "Show current race predictions.", Default: true, Cmd: &HealthRacePredictionsViewCmd{}},
	Custom{Name: "range", Help: "Show race predictions for a date range.", Cmd: &HealthRacePredictionsRangeCmd{}},
}}

// HealthRacePredictionsViewCmd shows current race predictions.
type HealthRacePredictionsViewCmd struct {
	Date string `arg:"" optional:"" help:"Unused. Current predictions are always returned."`
}

func (c *HealthRacePredictionsViewCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.GetRacePredictions(g.Context, "", "")
	if err != nil {
		return fmt.Errorf("get race predictions: %w", err)
	}

	return writeHealthJSON(g, data)
}

// HealthRacePredictionsRangeCmd shows race predictions for a date range.
type HealthRacePredictionsRangeCmd struct {
	StartDate Date `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"start"`
	EndDate   Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"end"`
}

func (c *HealthRacePredictionsRangeCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.GetRacePredictions(g.Context, string(c.StartDate), string(c.EndDate))
	if err != nil {
		return fmt.Errorf("get race predictions range: %w", err)
	}

	return writeHealthJSON(g, data)
}
