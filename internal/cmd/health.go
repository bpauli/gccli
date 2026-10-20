package cmd

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
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
	healthHeartRateGroup,
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
	healthFloorsGroup,
	healthSleepGroup,
	healthStressGroup,
	healthRespirationGroup,
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
	healthTrainingStatusGroup,
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
	healthCriticalSwimSpeedGroup,
	Leaf{
		Name: "power-to-weight", Help: "Show latest power-to-weight data as of a date.",
		Spec: argsOf(&struct {
			Date Date `arg:"" optional:"" default:"today" help:"Date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today."`
		}{}),
		Endpoint: "health.power-to-weight",
		Params:   Params{"date": In("Date")},
		Verb:     "get power to weight", Output: JSON{List: true},
	},
	healthThresholdsGroup,
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
		Name: "acclimation", Help: "Show acclimation data for a date range.",
		Spec: argsOf(&struct {
			StartDate Date `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"start"`
			EndDate   Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"end"`
		}{}),
		Endpoint: "health.acclimation",
		Params:   Params{"startDate": In("StartDate"), "endDate": In("EndDate")},
		Verb:     "get acclimation", Output: JSON{},
	},
	Leaf{
		Name: "one-rep-max", Help: "Show current estimated one-rep max values as of a date.",
		Spec: argsOf(&struct {
			Date Date `arg:"" optional:"" default:"today" help:"Date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today."`
		}{}),
		Endpoint: "health.one-rep-max",
		Params:   Params{"date": In("Date")},
		Verb:     "get one-rep max", Output: JSON{List: true},
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
		Verb:     "get steps", Output: JSON{List: true},
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

var healthHeartRateGroup = Group{Name: "hr", Help: "Show heart rate data.", Children: []Node{
	Leaf{
		Name: "view", Help: "Show heart rate data for a date.", Default: true,
		Spec: argsOf(&struct {
			Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
		}{}),
		Endpoint: "health.heart-rate",
		Params:   Params{"displayName": DisplayName, "date": In("Date")},
		Verb:     "get heart rate", Output: JSON{},
		LateDates: []string{"Date"},
	},
	Leaf{
		Name: "daily", Help: "Show daily heart rate statistics for a date range.",
		Spec: argsOf(&struct {
			StartDate Date `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"start"`
			EndDate   Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"end"`
		}{}),
		Endpoint: "health.daily-heart-rate",
		Params:   Params{"startDate": In("StartDate"), "endDate": In("EndDate")},
		Verb:     "get daily heart rate", Output: JSON{},
	},
	Leaf{
		Name: "weekly", Help: "Show weekly heart rate statistics.",
		Spec: argsOf(&struct {
			EndDate Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today." name:"end"`
			Weeks   int  `help:"Number of weeks." default:"4" name:"weeks"`
		}{}),
		Endpoint: "health.weekly-heart-rate",
		Params:   Params{"endDate": In("EndDate").Or(Today), "weeks": In("Weeks")},
		Verb:     "get weekly heart rate", Output: JSON{},
	},
}}

var healthFloorsGroup = Group{Name: "floors", Help: "Show floors climbed data.", Children: []Node{
	Leaf{
		Name: "view", Help: "Show floors climbed for a date.", Default: true,
		Spec: argsOf(&struct {
			Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
		}{}),
		Endpoint: "health.floors",
		Params:   Params{"date": In("Date")},
		Verb:     "get floors", Output: JSON{},
		LateDates: []string{"Date"},
	},
	Leaf{
		Name: "daily", Help: "Show daily floors climbed for a date range.",
		Spec: argsOf(&struct {
			StartDate Date `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"start"`
			EndDate   Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"end"`
		}{}),
		Endpoint: "health.daily-floors",
		Params:   Params{"startDate": In("StartDate"), "endDate": In("EndDate")},
		Verb:     "get daily floors", Output: JSON{},
	},
	Leaf{
		Name: "weekly", Help: "Show weekly floors climbed.",
		Spec: argsOf(&struct {
			EndDate Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today." name:"end"`
			Weeks   int  `help:"Number of weeks." default:"4" name:"weeks"`
		}{}),
		Endpoint: "health.weekly-floors",
		Params:   Params{"endDate": In("EndDate").Or(Today), "weeks": In("Weeks")},
		Verb:     "get weekly floors", Output: JSON{},
	},
}}

