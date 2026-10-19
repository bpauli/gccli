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
	healthCaloriesGroup,
	healthPushesGroup,
	Leaf{
		Name: "lifetime-totals", Help: "Show lifetime totals.",
		Endpoint: "health.lifetime-totals",
		Params:   Params{"displayName": DisplayName},
		Verb:     "get lifetime totals", Output: JSON{},
	},
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
	Leaf{
		Name: "spo2", Help: "Show SpO2 (blood oxygen) data.",
		Spec: argsOf(&struct {
			Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
		}{}),
		Endpoint: "health.spo2",
		Params:   Params{"date": In("Date")},
		Verb:     "get spo2", Output: JSON{},
		LateDates: []string{"Date"},
	},
	Leaf{
		Name: "hrv", Help: "Show heart rate variability data.",
		Spec: argsOf(&struct {
			Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
		}{}),
		Endpoint: "health.hrv",
		Params:   Params{"date": In("Date")},
		Verb:     "get hrv", Output: JSON{},
		LateDates: []string{"Date"},
	},
	healthBodyBatteryGroup,
	healthIntensityMinutesGroup,
	Leaf{
		Name: "training-readiness", Help: "Show training readiness data.",
		Spec: argsOf(&struct {
			Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
		}{}),
		Endpoint: "health.training-readiness",
		Params:   Params{"date": In("Date")},
		Verb:     "get training readiness", Output: JSON{},
		LateDates: []string{"Date"},
	},
	Leaf{
		Name: "training-status", Help: "Show training status data.",
		Spec: argsOf(&struct {
			Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
		}{}),
		Endpoint: "health.training-status",
		Params:   Params{"date": In("Date")},
		Verb:     "get training status", Output: JSON{},
		LateDates: []string{"Date"},
	},
	Leaf{
		Name: "fitness-age", Help: "Show fitness age data.",
		Spec: argsOf(&struct {
			Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
		}{}),
		Endpoint: "health.fitness-age",
		Params:   Params{"date": In("Date")},
		Verb:     "get fitness age", Output: JSON{},
		LateDates: []string{"Date"},
	},
	Leaf{
		Name: "max-metrics", Help: "Show VO2max and max metrics data.",
		Spec: argsOf(&struct {
			Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
		}{}),
		Endpoint: "health.max-metrics",
		Params:   Params{"date": In("Date")},
		Verb:     "get max metrics", Output: JSON{},
		LateDates: []string{"Date"},
	},
	Leaf{Name: "lactate-threshold", Help: "Show latest lactate threshold.", Endpoint: "health.lactate-threshold", Verb: "get lactate threshold", Output: JSON{}},
	Leaf{Name: "cycling-ftp", Help: "Show latest cycling FTP.", Endpoint: "health.cycling-ftp", Verb: "get cycling ftp", Output: JSON{}},
	Leaf{
		Name: "critical-swim-speed", Help: "Show latest critical swim speed as of a date.",
		Spec: argsOf(&struct {
			Date Date `arg:"" optional:"" default:"today" help:"Date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today."`
		}{}),
		Endpoint: "health.critical-swim-speed",
		Params:   Params{"date": In("Date")},
		Verb:     "get critical swim speed", Output: JSON{},
	},
	healthRacePredictionsGroup,
	Leaf{
		Name: "endurance-score", Help: "Show endurance score data.",
		Spec: argsOf(&struct {
			Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
		}{}),
		Endpoint: "health.endurance-score",
		Params:   Params{"date": In("Date")},
		Verb:     "get endurance score", Output: JSON{},
		LateDates: []string{"Date"},
	},
	Leaf{
		Name: "hill-score", Help: "Show hill score data.",
		Spec: argsOf(&struct {
			Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
		}{}),
		Endpoint: "health.hill-score",
		Params:   Params{"date": In("Date")},
		Verb:     "get hill score", Output: JSON{},
		LateDates: []string{"Date"},
	},
	Leaf{
		Name: "power-curve", Help: "Show power curve for a date range.",
		Spec: argsOf(&struct {
			StartDate Date `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"start"`
			EndDate   Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"end"`
		}{}),
		Endpoint: "health.power-curve",
		Params:   Params{"startDate": In("StartDate"), "endDate": In("EndDate")},
		Verb:     "get power curve", Output: JSON{},
	},
	Leaf{
		Name: "running-economy", Help: "Show latest running economy as of a date.",
		Spec: argsOf(&struct {
			Date Date `arg:"" optional:"" default:"today" help:"Date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today."`
		}{}),
		Endpoint: "health.running-economy",
		Params:   Params{"date": In("Date")},
		Verb:     "get running economy", Output: JSON{},
	},
	Leaf{
		Name: "events", Help: "Show daily wellness events.",
		Spec: argsOf(&struct {
			Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
		}{}),
		Endpoint: "health.all-day-events",
		Params:   Params{"date": In("Date")},
		Verb:     "get events", Output: JSON{},
		LateDates: []string{"Date"},
	},
	Leaf{
		Name: "lifestyle", Help: "Show lifestyle logging data.",
		Spec: argsOf(&struct {
			Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
		}{}),
		Endpoint: "health.lifestyle-logging",
		Params:   Params{"date": In("Date")},
		Verb:     "get lifestyle", Output: JSON{},
		LateDates: []string{"Date"},
	},
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
	Leaf{
		Name: "monthly", Help: "Show monthly step totals.",
		Spec: argsOf(&struct {
			EndDate Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today." name:"end"`
			Months  int  `help:"Number of months." default:"6" name:"months"`
		}{}),
		Endpoint: "health.monthly-steps",
		Params:   Params{"endDate": In("EndDate").Or(Today), "months": In("Months")},
		Verb:     "get monthly steps", Output: JSON{},
	},
}}

