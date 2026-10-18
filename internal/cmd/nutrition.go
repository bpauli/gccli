package cmd

var nutritionGroup = Group{Name: "nutrition", Help: "View nutrition data.", Children: []Node{
	Leaf{
		Name: "food-log", Help: "Show daily food log for a date.", Default: true,
		Spec: argsOf(&struct {
			Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
		}{}),
		Endpoint: "nutrition.daily-food-log",
		Params:   Params{"date": In("Date")},
		Verb:     "get nutrition food log", Output: JSON{},
		LateDates: []string{"Date"},
	},
	Leaf{
		Name: "meals", Help: "Show meal summaries for a date.",
		Spec: argsOf(&struct {
			Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
		}{}),
		Endpoint: "nutrition.daily-meals",
		Params:   Params{"date": In("Date")},
		Verb:     "get nutrition meals", Output: JSON{},
		LateDates: []string{"Date"},
	},
	Leaf{
		Name: "settings", Help: "Show nutrition settings for a date.",
		Spec: argsOf(&struct {
			Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
		}{}),
		Endpoint: "nutrition.daily-settings",
		Params:   Params{"date": In("Date")},
		Verb:     "get nutrition settings", Output: JSON{},
		LateDates: []string{"Date"},
	},
}}
