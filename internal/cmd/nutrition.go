package cmd

var foodColumns = []Column{
	{Label: "ID", Path: "foodMetaData.foodId"},
	{Label: "NAME", Path: "foodMetaData.foodName"},
	{Label: "SOURCE", Path: "foodMetaData.source"},
	{Label: "SERVING", Path: "nutritionContents.0.servingUnit"},
	{Label: "CALORIES", Path: "nutritionContents.0.calories"},
}

var nutritionGroup = Group{Name: "nutrition", Help: "View nutrition data.", Children: []Node{
	Leaf{
		Name: "food-log", Help: "Show daily food log for a date.", Default: true,
		Spec: argsOf(&struct {
			Date Date `arg:"" optional:"" default:"today" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
		}{}),
		Endpoint: "nutrition.daily-food-log",
		Params:   Params{"date": In("Date")},
		Verb:     "get nutrition food log", Output: JSON{},
	},
	Leaf{
		Name: "meals", Help: "Show meal summaries for a date.",
		Spec: argsOf(&struct {
			Date Date `arg:"" optional:"" default:"today" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
		}{}),
		Endpoint: "nutrition.daily-meals",
		Params:   Params{"date": In("Date")},
		Verb:     "get nutrition meals", Output: JSON{},
	},
	Leaf{
		Name: "settings", Help: "Show nutrition settings for a date.",
		Spec: argsOf(&struct {
			Date Date `arg:"" optional:"" default:"today" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
		}{}),
		Endpoint: "nutrition.daily-settings",
		Params:   Params{"date": In("Date")},
		Verb:     "get nutrition settings", Output: JSON{},
	},
	Leaf{
		Name: "search", Help: "Search the Garmin food database. Needs Garmin Connect+.",
		Spec: argsOf(&struct {
			Query      FreeText `arg:"" help:"Text to search for, such as a food name."`
			Pagination `embed:"" set:"items=foods"`
		}{}),
		Endpoint: "nutrition.food-search",
		Params:   Params{"query": In("Query"), "start": In("Offset"), "limit": In("Limit")},
		Verb:     "search foods",
		Output:   Rows{Noun: "foods", Items: "results", Columns: foodColumns},
	},
}}
