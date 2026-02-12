package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bpauli/gccli/internal/outfmt"
)

// nowFn returns the current time. Variable for testing.
var nowFn = time.Now

// HealthCmd groups health data subcommands.
type HealthCmd struct {
	Summary           HealthSummaryCmd           `cmd:"" default:"withargs" help:"Show daily health summary."`
	Steps             HealthStepsCmd             `cmd:"" help:"Show step data."`
	HR                HealthHRCmd                `cmd:"" name:"hr" help:"Show heart rate data."`
	RHR               HealthRHRCmd               `cmd:"" name:"rhr" help:"Show resting heart rate data."`
	Floors            HealthFloorsCmd            `cmd:"" help:"Show floors climbed data."`
	Sleep             HealthSleepCmd             `cmd:"" help:"Show sleep data."`
	Stress            HealthStressCmd            `cmd:"" help:"Show stress data."`
	Respiration       HealthRespirationCmd       `cmd:"" help:"Show respiration data."`
	SPO2              HealthSPO2Cmd              `cmd:"" name:"spo2" help:"Show SpO2 (blood oxygen) data."`
	HRV               HealthHRVCmd               `cmd:"" name:"hrv" help:"Show heart rate variability data."`
	BodyBattery       HealthBodyBatteryCmd       `cmd:"" name:"body-battery" help:"Show body battery data."`
	IntensityMinutes  HealthIntensityMinutesCmd  `cmd:"" name:"intensity-minutes" help:"Show intensity minutes data."`
	TrainingReadiness HealthTrainingReadinessCmd `cmd:"" name:"training-readiness" help:"Show training readiness data."`
	TrainingStatus    HealthTrainingStatusCmd    `cmd:"" name:"training-status" help:"Show training status data."`
	FitnessAge        HealthFitnessAgeCmd        `cmd:"" name:"fitness-age" help:"Show fitness age data."`
	MaxMetrics        HealthMaxMetricsCmd        `cmd:"" name:"max-metrics" help:"Show VO2max and max metrics data."`
	LactateThreshold  HealthLactateThresholdCmd  `cmd:"" name:"lactate-threshold" help:"Show latest lactate threshold."`
	CyclingFTP        HealthCyclingFTPCmd        `cmd:"" name:"cycling-ftp" help:"Show latest cycling FTP."`
	RacePredictions   HealthRacePredictionsCmd   `cmd:"" name:"race-predictions" help:"Show race predictions."`
	EnduranceScore    HealthEnduranceScoreCmd    `cmd:"" name:"endurance-score" help:"Show endurance score data."`
	HillScore         HealthHillScoreCmd         `cmd:"" name:"hill-score" help:"Show hill score data."`
	Events            HealthEventsCmd            `cmd:"" help:"Show daily wellness events."`
	Lifestyle         HealthLifestyleCmd         `cmd:"" help:"Show lifestyle logging data."`
}

// HealthSummaryCmd shows the daily health summary.
type HealthSummaryCmd struct {
	Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
}

func (c *HealthSummaryCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	date, err := resolveDate(c.Date)
	if err != nil {
		return err
	}

	displayName := client.Tokens().DisplayName
	data, err := client.GetDailySummary(g.Context, displayName, date)
	if err != nil {
		return fmt.Errorf("get daily summary: %w", err)
	}

	return writeHealthJSON(g, data)
}

// HealthStepsCmd shows step data with daily and weekly subcommands.
type HealthStepsCmd struct {
	View   HealthStepsViewCmd   `cmd:"" default:"withargs" help:"Show step chart data for a date."`
	Daily  HealthStepsDailyCmd  `cmd:"" help:"Show daily step totals for a date range."`
	Weekly HealthStepsWeeklyCmd `cmd:"" help:"Show weekly step totals."`
}

// HealthStepsViewCmd shows step chart data for a date.
type HealthStepsViewCmd struct {
	Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
}

func (c *HealthStepsViewCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	date, err := resolveDate(c.Date)
	if err != nil {
		return err
	}

	displayName := client.Tokens().DisplayName
	data, err := client.GetSteps(g.Context, displayName, date)
	if err != nil {
		return fmt.Errorf("get steps: %w", err)
	}

	return writeHealthJSON(g, data)
}

// HealthStepsDailyCmd shows daily step totals for a date range.
type HealthStepsDailyCmd struct {
	StartDate string `help:"Start date (YYYY-MM-DD)." required:"" name:"start"`
	EndDate   string `help:"End date (YYYY-MM-DD)." required:"" name:"end"`
}

func (c *HealthStepsDailyCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.GetDailySteps(g.Context, c.StartDate, c.EndDate)
	if err != nil {
		return fmt.Errorf("get daily steps: %w", err)
	}

	return writeHealthJSON(g, data)
}

// HealthStepsWeeklyCmd shows weekly step totals.
type HealthStepsWeeklyCmd struct {
	EndDate string `help:"End date (YYYY-MM-DD). Defaults to today." name:"end"`
	Weeks   int    `help:"Number of weeks." default:"4" name:"weeks"`
}

func (c *HealthStepsWeeklyCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	endDate := c.EndDate
	if endDate == "" {
		endDate = nowFn().Format("2006-01-02")
	}

	data, err := client.GetWeeklySteps(g.Context, endDate, c.Weeks)
	if err != nil {
		return fmt.Errorf("get weekly steps: %w", err)
	}

	return writeHealthJSON(g, data)
}

