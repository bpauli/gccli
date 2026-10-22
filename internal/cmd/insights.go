package cmd

var insightsGroup = Group{Name: "insights", Help: "Compare your metrics with people of your age and gender.", Children: []Node{
	Leaf{
		Name: "options", Help: "List the genders, ages, dates, activities and fields for insights compare.", Default: true,
		Spec: argsOf(&struct {
			LocaleUnit string `help:"Units of the field names: METRIC or STATUTE." default:"METRIC" enum:"METRIC,STATUTE"`
		}{}),
		Endpoint: "insights.selections",
		Params:   Params{"localeUnit": In("LocaleUnit")},
		Verb:     "get insights options", Output: JSON{},
	},
	Leaf{
		Name: "compare", Help: "Compare a metric with your age group. Needs Insights on in Garmin Connect.",
		Spec: argsOf(&struct {
			Activity   string   `help:"Activity as insights options prints it: STEPS, SLEEP, FLOORS, RUNNING, CYCLING, LAP_SWIMMING or OPEN_WATER_SWIMMING." required:"" enum:"STEPS,SLEEP,FLOORS,RUNNING,CYCLING,LAP_SWIMMING,OPEN_WATER_SWIMMING"`
			Field      string   `help:"Field of the activity as insights options prints it: AVG_SPEED, VO2_MAX_VALUE, AVG_POWER, WEEKLY_AVG_DISTANCE, WEEKLY_AVG_DURATION, STEPS, SLEEP, FLOORS or COUNT." required:"" enum:"AVG_SPEED,VO2_MAX_VALUE,AVG_POWER,WEEKLY_AVG_DISTANCE,WEEKLY_AVG_DURATION,STEPS,SLEEP,FLOORS,COUNT"`
			Age        FreeText `help:"Age group as insights options prints it (e.g. 40-44, 75+)." required:""`
			Gender     string   `help:"Gender: ALL, M or F." default:"ALL" enum:"ALL,M,F"`
			Year       int      `help:"Year (e.g. 2025)." required:""`
			Month      Month    `help:"Month (1-12). Omit it for the whole year."`
			LocaleUnit string   `help:"Units of the answer: METRIC or STATUTE." default:"METRIC" enum:"METRIC,STATUTE"`
		}{}),
		Endpoint: "insights.compare",
		Params: Params{
			"gender": In("Gender"), "age": In("Age"), "type": In("Activity"), "field": In("Field"),
			"localeUnit": In("LocaleUnit"), "year": In("Year"), "month": In("Month"),
		},
		Verb: "compare insights", Output: JSON{},
	},
	Leaf{Name: "preferences", Help: "Show your saved Insights comparisons.", Endpoint: "insights.preferences", Verb: "get insights preferences", Output: JSON{List: true}},
}}
