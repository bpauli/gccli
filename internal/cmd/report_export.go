package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/bpauli/gccli/internal/garminapi"
)

// BodyExportBloodPressureCmd saves the Garmin blood pressure report that
// ends on a date. The period sets the layout and the length of the range.
type BodyExportBloodPressureCmd struct {
	Period  string `help:"Report layout and range: daily (1 day), seven-days, four-weeks (28 days) or yearly (52 weeks)." default:"daily" enum:"daily,seven-days,four-weeks,yearly"`
	EndDate Date   `help:"Last day of the report (YYYY-MM-DD, today, yesterday, 3d). Defaults to today." default:"today" name:"end"`
	Output  string `help:"Output file path (default: blood_pressure_{period}_{end}.pdf)." short:"o"`
}

func (c *BodyExportBloodPressureCmd) Help() string {
	return "Garmin creates the report and the command waits for it, up to two minutes. The report is a PDF: Garmin also accepts JSON as a format but returns a PDF for it. The period fixes the range, because Garmin answers DATA_NOT_FOUND for any other length. An existing output file is overwritten. For the readings as data, use body blood-pressure."
}

var bloodPressurePeriods = map[string]garminapi.ReportPeriod{
	"daily":      garminapi.PeriodDaily,
	"seven-days": garminapi.PeriodSevenDays,
	"four-weeks": garminapi.PeriodFourWeeks,
	"yearly":     garminapi.PeriodYearly,
}

func (c *BodyExportBloodPressureCmd) Run(g *Globals) error {
	end, err := time.Parse(time.DateOnly, string(c.EndDate))
	if err != nil {
		return fmt.Errorf("--end: %w", err)
	}
	period := bloodPressurePeriods[c.Period]
	start := end.AddDate(0, 0, 1-period.Days()).Format(time.DateOnly)

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
		out = fmt.Sprintf("blood_pressure_%s_%s.pdf", c.Period, c.EndDate)
	}
	return saveReport(g, client, garminapi.BloodPressureReport(start, string(c.EndDate), period, prefs), out, "export blood pressure report")
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