// HealthHRCmd shows heart rate data for a date.
type HealthHRCmd struct {
	Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
}

func (c *HealthHRCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	date, err := resolveDate(c.Date)
	if err != nil {
		return err
	}

	displayName := client.Tokens().DisplayName
	data, err := client.GetHeartRate(g.Context, displayName, date)
	if err != nil {
		return fmt.Errorf("get heart rate: %w", err)
	}

	return writeHealthJSON(g, data)
}

// HealthRHRCmd shows resting heart rate data for a date.
type HealthRHRCmd struct {
	Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
}

func (c *HealthRHRCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	date, err := resolveDate(c.Date)
	if err != nil {
		return err
	}

	displayName := client.Tokens().DisplayName
	data, err := client.GetRestingHeartRate(g.Context, displayName, date)
	if err != nil {
		return fmt.Errorf("get resting heart rate: %w", err)
	}

	return writeHealthJSON(g, data)
}

// HealthFloorsCmd shows floors climbed data for a date.
type HealthFloorsCmd struct {
	Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
}

func (c *HealthFloorsCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	date, err := resolveDate(c.Date)
	if err != nil {
		return err
	}

	data, err := client.GetFloors(g.Context, date)
	if err != nil {
		return fmt.Errorf("get floors: %w", err)
	}

	return writeHealthJSON(g, data)
}

// HealthSleepCmd shows sleep data for a date.
type HealthSleepCmd struct {
	Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
}

func (c *HealthSleepCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	date, err := resolveDate(c.Date)
	if err != nil {
		return err
	}

	displayName := client.Tokens().DisplayName
	data, err := client.GetSleep(g.Context, displayName, date)
	if err != nil {
		return fmt.Errorf("get sleep: %w", err)
	}

	return writeHealthJSON(g, data)
}

// HealthStressCmd shows stress data with a weekly subcommand.
type HealthStressCmd struct {
	View   HealthStressViewCmd   `cmd:"" default:"withargs" help:"Show stress data for a date."`
	Weekly HealthStressWeeklyCmd `cmd:"" help:"Show weekly stress data."`
}

// HealthStressViewCmd shows stress data for a date.
type HealthStressViewCmd struct {
	Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
}

func (c *HealthStressViewCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	date, err := resolveDate(c.Date)
	if err != nil {
		return err
	}

	data, err := client.GetStress(g.Context, date)
	if err != nil {
		return fmt.Errorf("get stress: %w", err)
	}

	return writeHealthJSON(g, data)
}

// HealthStressWeeklyCmd shows weekly stress data.
type HealthStressWeeklyCmd struct {
	EndDate string `help:"End date (YYYY-MM-DD). Defaults to today." name:"end"`
	Weeks   int    `help:"Number of weeks." default:"4" name:"weeks"`
}

func (c *HealthStressWeeklyCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	endDate := c.EndDate
	if endDate == "" {
		endDate = nowFn().Format("2006-01-02")
	}

	data, err := client.GetWeeklyStress(g.Context, endDate, c.Weeks)
	if err != nil {
		return fmt.Errorf("get weekly stress: %w", err)
	}

	return writeHealthJSON(g, data)
}

// HealthRespirationCmd shows respiration data for a date.
type HealthRespirationCmd struct {
	Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
}

func (c *HealthRespirationCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	date, err := resolveDate(c.Date)
	if err != nil {
		return err
	}

	data, err := client.GetRespiration(g.Context, date)
	if err != nil {
		return fmt.Errorf("get respiration: %w", err)
	}

	return writeHealthJSON(g, data)
}

// writeHealthJSON writes health data as JSON output.
// Health endpoints return complex nested JSON, so we always output JSON
// regardless of output mode.
func writeHealthJSON(g *Globals, data json.RawMessage) error {
	if data == nil {
		_, _ = fmt.Fprintln(os.Stdout, "{}")
		return nil
	}
	return outfmt.WriteJSON(os.Stdout, json.RawMessage(data))
}

// relativeDateRe matches relative date strings like "3d", "7d", "14d".
var relativeDateRe = regexp.MustCompile(`^(\d+)d$`)

// resolveDate parses a date string that can be:
//   - empty or "today": today's date
//   - "yesterday": yesterday's date
//   - "Nd" (e.g. "3d"): N days ago
//   - "YYYY-MM-DD": an explicit date
func resolveDate(s string) (string, error) {
	s = strings.TrimSpace(s)

	now := nowFn()

	if s == "" || strings.EqualFold(s, "today") {
		return now.Format("2006-01-02"), nil
	}

	if strings.EqualFold(s, "yesterday") {
		return now.AddDate(0, 0, -1).Format("2006-01-02"), nil
	}

	if m := relativeDateRe.FindStringSubmatch(s); m != nil {
		days, _ := strconv.Atoi(m[1])
		return now.AddDate(0, 0, -days).Format("2006-01-02"), nil
	}

	// Validate YYYY-MM-DD format.
	if _, err := time.Parse("2006-01-02", s); err != nil {
		return "", fmt.Errorf("invalid date %q: expected YYYY-MM-DD, today, yesterday, or Nd (e.g. 3d)", s)
	}

	return s, nil
}
