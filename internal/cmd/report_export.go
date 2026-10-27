package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bpauli/gccli/internal/garminapi"
)

// BodyExportBloodPressureCmd saves the Garmin blood pressure report of a
// date range as a PDF or JSON file.
type BodyExportBloodPressureCmd struct {
	StartDate Date    `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"start"`
	EndDate   Date    `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"end"`
	Period    *string `help:"Report layout: daily, seven-days, four-weeks or yearly. Default: the shortest one that covers the range." enum:"daily,seven-days,four-weeks,yearly"`
	Format    string  `help:"File format: pdf or json." default:"pdf" enum:"pdf,json"`
	Output    string  `help:"Output file path (default: blood_pressure_{start}_{end}.{format})." short:"o"`
}

func (c *BodyExportBloodPressureCmd) Help() string {
	return "Garmin creates the report and the command waits for it, up to two minutes. Garmin offers PDF and JSON only. An existing output file is overwritten. For the readings as data, use body blood-pressure."
}

func (c *BodyExportBloodPressureCmd) Validate() error {
	if c.StartDate == "" || c.EndDate == "" {
		return nil
	}
	_, err := c.days()
	return err
}

func (c *BodyExportBloodPressureCmd) days() (int, error) {
	start, err := time.Parse(time.DateOnly, string(c.StartDate))
	if err != nil {
		return 0, fmt.Errorf("--start: %w", err)
	}
	end, err := time.Parse(time.DateOnly, string(c.EndDate))
	if err != nil {
		return 0, fmt.Errorf("--end: %w", err)
	}
	if end.Before(start) {
		return 0, fmt.Errorf("--end %s is before --start %s", c.EndDate, c.StartDate)
	}
	return int(end.Sub(start).Hours()/24) + 1, nil
}

var bloodPressurePeriods = map[string]garminapi.ReportPeriod{
	"daily":      garminapi.PeriodDaily,
	"seven-days": garminapi.PeriodSevenDays,
	"four-weeks": garminapi.PeriodFourWeeks,
	"yearly":     garminapi.PeriodYearly,
}

func (c *BodyExportBloodPressureCmd) period() garminapi.ReportPeriod {
	if c.Period != nil {
		return bloodPressurePeriods[*c.Period]
	}
	days, _ := c.days()
	switch {
	case days <= 1:
		return garminapi.PeriodDaily
	case days <= 7:
		return garminapi.PeriodSevenDays
	case days <= 28:
		return garminapi.PeriodFourWeeks
	default:
		return garminapi.PeriodYearly
	}
}

func (c *BodyExportBloodPressureCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	prefs, err := client.GetReportPreferences(g.Context)
	if err != nil {
		return fmt.Errorf("get profile settings: %w", err)
	}
	format := garminapi.ReportFormat(strings.ToUpper(c.Format))
	req := garminapi.BloodPressureReport(string(c.StartDate), string(c.EndDate), c.period(), format, prefs)
	out := c.Output
	if out == "" {
		out = fmt.Sprintf("blood_pressure_%s_%s.%s", c.StartDate, c.EndDate, c.Format)
	}
	return saveReport(g, client, req, out, "export blood pressure report")
}

// HealthSnapshotReportCmd saves the Garmin PDF report of a Health Snapshot
// session.
type HealthSnapshotReportCmd struct {
	Date   Date   `arg:"" help:"Date of the session (YYYY-MM-DD, today, yesterday, 3d)."`
	UUID   ID     `arg:"" help:"Session UUID (activityUuid.uuid in 'health snapshots list')."`
	Output string `help:"Output file path (default: health_snapshot_{uuid}.pdf)." short:"o"`
}

func (c *HealthSnapshotReportCmd) Help() string {
	return "Garmin creates the report and the command waits for it, up to two minutes. The report is a PDF. An existing output file is overwritten."
}

func (c *HealthSnapshotReportCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	prefs, err := client.GetReportPreferences(g.Context)
	if err != nil {
		return fmt.Errorf("get profile settings: %w", err)
	}
	out := c.Output
	if out == "" {
		out = fmt.Sprintf("health_snapshot_%s.pdf", c.UUID)
	}
	return saveReport(g, client, garminapi.HealthSnapshotReport(string(c.Date), string(c.UUID), prefs), out, "export health snapshot report")
}

func saveReport(g *Globals, client *garminapi.Client, req garminapi.ReportRequest, out, verb string) error {
	data, err := client.ExportReport(g.Context, req)
	if err != nil {
		return fmt.Errorf("%s: %w", verb, err)
	}
	if err := os.WriteFile(out, data, 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	g.UI.Successf("Downloaded %s (%d bytes)", out, len(data))
	return nil
}
