package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/bpauli/gccli/internal/fit"
	"github.com/bpauli/gccli/internal/outfmt"
)

// BodyCmd groups body composition subcommands.
type BodyCmd struct {
	Composition    BodyCompositionCmd    `cmd:"" default:"withargs" help:"Show body composition data."`
	WeighIns       BodyWeighInsCmd       `cmd:"" name:"weigh-ins" help:"Show weigh-in data for a date range."`
	DailyWeighIns  BodyDailyWeighInsCmd  `cmd:"" name:"daily-weigh-ins" help:"Show weigh-in data for a single day."`
	AddWeight      BodyAddWeightCmd      `cmd:"" name:"add-weight" help:"Add a weigh-in entry."`
	AddComposition BodyAddCompositionCmd `cmd:"" name:"add-composition" help:"Add body composition data (weight, body fat, muscle mass, etc.) via FIT upload."`
	DeleteWeight   BodyDeleteWeightCmd   `cmd:"" name:"delete-weight" help:"Delete a weigh-in entry."`
	BloodPressure  BodyBloodPressureCmd  `cmd:"" name:"blood-pressure" help:"Show blood pressure data."`
	AddBP          BodyAddBPCmd          `cmd:"" name:"add-blood-pressure" help:"Add a blood pressure measurement."`
	DeleteBP       BodyDeleteBPCmd       `cmd:"" name:"delete-blood-pressure" help:"Delete a blood pressure entry."`
}

// BodyCompositionCmd shows body composition data for a date or range.
type BodyCompositionCmd struct {
	Date      string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
	StartDate string `help:"Start date for range query (YYYY-MM-DD)." name:"start"`
	EndDate   string `help:"End date for range query (YYYY-MM-DD)." name:"end"`
}

func (c *BodyCompositionCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	var startDate, endDate string
	if c.StartDate != "" && c.EndDate != "" {
		startDate = c.StartDate
		endDate = c.EndDate
	} else {
		date, err := resolveDate(c.Date)
		if err != nil {
			return err
		}
		startDate = date
		endDate = date
	}

	data, err := client.GetBodyComposition(g.Context, startDate, endDate)
	if err != nil {
		return fmt.Errorf("get body composition: %w", err)
	}

	return writeHealthJSON(g, data)
}

// BodyWeighInsCmd shows weigh-in data for a date range.
type BodyWeighInsCmd struct {
	StartDate string `help:"Start date (YYYY-MM-DD)." required:"" name:"start"`
	EndDate   string `help:"End date (YYYY-MM-DD)." required:"" name:"end"`
}

func (c *BodyWeighInsCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.GetWeighIns(g.Context, c.StartDate, c.EndDate)
	if err != nil {
		return fmt.Errorf("get weigh-ins: %w", err)
	}

	return writeHealthJSON(g, data)
}

// BodyDailyWeighInsCmd shows weigh-in data for a single day.
type BodyDailyWeighInsCmd struct {
	Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
}

func (c *BodyDailyWeighInsCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	date, err := resolveDate(c.Date)
	if err != nil {
		return err
	}

	data, err := client.GetDailyWeighIns(g.Context, date)
	if err != nil {
		return fmt.Errorf("get daily weigh-ins: %w", err)
	}

	return writeHealthJSON(g, data)
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

	if outfmt.IsJSON(g.Context) {
		return outfmt.WriteJSON(os.Stdout, json.RawMessage(data))
	}

	g.UI.Successf("Added weight: %.1f %s", c.Value, c.Unit)
	return nil
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

	if outfmt.IsJSON(g.Context) {
		return outfmt.WriteJSON(os.Stdout, json.RawMessage(data))
	}

	g.UI.Successf("Added body composition: %.1f kg", c.Weight)
	return nil
}

// BodyDeleteWeightCmd deletes a weigh-in entry.
type BodyDeleteWeightCmd struct {
	PK    string `arg:"" help:"Weigh-in version (primary key)."`
	Date  string `help:"Date of the weigh-in (YYYY-MM-DD)." required:"" name:"date"`
	Force bool   `help:"Skip confirmation prompt." short:"f"`
}

func (c *BodyDeleteWeightCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	ok, err := confirm(os.Stderr, fmt.Sprintf("Delete weigh-in %s on %s?", c.PK, c.Date), c.Force)
	if err != nil {
		return err
	}
	if !ok {
		g.UI.Infof("Cancelled")
		return nil
	}

	if err := client.DeleteWeight(g.Context, c.Date, c.PK); err != nil {
		return fmt.Errorf("delete weight: %w", err)
	}

	g.UI.Successf("Deleted weigh-in %s", c.PK)
	return nil
}

// BodyBloodPressureCmd shows blood pressure data for a date range.
type BodyBloodPressureCmd struct {
	StartDate string `help:"Start date (YYYY-MM-DD)." required:"" name:"start"`
	EndDate   string `help:"End date (YYYY-MM-DD)." required:"" name:"end"`
}

func (c *BodyBloodPressureCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.GetBloodPressure(g.Context, c.StartDate, c.EndDate)
	if err != nil {
		return fmt.Errorf("get blood pressure: %w", err)
	}

	return writeHealthJSON(g, data)
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

	if outfmt.IsJSON(g.Context) {
		return outfmt.WriteJSON(os.Stdout, json.RawMessage(data))
	}

	g.UI.Successf("Added blood pressure: %d/%d mmHg, pulse %d bpm", c.Systolic, c.Diastolic, c.Pulse)
	return nil
}

// BodyDeleteBPCmd deletes a blood pressure entry.
type BodyDeleteBPCmd struct {
	Version string `arg:"" help:"Blood pressure entry version."`
	Date    string `help:"Date of the measurement (YYYY-MM-DD)." required:"" name:"date"`
	Force   bool   `help:"Skip confirmation prompt." short:"f"`
}

func (c *BodyDeleteBPCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	ok, err := confirm(os.Stderr, fmt.Sprintf("Delete blood pressure entry %s on %s?", c.Version, c.Date), c.Force)
	if err != nil {
		return err
	}
	if !ok {
		g.UI.Infof("Cancelled")
		return nil
	}

	if err := client.DeleteBloodPressure(g.Context, c.Date, c.Version); err != nil {
		return fmt.Errorf("delete blood pressure: %w", err)
	}

	g.UI.Successf("Deleted blood pressure entry %s", c.Version)
	return nil
}