var healthCaloriesGroup = Group{Name: "calories", Help: "Show calorie statistics.", Children: []Node{
	Leaf{
		Name: "weekly", Help: "Show weekly calorie totals.",
		Spec: argsOf(&struct {
			EndDate Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today." name:"end"`
			Weeks   int  `help:"Number of weeks." default:"4" name:"weeks"`
		}{}),
		Endpoint: "health.weekly-calories",
		Params:   Params{"endDate": In("EndDate").Or(Today), "weeks": In("Weeks")},
		Verb:     "get weekly calories", Output: JSON{},
	},
}}

var healthPushesGroup = Group{Name: "pushes", Help: "Show wheelchair push statistics.", Children: []Node{
	Leaf{
		Name: "weekly", Help: "Show weekly wheelchair push totals.",
		Spec: argsOf(&struct {
			EndDate Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today." name:"end"`
			Weeks   int  `help:"Number of weeks." default:"4" name:"weeks"`
		}{}),
		Endpoint: "health.weekly-pushes",
		Params:   Params{"endDate": In("EndDate").Or(Today), "weeks": In("Weeks")},
		Verb:     "get weekly pushes", Output: JSON{},
	},
	Leaf{
		Name: "monthly", Help: "Show monthly wheelchair push totals.",
		Spec: argsOf(&struct {
			EndDate Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today." name:"end"`
			Months  int  `help:"Number of months." default:"6" name:"months"`
		}{}),
		Endpoint: "health.monthly-pushes",
		Params:   Params{"endDate": In("EndDate").Or(Today), "months": In("Months")},
		Verb:     "get monthly pushes", Output: JSON{},
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
		Name: "daily", Help: "Show daily stress statistics for a date range.",
		Spec: argsOf(&struct {
			StartDate Date `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"start"`
			EndDate   Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"end"`
		}{}),
		Endpoint: "health.daily-stress",
		Params:   Params{"startDate": In("StartDate"), "endDate": In("EndDate")},
		Verb:     "get daily stress", Output: JSON{},
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
