package cmd

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bpauli/gccli/internal/outfmt"
)

// nowFn returns the current time. Variable for testing.
var nowFn = time.Now

var healthGroup = Group{Name: "health", Help: "View health data.", Children: []Node{
	Custom{Name: "summary", Help: "Show daily health summary.", Default: true, Cmd: &HealthSummaryCmd{}},
	healthStepsGroup,
	Custom{Name: "hr", Help: "Show heart rate data.", Cmd: &HealthHRCmd{}},
	Custom{Name: "rhr", Help: "Show resting heart rate data.", Cmd: &HealthRHRCmd{}},
	Custom{Name: "floors", Help: "Show floors climbed data.", Cmd: &HealthFloorsCmd{}},
	Custom{Name: "sleep", Help: "Show sleep data.", Cmd: &HealthSleepCmd{}},
	healthStressGroup,
	Custom{Name: "respiration", Help: "Show respiration data.", Cmd: &HealthRespirationCmd{}},
	Custom{Name: "spo2", Help: "Show SpO2 (blood oxygen) data.", Cmd: &HealthSPO2Cmd{}},
	Custom{Name: "hrv", Help: "Show heart rate variability data.", Cmd: &HealthHRVCmd{}},
	healthBodyBatteryGroup,
	healthIntensityMinutesGroup,
	Custom{Name: "training-readiness", Help: "Show training readiness data.", Cmd: &HealthTrainingReadinessCmd{}},
	Custom{Name: "training-status", Help: "Show training status data.", Cmd: &HealthTrainingStatusCmd{}},
	Custom{Name: "fitness-age", Help: "Show fitness age data.", Cmd: &HealthFitnessAgeCmd{}},
	Custom{Name: "max-metrics", Help: "Show VO2max and max metrics data.", Cmd: &HealthMaxMetricsCmd{}},
	Custom{Name: "lactate-threshold", Help: "Show latest lactate threshold.", Cmd: &HealthLactateThresholdCmd{}},
	Custom{Name: "cycling-ftp", Help: "Show latest cycling FTP.", Cmd: &HealthCyclingFTPCmd{}},
	healthRacePredictionsGroup,
	Custom{Name: "endurance-score", Help: "Show endurance score data.", Cmd: &HealthEnduranceScoreCmd{}},
	Custom{Name: "hill-score", Help: "Show hill score data.", Cmd: &HealthHillScoreCmd{}},
	Custom{Name: "events", Help: "Show daily wellness events.", Cmd: &HealthEventsCmd{}},
	Custom{Name: "lifestyle", Help: "Show lifestyle logging data.", Cmd: &HealthLifestyleCmd{}},
}}

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

	displayName, err := resolveDisplayName(g, client)
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "health.daily-summary", map[string]string{"displayName": displayName, "date": date}, nil)
	if err != nil {
		return fmt.Errorf("get daily summary: %w", err)
	}

	return writeHealthJSON(g, data)
}

var healthStepsGroup = Group{Name: "steps", Help: "Show step data.", Children: []Node{
	Custom{Name: "view", Help: "Show step chart data for a date.", Default: true, Cmd: &HealthStepsViewCmd{}},
	Custom{Name: "daily", Help: "Show daily step totals for a date range.", Cmd: &HealthStepsDailyCmd{}},
	Custom{Name: "weekly", Help: "Show weekly step totals.", Cmd: &HealthStepsWeeklyCmd{}},
}}

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

	displayName, err := resolveDisplayName(g, client)
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "health.steps", map[string]string{"displayName": displayName, "date": date}, nil)
	if err != nil {
		return fmt.Errorf("get steps: %w", err)
	}

	return writeHealthJSON(g, data)
}

// HealthStepsDailyCmd shows daily step totals for a date range.
type HealthStepsDailyCmd struct {
	StartDate Date `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"start"`
	EndDate   Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"end"`
}

func (c *HealthStepsDailyCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "health.daily-steps", map[string]string{"startDate": string(c.StartDate), "endDate": string(c.EndDate)}, nil)
	if err != nil {
		return fmt.Errorf("get daily steps: %w", err)
	}

	return writeHealthJSON(g, data)
}

// HealthStepsWeeklyCmd shows weekly step totals.
type HealthStepsWeeklyCmd struct {
	EndDate Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today." name:"end"`
	Weeks   int  `help:"Number of weeks." default:"4" name:"weeks"`
}

