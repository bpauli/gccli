package cmd

var healthBodyBatteryGroup = Group{Name: "body-battery", Help: "Show body battery data.", Children: []Node{
	Leaf{
		Name: "view", Help: "Show body battery data for a date.", Default: true,
		Spec: argsOf(&struct {
			Date Date `arg:"" optional:"" default:"today" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
		}{}),
		Endpoint: "health.body-battery",
		Params:   Params{"startDate": In("Date"), "endDate": In("Date")},
		Verb:     "get body battery", Output: JSON{List: true},
	},
	Leaf{
		Name: "range", Help: "Show body battery data for a date range.",
		Spec: argsOf(&struct {
			StartDate Date `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"start"`
			EndDate   Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"end"`
		}{}),
		Endpoint: "health.body-battery",
		Params:   Params{"startDate": In("StartDate"), "endDate": In("EndDate")},
		Verb:     "get body battery range", Output: JSON{List: true},
	},
	Leaf{
		Name: "daily", Help: "Show the daily low and high body battery for a date range (at most 28 days).",
		Spec: argsOf(&struct {
			StartDate Date `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"start"`
			EndDate   Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"end"`
		}{}),
		Endpoint: "health.daily-body-battery",
		Params:   Params{"startDate": In("StartDate"), "endDate": In("EndDate")},
		Verb:     "get daily body battery",
		Output: Rows{Noun: "daily body battery", Columns: []Column{
			{Label: "DATE", Path: "calendarDate"},
			{Label: "LOW", Path: "values.lowBodyBattery"},
			{Label: "HIGH", Path: "values.highBodyBattery"},
		}},
	},
}}

var healthIntensityMinutesGroup = Group{Name: "intensity-minutes", Help: "Show intensity minutes data.", Children: []Node{
	Leaf{
		Name: "view", Help: "Show intensity minutes for a date.", Default: true,
		Spec: argsOf(&struct {
			Date Date `arg:"" optional:"" default:"today" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
		}{}),
		Endpoint: "health.intensity-minutes",
		Params:   Params{"date": In("Date")},
		Verb:     "get intensity minutes", Output: JSON{},
	},
	Leaf{
		Name: "weekly", Help: "Show weekly intensity minutes for a date range.",
		Spec: argsOf(&struct {
			StartDate Date `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"start"`
			EndDate   Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"end"`
		}{}),
		Endpoint: "health.weekly-intensity-minutes",
		Params:   Params{"startDate": In("StartDate"), "endDate": In("EndDate")},
		Verb:     "get weekly intensity minutes", Output: JSON{List: true},
	},
}}

var healthRacePredictionsGroup = Group{Name: "race-predictions", Help: "Show race predictions.", Children: []Node{
	Leaf{
		Name: "view", Help: "Show current race predictions.", Default: true,
		Endpoint: "health.race-predictions",
		Params:   Params{"displayName": DisplayName},
		Verb:     "get race predictions", Output: JSON{},
	},
	Leaf{
		Name: "range", Help: "Show race predictions for a date range.",
		Spec: argsOf(&struct {
			StartDate Date `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"start"`
			EndDate   Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"end"`
		}{}),
		Endpoint: "health.race-predictions-range",
		Params:   Params{"displayName": DisplayName, "startDate": In("StartDate"), "endDate": In("EndDate")},
		Verb:     "get race predictions range", Output: JSON{List: true},
	},
}}
