package cmd

var foodColumns = []Column{
	{Label: "ID", Path: "foodMetaData.foodId"},
	{Label: "NAME", Path: "foodMetaData.foodName"},
	{Label: "SOURCE", Path: "foodMetaData.source"},
	{Label: "SERVING", Path: "nutritionContents.0.servingUnit"},
	{Label: "CALORIES", Path: "nutritionContents.0.calories"},
}

// savedFoodColumns show a food the user logged or saved, with the serving
// and quantity chosen then. Garmin gives the calories of one serving.
var savedFoodColumns = []Column{
	{Label: "ID", Path: "foodMetaData.foodId"},
	{Label: "NAME", Path: "foodMetaData.foodName"},
	{Label: "SOURCE", Path: "foodMetaData.source"},
	{Label: "SERVING", Path: "selectedNutritionContent.servingUnit"},
	{Label: "QTY", Path: "servingQty"},
	{Label: "CALORIES", Path: "selectedNutritionContent.calories"},
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
	Leaf{
		Name: "recent", Help: "List the foods you logged recently. Needs Garmin Connect+.",
		Spec: argsOf(&struct {
			Date       Date `arg:"" optional:"" default:"today" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
			Pagination `embed:"" set:"items=foods"`
		}{}),
		Endpoint: "nutrition.recent-foods",
		Params:   Params{"date": In("Date"), "start": In("Offset"), "limit": In("Limit")},
		Verb:     "list recent foods",
		Output:   Rows{Noun: "recent foods", Items: "recentFoods", Columns: savedFoodColumns},
	},
	Group{Name: "favorites", Help: "List your favorite foods. Needs Garmin Connect+.", Children: []Node{
		Leaf{
			Name: "list", Help: "List your favorite foods. Needs Garmin Connect+.", Default: true,
			Spec: argsOf(&struct {
				Pagination `embed:"" set:"items=foods"`
			}{}),
			Endpoint: "nutrition.favorite-foods",
			Params:   Params{"start": In("Offset"), "limit": In("Limit")},
			Verb:     "list favorite foods",
			Output:   Rows{Noun: "favorite foods", Items: "consumables", Columns: savedFoodColumns},
		},
	}},
}}
