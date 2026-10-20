package cmd

import (
	"fmt"
	"time"

	"github.com/bpauli/gccli/internal/fit"
)

var bodyGroup = Group{Name: "body", Help: "Body composition and weight data.", Children: []Node{
	Custom{Name: "composition", Help: "Show body composition data.", Default: true, Cmd: &BodyCompositionCmd{}},
	Leaf{
		Name: "weigh-ins", Help: "Show weigh-in data for a date range.",
		Spec: argsOf(&struct {
			StartDate Date `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"start"`
			EndDate   Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"end"`
		}{}),
		Endpoint: "body.weigh-ins",
		Params:   Params{"startDate": In("StartDate"), "endDate": In("EndDate")},
		Verb:     "get weigh-ins", Output: JSON{},
	},
	Leaf{
		Name: "daily-weigh-ins", Help: "Show weigh-in data for a single day.",
		Spec: argsOf(&struct {
			Date Date `arg:"" optional:"" default:"today" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
		}{}),
		Endpoint: "body.daily-weigh-ins",
		Params:   Params{"date": In("Date")},
		Verb:     "get daily weigh-ins", Output: JSON{},
	},
	Leaf{
		Name: "latest", Help: "Show the latest weigh-in as of a date.",
		Spec: argsOf(&struct {
			Date Date `arg:"" optional:"" default:"today" help:"Date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today."`
		}{}),
		Endpoint: "body.latest-weight",
		Params:   Params{"date": In("Date")},
		Verb:     "get latest weigh-in", Output: JSON{},
	},
	Custom{Name: "weekly", Help: "Show weekly weight summaries for the 52 weeks that end on a date.", Cmd: &BodyWeeklyCmd{}},
	Custom{Name: "add-weight", Help: "Add a weigh-in entry.", Cmd: &BodyAddWeightCmd{}},
	Custom{Name: "add-composition", Help: "Add body composition data (weight, body fat, muscle mass, etc.) via FIT upload.", Cmd: &BodyAddCompositionCmd{}},
	Leaf{
		Name: "delete-weight", Help: "Delete a weigh-in entry.",
		Spec: argsOf(&struct {
			PK   string `arg:"" help:"Weigh-in version (primary key)."`
			Date string `help:"Date of the weigh-in (YYYY-MM-DD)." required:"" name:"date"`
		}{}),
		Endpoint: "body.delete-weight",
		Params:   Params{"date": In("Date"), "version": In("PK")},
		Verb:     "delete weight",
		Output:   Write{Done: "Deleted weigh-in {PK}", Confirm: "Delete weigh-in {PK} on {Date}?"},
	},
	Leaf{
		Name: "blood-pressure", Help: "Show blood pressure data.",
		Spec: argsOf(&struct {
			StartDate Date `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"start"`
			EndDate   Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"end"`
		}{}),
		Endpoint: "body.blood-pressure",
		Params:   Params{"startDate": In("StartDate"), "endDate": In("EndDate")},
		Verb:     "get blood pressure", Output: JSON{},
	},
	Custom{Name: "add-blood-pressure", Help: "Add a blood pressure measurement.", Cmd: &BodyAddBPCmd{}},
	Leaf{
		Name: "delete-blood-pressure", Help: "Delete a blood pressure entry.",
		Spec: argsOf(&struct {
			Version string `arg:"" help:"Blood pressure entry version."`
			Date    string `help:"Date of the measurement (YYYY-MM-DD)." required:"" name:"date"`
		}{}),
		Endpoint: "body.delete-blood-pressure",
		Params:   Params{"date": In("Date"), "version": In("Version")},
		Verb:     "delete blood pressure",
		Output:   Write{Done: "Deleted blood pressure entry {Version}", Confirm: "Delete blood pressure entry {Version} on {Date}?"},
	},
	Leaf{
		Name: "glucose", Help: "Show glucose readings for a date.",
		Spec: argsOf(&struct {
			Date Date `arg:"" optional:"" default:"today" help:"Date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today."`
		}{}),
		Endpoint: "body.glucose",
		Params:   Params{"date": In("Date")},
		Verb:     "get glucose", Output: JSON{},
	},
}}

// BodyCompositionCmd shows body composition data for a date or range.
type BodyCompositionCmd struct {
	Date      Date `arg:"" optional:"" default:"today" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
	StartDate Date `help:"Start date for range query (YYYY-MM-DD, today, yesterday, 3d, +30d)." name:"start"`
	EndDate   Date `help:"End date for range query (YYYY-MM-DD, today, yesterday, 3d, +30d)." name:"end"`
}

func (c *BodyCompositionCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	startDate, endDate := string(c.Date), string(c.Date)
	if c.StartDate != "" && c.EndDate != "" {
		startDate = string(c.StartDate)
		endDate = string(c.EndDate)
	}

	data, err := client.Call(g.Context, "body.composition", map[string]string{"startDate": startDate, "endDate": endDate}, nil)
	if err != nil {
		return fmt.Errorf("get body composition: %w", err)
	}

	return writeHealthJSON(g, data)
}

// BodyWeeklyCmd shows the weekly weight summaries of the 52 weeks that end
// on a date.
type BodyWeeklyCmd struct {
	End Date `arg:"" optional:"" default:"today" help:"Last day of the 52 weeks (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today."`
}

var weeklyWeightRows = Rows{Noun: "weekly weight", Items: "weeklyWeightSummaries", Columns: []Column{
	{Label: "START", Path: "startDate"},
	{Label: "END", Path: "endDate"},
	{Label: "MIN", Path: "minWeight", Format: Weight},
	{Label: "MAX", Path: "maxWeight", Format: Weight},
	{Label: "AVERAGE", Path: "averageWeight.weight", Format: Weight},
}}

func (c *BodyWeeklyCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	start, err := weeklyWeightStart(c.End)
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "body.weekly-weight", map[string]string{"startDate": start, "endDate": string(c.End)}, nil)
	if err != nil {
		return fmt.Errorf("get weekly weight: %w", err)
	}
	return weeklyWeightRows.print(g, client, data)
}

// weeklyWeightStart returns the first day of the window that ends on end.
// Garmin answers 400 unless the window is 52 weeks, both ends counted.
func weeklyWeightStart(end Date) (string, error) {
	t, err := time.Parse("2006-01-02", string(end))
	if err != nil {
		return "", fmt.Errorf("parse end date: %w", err)
	}
	return t.AddDate(0, 0, -(52*7 - 1)).Format("2006-01-02"), nil
}

// BodyAddWeightCmd adds a weigh-in entry.
type BodyAddWeightCmd struct {
	Value float64 `arg:"" help:"Weight value."`
	Unit  string  `help:"Unit: kg or lbs." default:"kg" enum:"kg,lbs" name:"unit"`
}

func (c *BodyAddWeightCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	now := nowFn()
	localTS := now.Format("2006-01-02T15:04:05")
	gmtTS := now.UTC().Format("2006-01-02T15:04:05")

	data, err := client.AddWeight(g.Context, c.Value, c.Unit, localTS, gmtTS)
	if err != nil {
		return fmt.Errorf("add weight: %w", err)
	}

	return reportWrite(g, data, "Added weight: %.1f %s", c.Value, c.Unit)
}

// BodyAddCompositionCmd adds body composition data via FIT file upload.
type BodyAddCompositionCmd struct {
	Weight            float64  `arg:"" help:"Weight in kg."`
	BodyFat           *float64 `help:"Body fat percentage." name:"body-fat"`
	PercentHydration  *float64 `help:"Hydration percentage." name:"percent-hydration"`
	VisceralFatMass   *float64 `help:"Visceral fat mass in kg." name:"visceral-fat-mass"`
	BoneMass          *float64 `help:"Bone mass in kg." name:"bone-mass"`
	MuscleMass        *float64 `help:"Muscle mass in kg." name:"muscle-mass"`
	BasalMet          *float64 `help:"Basal metabolic rate (kcal/day)." name:"basal-met"`
	ActiveMet         *float64 `help:"Active metabolic rate (kcal/day)." name:"active-met"`
	PhysiqueRating    *float64 `help:"Physique rating." name:"physique-rating"`
	MetabolicAge      *float64 `help:"Metabolic age." name:"metabolic-age"`
	VisceralFatRating *float64 `help:"Visceral fat rating." name:"visceral-fat-rating"`
	BMI               *float64 `help:"Body mass index." name:"bmi"`
}

func (c *BodyAddCompositionCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	now := nowFn()
	fitData := fit.EncodeWeightScale(fit.WeightScaleData{
		Timestamp:         now,
		Weight:            c.Weight,
		PercentFat:        c.BodyFat,
		PercentHydration:  c.PercentHydration,
		VisceralFatMass:   c.VisceralFatMass,
		BoneMass:          c.BoneMass,
		MuscleMass:        c.MuscleMass,
		BasalMet:          c.BasalMet,
		ActiveMet:         c.ActiveMet,
		PhysiqueRating:    c.PhysiqueRating,
		MetabolicAge:      c.MetabolicAge,
		VisceralFatRating: c.VisceralFatRating,
		BMI:               c.BMI,
	})

	data, err := client.UploadBodyComposition(g.Context, fitData)
	if err != nil {
		return fmt.Errorf("upload body composition: %w", err)
	}

	return reportWrite(g, data, "Added body composition: %.1f kg", c.Weight)
}

// BodyAddBPCmd adds a blood pressure measurement.
type BodyAddBPCmd struct {
	Systolic  int    `help:"Systolic pressure (mmHg)." required:"" name:"systolic"`
	Diastolic int    `help:"Diastolic pressure (mmHg)." required:"" name:"diastolic"`
	Pulse     int    `help:"Pulse rate (bpm)." required:"" name:"pulse"`
	Notes     string `help:"Optional notes." name:"notes"`
}

func (c *BodyAddBPCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	now := nowFn().UTC()
	ts := now.Format(time.RFC3339Nano)

	data, err := client.AddBloodPressure(g.Context, c.Systolic, c.Diastolic, c.Pulse, ts, ts, c.Notes)
	if err != nil {
		return fmt.Errorf("add blood pressure: %w", err)
	}

	return reportWrite(g, data, "Added blood pressure: %d/%d mmHg, pulse %d bpm", c.Systolic, c.Diastolic, c.Pulse)
}