var healthRespirationGroup = Group{Name: "respiration", Help: "Show respiration data.", Children: []Node{
	Leaf{
		Name: "view", Help: "Show respiration data for a date.", Default: true,
		Spec: argsOf(&struct {
			Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
		}{}),
		Endpoint: "health.respiration",
		Params:   Params{"date": In("Date")},
		Verb:     "get respiration", Output: JSON{},
		LateDates: []string{"Date"},
	},
	Leaf{
		Name: "daily", Help: "Show daily respiration statistics for a date range.",
		Spec: argsOf(&struct {
			StartDate Date `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"start"`
			EndDate   Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"end"`
		}{}),
		Endpoint: "health.daily-respiration",
		Params:   Params{"startDate": In("StartDate"), "endDate": In("EndDate")},
		Verb:     "get daily respiration", Output: JSON{},
	},
}}

var healthSleepGroup = Group{Name: "sleep", Help: "Show sleep data.", Children: []Node{
	Leaf{
		Name: "view", Help: "Show sleep data for a date.", Default: true,
		Spec: argsOf(&struct {
			Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
		}{}),
		Endpoint: "health.sleep",
		Params:   Params{"displayName": DisplayName, "date": In("Date")},
		Verb:     "get sleep", Output: JSON{},
		LateDates: []string{"Date"},
	},
	healthSleepNapsGroup,
	Leaf{
		Name: "daily", Help: "Show daily sleep statistics for a date range (at most 28 days).",
		Spec: argsOf(&struct {
			StartDate Date `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"start"`
			EndDate   Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"end"`
		}{}),
		Endpoint: "health.daily-sleep",
		Params:   Params{"startDate": In("StartDate"), "endDate": In("EndDate")},
		Verb:     "get daily sleep", Output: JSON{},
	},
	Leaf{
		Name: "weekly", Help: "Show weekly sleep statistics for the 52 weeks up to a date.",
		Spec: argsOf(&struct {
			EndDate Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today." name:"end"`
		}{}),
		Endpoint: "health.weekly-sleep",
		Params:   Params{"endDate": In("EndDate").Or(Today)},
		Verb:     "get weekly sleep", Output: JSON{},
	},
	Leaf{
		Name: "overall", Help: "Show overall sleep statistics for a date range.",
		Spec: argsOf(&struct {
			StartDate Date `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"start"`
			EndDate   Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"end"`
		}{}),
		Endpoint: "health.overall-sleep",
		Params:   Params{"startDate": In("StartDate"), "endDate": In("EndDate")},
		Verb:     "get overall sleep", Output: JSON{},
	},
}}

var healthTrainingStatusGroup = Group{Name: "training-status", Help: "Show training status data.", Children: []Node{
	Leaf{
		Name: "view", Help: "Show aggregated training status for a date.", Default: true,
		Spec: argsOf(&struct {
			Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
		}{}),
		Endpoint: "health.training-status",
		Params:   Params{"date": In("Date")},
		Verb:     "get training status", Output: JSON{},
		LateDates: []string{"Date"},
	},
	Leaf{
		Name: "daily", Help: "Show daily training status for a date.",
		Spec: argsOf(&struct {
			Date Date `arg:"" optional:"" default:"today" help:"Date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today."`
		}{}),
		Endpoint: "health.daily-training-status",
		Params:   Params{"date": In("Date")},
		Verb:     "get daily training status", Output: JSON{},
	},
}}

var healthCriticalSwimSpeedGroup = Group{Name: "critical-swim-speed", Help: "Show critical swim speed data.", Children: []Node{
	Leaf{
		Name: "view", Help: "Show latest critical swim speed as of a date.", Default: true,
		Spec: argsOf(&struct {
			Date Date `arg:"" optional:"" default:"today" help:"Date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today."`
		}{}),
		Endpoint: "health.critical-swim-speed",
		Params:   Params{"date": In("Date")},
		Verb:     "get critical swim speed", Output: JSON{},
	},
	Leaf{
		Name: "range", Help: "Show critical swim speed for a date range.",
		Spec: argsOf(&struct {
			StartDate Date `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"start"`
			EndDate   Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"end"`
		}{}),
		Endpoint: "health.critical-swim-speed-range",
		Params:   Params{"startDate": In("StartDate"), "endDate": In("EndDate")},
		Verb:     "get critical swim speed range", Output: JSON{},
	},
}}

var healthThresholdsGroup = Group{Name: "thresholds", Help: "Show threshold and power-to-weight data for a date range.", Children: []Node{
	Leaf{
		Name: "cycling-ftp", Help: "Show cycling FTP for a date range.",
		Spec: argsOf(&struct {
			StartDate Date `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"start"`
			EndDate   Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"end"`
		}{}),
		Endpoint: "health.cycling-ftp-range",
		Params:   Params{"startDate": In("StartDate"), "endDate": In("EndDate")},
		Verb:     "get cycling ftp range", Output: JSON{List: true},
	},
	Leaf{
		Name: "lactate-heart-rate", Help: "Show lactate threshold heart rate for a date range.",
		Spec: argsOf(&struct {
			StartDate Date `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"start"`
			EndDate   Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"end"`
		}{}),
		Endpoint: "health.lactate-heart-rate-range",
		Params:   Params{"startDate": In("StartDate"), "endDate": In("EndDate")},
		Verb:     "get lactate threshold heart rate range", Output: JSON{List: true},
	},
	Leaf{
		Name: "lactate-speed", Help: "Show lactate threshold speed for a date range.",
		Spec: argsOf(&struct {
			StartDate Date `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"start"`
			EndDate   Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"end"`
		}{}),
		Endpoint: "health.lactate-speed-range",
		Params:   Params{"startDate": In("StartDate"), "endDate": In("EndDate")},
		Verb:     "get lactate threshold speed range", Output: JSON{List: true},
	},
	Leaf{
		Name: "power-to-weight", Help: "Show power-to-weight data for a date range.",
		Spec: argsOf(&struct {
			StartDate Date `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"start"`
			EndDate   Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"end"`
		}{}),
		Endpoint: "health.power-to-weight-range",
		Params:   Params{"startDate": In("StartDate"), "endDate": In("EndDate")},
		Verb:     "get power to weight range", Output: JSON{List: true},
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
