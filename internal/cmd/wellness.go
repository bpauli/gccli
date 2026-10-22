package cmd

var wellnessGroup = Group{Name: "wellness", Help: "View wellness data.", Children: []Node{
	Leaf{
		Name: "menstrual-cycle", Help: "Show the menstrual cycle day view of a date.",
		Spec: argsOf(&struct {
			Date Date `arg:"" optional:"" default:"today" help:"Date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today."`
		}{}),
		Endpoint: "wellness.menstrual-cycle-data",
		Params:   Params{"date": In("Date"), "todayDate": Today},
		Verb:     "get menstrual cycle data", Output: JSON{},
	},
	Leaf{
		Name: "menstrual-summary", Help: "Show the menstrual cycle summary of a date.",
		Spec: argsOf(&struct {
			Date Date `arg:"" optional:"" default:"today" help:"Date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today."`
		}{}),
		Endpoint: "wellness.menstrual-cycle-summary",
		Params:   Params{"date": In("Date")},
		Verb:     "get menstrual cycle summary", Output: JSON{},
	},
	Leaf{
		Name: "menstrual-calendar", Help: "Show the menstrual cycle calendar for a date range.",
		Spec: argsOf(&struct {
			StartDate Date `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"start"`
			EndDate   Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"end"`
		}{}),
		Endpoint: "wellness.menstrual-calendar",
		Params:   Params{"startDate": In("StartDate"), "endDate": In("EndDate")},
		Verb:     "get menstrual calendar", Output: JSON{},
	},
	wellnessMenstrualLogGroup,
	Leaf{Name: "pregnancy-summary", Help: "Show pregnancy summary.", Endpoint: "wellness.pregnancy-summary", Verb: "get pregnancy summary", Output: JSON{}},
	wellnessJetLagGroup,
}}
