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
	Leaf{
		Name: "summary", Help: "Show daily health summary.", Default: true,
		Spec: argsOf(&struct {
			Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
		}{}),
		Endpoint: "health.daily-summary",
		Params:   Params{"displayName": DisplayName, "date": In("Date")},
		Verb:     "get daily summary", Output: JSON{},
		LateDates: []string{"Date"},
	},
	healthStepsGroup,
	Leaf{
		Name: "hr", Help: "Show heart rate data.",
		Spec: argsOf(&struct {
			Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
		}{}),
		Endpoint: "health.heart-rate",
		Params:   Params{"displayName": DisplayName, "date": In("Date")},
		Verb:     "get heart rate", Output: JSON{},
		LateDates: []string{"Date"},
	},
	Leaf{
		Name: "rhr", Help: "Show resting heart rate data.",
		Spec: argsOf(&struct {
			Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
		}{}),
		Endpoint: "health.resting-heart-rate",
		Params:   Params{"displayName": DisplayName, "date": In("Date")},
		Verb:     "get resting heart rate", Output: JSON{},
		LateDates: []string{"Date"},
	},
	Leaf{
		Name: "floors", Help: "Show floors climbed data.",
		Spec: argsOf(&struct {
			Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
		}{}),
		Endpoint: "health.floors",
		Params:   Params{"date": In("Date")},
		Verb:     "get floors", Output: JSON{},
		LateDates: []string{"Date"},
	},
	Leaf{
		Name: "sleep", Help: "Show sleep data.",
		Spec: argsOf(&struct {
			Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
		}{}),
		Endpoint: "health.sleep",
		Params:   Params{"displayName": DisplayName, "date": In("Date")},
		Verb:     "get sleep", Output: JSON{},
		LateDates: []string{"Date"},
	},
	healthStressGroup,
	Leaf{
		Name: "respiration", Help: "Show respiration data.",
		Spec: argsOf(&struct {
			Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
		}{}),
		Endpoint: "health.respiration",
		Params:   Params{"date": In("Date")},
		Verb:     "get respiration", Output: JSON{},
		LateDates: []string{"Date"},
	},
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

var healthStepsGroup = Group{Name: "steps", Help: "Show step data.", Children: []Node{
	Leaf{
		Name: "view", Help: "Show step chart data for a date.", Default: true,
		Spec: argsOf(&struct {
			Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
		}{}),
		Endpoint: "health.steps",
		Params:   Params{"displayName": DisplayName, "date": In("Date")},
		Verb:     "get steps", Output: JSON{},
		LateDates: []string{"Date"},
	},
	Leaf{
		Name: "daily", Help: "Show daily step totals for a date range.",
		Spec: argsOf(&struct {
			StartDate Date `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"start"`
			EndDate   Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"end"`
		}{}),
		Endpoint: "health.daily-steps",
		Params:   Params{"startDate": In("StartDate"), "endDate": In("EndDate")},
		Verb:     "get daily steps", Output: JSON{},
	},
	Leaf{
		Name: "weekly", Help: "Show weekly step totals.",
		Spec: argsOf(&struct {
			EndDate Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today." name:"end"`
			Weeks   int  `help:"Number of weeks." default:"4" name:"weeks"`
		}{}),
		Endpoint: "health.weekly-steps",
		Params:   Params{"endDate": In("EndDate").Or(Today), "weeks": In("Weeks")},
		Verb:     "get weekly steps", Output: JSON{},
	},
}}

var healthStressGroup = Group{Name: "stress", Help: "Show stress data.", Children: []Node{
	Leaf{
		Name: "view", Help: "Show stress data for a date.", Default: true,
		Spec: argsOf(&struct {
			Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
		}{}),
		Endpoint: "health.stress",
		Params:   Params{"date": In("Date")},
		Verb:     "get stress", Output: JSON{},
		LateDates: []string{"Date"},
	},
	Leaf{
		Name: "weekly", Help: "Show weekly stress data.",
		Spec: argsOf(&struct {
			EndDate Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today." name:"end"`
			Weeks   int  `help:"Number of weeks." default:"4" name:"weeks"`
		}{}),
		Endpoint: "health.weekly-stress",
		Params:   Params{"endDate": In("EndDate").Or(Today), "weeks": In("Weeks")},
		Verb:     "get weekly stress", Output: JSON{},
	},
}}

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