func (c *HealthStepsWeeklyCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	endDate := string(c.EndDate)
	if endDate == "" {
		endDate = nowFn().Format("2006-01-02")
	}

	data, err := client.Call(g.Context, "health.weekly-steps", map[string]string{"endDate": endDate, "weeks": strconv.Itoa(c.Weeks)}, nil)
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

	displayName, err := resolveDisplayName(g, client)
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "health.heart-rate", map[string]string{"displayName": displayName, "date": date}, nil)
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

	displayName, err := resolveDisplayName(g, client)
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "health.resting-heart-rate", map[string]string{"displayName": displayName, "date": date}, nil)
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

	data, err := client.Call(g.Context, "health.floors", map[string]string{"date": date}, nil)
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

	displayName, err := resolveDisplayName(g, client)
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "health.sleep", map[string]string{"displayName": displayName, "date": date}, nil)
	if err != nil {
		return fmt.Errorf("get sleep: %w", err)
	}

	return writeHealthJSON(g, data)
}

var healthStressGroup = Group{Name: "stress", Help: "Show stress data.", Children: []Node{
	Custom{Name: "view", Help: "Show stress data for a date.", Default: true, Cmd: &HealthStressViewCmd{}},
	Custom{Name: "weekly", Help: "Show weekly stress data.", Cmd: &HealthStressWeeklyCmd{}},
}}

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

	data, err := client.Call(g.Context, "health.stress", map[string]string{"date": date}, nil)
	if err != nil {
		return fmt.Errorf("get stress: %w", err)
	}

	return writeHealthJSON(g, data)
}

// HealthStressWeeklyCmd shows weekly stress data.
type HealthStressWeeklyCmd struct {
	EndDate Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today." name:"end"`
	Weeks   int  `help:"Number of weeks." default:"4" name:"weeks"`
}

func (c *HealthStressWeeklyCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	endDate := string(c.EndDate)
	if endDate == "" {
		endDate = nowFn().Format("2006-01-02")
	}

	data, err := client.Call(g.Context, "health.weekly-stress", map[string]string{"endDate": endDate, "weeks": strconv.Itoa(c.Weeks)}, nil)
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

	data, err := client.Call(g.Context, "health.respiration", map[string]string{"date": date}, nil)
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
		_, _ = fmt.Fprintln(g.Stdout, "{}")
		return nil
	}
	return outfmt.WriteJSON(g.Stdout, json.RawMessage(data))
}

// relativeDateRe matches relative date strings like "3d", "7d", "14d".
var relativeDateRe = regexp.MustCompile(`^(\d+)d$`)

// futureDateRe matches forward-looking date strings like "+7d", "+365d".
var futureDateRe = regexp.MustCompile(`^\+(\d+)d$`)

// resolveDate parses a date string that can be:
//   - empty or "today": today's date
//   - "yesterday": yesterday's date
//   - "Nd" (e.g. "3d"): N days ago
//   - "+Nd" (e.g. "+365d"): N days from now
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

	if m := futureDateRe.FindStringSubmatch(s); m != nil {
		days, _ := strconv.Atoi(m[1])
		return now.AddDate(0, 0, days).Format("2006-01-02"), nil
	}

	if m := relativeDateRe.FindStringSubmatch(s); m != nil {
		days, _ := strconv.Atoi(m[1])
		return now.AddDate(0, 0, -days).Format("2006-01-02"), nil
	}

	// Validate YYYY-MM-DD format.
	if _, err := time.Parse("2006-01-02", s); err != nil {
		return "", fmt.Errorf("invalid date %q: expected YYYY-MM-DD, today, yesterday, Nd (e.g. 3d), or +Nd (e.g. +30d)", s)
	}

	return s, nil
}
