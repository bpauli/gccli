package cmd

import "fmt"

// HealthSPO2Cmd shows SpO2 (blood oxygen) data for a date.
type HealthSPO2Cmd struct {
	Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
}

func (c *HealthSPO2Cmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	date, err := resolveDate(c.Date)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "health.spo2", map[string]string{"date": date}, nil)
	if err != nil {
		return fmt.Errorf("get spo2: %w", err)
	}

	return writeHealthJSON(g, data)
}

// HealthHRVCmd shows heart rate variability data for a date.
type HealthHRVCmd struct {
	Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
}

func (c *HealthHRVCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	date, err := resolveDate(c.Date)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "health.hrv", map[string]string{"date": date}, nil)
	if err != nil {
		return fmt.Errorf("get hrv: %w", err)
	}

	return writeHealthJSON(g, data)
}

var healthBodyBatteryGroup = Group{Name: "body-battery", Help: "Show body battery data.", Children: []Node{
	Custom{Name: "view", Help: "Show body battery data for a date.", Default: true, Cmd: &HealthBodyBatteryViewCmd{}},
	Custom{Name: "range", Help: "Show body battery data for a date range.", Cmd: &HealthBodyBatteryRangeCmd{}},
}}

// HealthBodyBatteryViewCmd shows body battery data for a single date.
type HealthBodyBatteryViewCmd struct {
	Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
}

func (c *HealthBodyBatteryViewCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	date, err := resolveDate(c.Date)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "health.body-battery", map[string]string{"startDate": date, "endDate": date}, nil)
	if err != nil {
		return fmt.Errorf("get body battery: %w", err)
	}

	return writeHealthJSON(g, data)
}

// HealthBodyBatteryRangeCmd shows body battery data for a date range.
type HealthBodyBatteryRangeCmd struct {
	StartDate Date `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"start"`
	EndDate   Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"end"`
}

func (c *HealthBodyBatteryRangeCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "health.body-battery", map[string]string{"startDate": string(c.StartDate), "endDate": string(c.EndDate)}, nil)
	if err != nil {
		return fmt.Errorf("get body battery range: %w", err)
	}

	return writeHealthJSON(g, data)
}

var healthIntensityMinutesGroup = Group{Name: "intensity-minutes", Help: "Show intensity minutes data.", Children: []Node{
	Custom{Name: "view", Help: "Show intensity minutes for a date.", Default: true, Cmd: &HealthIntensityMinutesViewCmd{}},
	Custom{Name: "weekly", Help: "Show weekly intensity minutes for a date range.", Cmd: &HealthIntensityMinutesWeeklyCmd{}},
}}

// HealthIntensityMinutesViewCmd shows intensity minutes for a single date.
type HealthIntensityMinutesViewCmd struct {
	Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
}

func (c *HealthIntensityMinutesViewCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	date, err := resolveDate(c.Date)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "health.intensity-minutes", map[string]string{"date": date}, nil)
	if err != nil {
		return fmt.Errorf("get intensity minutes: %w", err)
	}

	return writeHealthJSON(g, data)
}

// HealthIntensityMinutesWeeklyCmd shows weekly intensity minutes for a date range.
type HealthIntensityMinutesWeeklyCmd struct {
	StartDate Date `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"start"`
	EndDate   Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"end"`
}

func (c *HealthIntensityMinutesWeeklyCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "health.weekly-intensity-minutes", map[string]string{"startDate": string(c.StartDate), "endDate": string(c.EndDate)}, nil)
	if err != nil {
		return fmt.Errorf("get weekly intensity minutes: %w", err)
	}

	return writeHealthJSON(g, data)
}

// HealthTrainingReadinessCmd shows training readiness data for a date.
type HealthTrainingReadinessCmd struct {
	Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
}

func (c *HealthTrainingReadinessCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	date, err := resolveDate(c.Date)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "health.training-readiness", map[string]string{"date": date}, nil)
	if err != nil {
		return fmt.Errorf("get training readiness: %w", err)
	}

	return writeHealthJSON(g, data)
}

// HealthTrainingStatusCmd shows training status data for a date.
type HealthTrainingStatusCmd struct {
	Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
}

func (c *HealthTrainingStatusCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	date, err := resolveDate(c.Date)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "health.training-status", map[string]string{"date": date}, nil)
	if err != nil {
		return fmt.Errorf("get training status: %w", err)
	}

	return writeHealthJSON(g, data)
}

// HealthFitnessAgeCmd shows fitness age data for a date.
type HealthFitnessAgeCmd struct {
	Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
}

func (c *HealthFitnessAgeCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	date, err := resolveDate(c.Date)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "health.fitness-age", map[string]string{"date": date}, nil)
	if err != nil {
		return fmt.Errorf("get fitness age: %w", err)
	}

	return writeHealthJSON(g, data)
}

// HealthMaxMetricsCmd shows VO2max and other max metrics for a date.
type HealthMaxMetricsCmd struct {
	Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
}

func (c *HealthMaxMetricsCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	date, err := resolveDate(c.Date)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "health.max-metrics", map[string]string{"date": date}, nil)
	if err != nil {
		return fmt.Errorf("get max metrics: %w", err)
	}

	return writeHealthJSON(g, data)
}

// HealthLactateThresholdCmd shows the latest lactate threshold data.
type HealthLactateThresholdCmd struct{}

func (c *HealthLactateThresholdCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "health.lactate-threshold", nil, nil)
	if err != nil {
		return fmt.Errorf("get lactate threshold: %w", err)
	}

	return writeHealthJSON(g, data)
}

// HealthCyclingFTPCmd shows the latest cycling functional threshold power.
type HealthCyclingFTPCmd struct{}

func (c *HealthCyclingFTPCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "health.cycling-ftp", nil, nil)
	if err != nil {
		return fmt.Errorf("get cycling ftp: %w", err)
	}

	return writeHealthJSON(g, data)
}

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

// HealthEnduranceScoreCmd shows endurance score data for a date.
type HealthEnduranceScoreCmd struct {
	Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
}

func (c *HealthEnduranceScoreCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	date, err := resolveDate(c.Date)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "health.endurance-score", map[string]string{"date": date}, nil)
	if err != nil {
		return fmt.Errorf("get endurance score: %w", err)
	}

	return writeHealthJSON(g, data)
}

// HealthHillScoreCmd shows hill score data for a date.
type HealthHillScoreCmd struct {
	Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
}

func (c *HealthHillScoreCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	date, err := resolveDate(c.Date)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "health.hill-score", map[string]string{"date": date}, nil)
	if err != nil {
		return fmt.Errorf("get hill score: %w", err)
	}

	return writeHealthJSON(g, data)
}

// HealthEventsCmd shows daily wellness events for a date.
type HealthEventsCmd struct {
	Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
}

func (c *HealthEventsCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	date, err := resolveDate(c.Date)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "health.all-day-events", map[string]string{"date": date}, nil)
	if err != nil {
		return fmt.Errorf("get events: %w", err)
	}

	return writeHealthJSON(g, data)
}

// HealthLifestyleCmd shows lifestyle logging data for a date.
type HealthLifestyleCmd struct {
	Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
}

func (c *HealthLifestyleCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	date, err := resolveDate(c.Date)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "health.lifestyle-logging", map[string]string{"date": date}, nil)
	if err != nil {
		return fmt.Errorf("get lifestyle: %w", err)
	}

	return writeHealthJSON(g, data)
}
