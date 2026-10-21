package cmd

import (
	"fmt"
	"strings"

	"github.com/bpauli/gccli/internal/garminapi"
)

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

var nutritionGroup = Group{Name: "nutrition", Help: "View and log nutrition data.", Children: []Node{
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
			Query             FreeText `arg:"" help:"Text to search for, such as a food name."`
			AlignedPagination `embed:"" set:"items=foods"`
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
	Custom{Name: "quick-add", Help: "Log calories and macros in a meal without a food. Needs Garmin Connect+.", Cmd: &NutritionQuickAddCmd{}},
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

// NutritionQuickAddCmd logs calories and macros in a meal without a food.
type NutritionQuickAddCmd struct {
	Date     Date     `arg:"" help:"Date of the meal (YYYY-MM-DD, today, yesterday, 3d)."`
	Meal     FreeText `help:"Meal: breakfast, lunch, dinner, snacks, or a custom meal you enabled." required:"" short:"m" placeholder:"MEAL"`
	Calories float64  `help:"Calories (kcal)." required:"" placeholder:"KCAL"`
	Carbs    *float64 `help:"Carbohydrates in grams." placeholder:"GRAMS"`
	Protein  *float64 `help:"Protein in grams." placeholder:"GRAMS"`
	Fat      *float64 `help:"Fat in grams." placeholder:"GRAMS"`
	Name     FreeText `help:"Name of the entry, such as Coffee with milk." placeholder:"NAME"`
	Time     Clock    `help:"Meal time (HH:MM). Defaults to the start time of the meal, or to the current time for a meal without one, such as snacks." placeholder:"HH:MM"`
}

func (c *NutritionQuickAddCmd) Validate() error {
	for _, v := range []struct {
		flag  string
		value *float64
	}{{"--calories", &c.Calories}, {"--carbs", c.Carbs}, {"--protein", c.Protein}, {"--fat", c.Fat}} {
		if v.value != nil && *v.value < 0 {
			return fmt.Errorf("%s must not be negative, got %g", v.flag, *v.value)
		}
	}
	return nil
}

func (c *NutritionQuickAddCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	entry, err := mealEntry(g, client, c.Date, c.Meal, c.Time)
	if err != nil {
		return err
	}
	data, err := client.QuickAdd(g.Context, garminapi.QuickAddEntry{
		MealEntry: entry, Name: string(c.Name),
		Calories: c.Calories, Carbs: c.Carbs, Protein: c.Protein, Fat: c.Fat,
	})
	if err != nil {
		return fmt.Errorf("quick add: %w", err)
	}
	return reportWrite(g, data, "Added %g kcal to %s on %s%s", c.Calories, mealName(entry.Meal), c.Date, logIDNote(data, entry))
}

// mealEntry resolves the meal named name on date and the meal time: at, or
// the start time of the meal, or the current time for a meal without one.
func mealEntry(g *Globals, client *garminapi.Client, date Date, name FreeText, at Clock) (garminapi.MealEntry, error) {
	meals, err := client.Meals(g.Context, string(date))
	if err != nil {
		return garminapi.MealEntry{}, fmt.Errorf("get meals: %w", err)
	}
	meal, err := findMeal(meals, string(name), string(date))
	if err != nil {
		return garminapi.MealEntry{}, err
	}
	now := nowFn()
	t := string(at)
	switch {
	case t != "":
	case meal.StartTime != "":
		t = meal.StartTime[:min(len(meal.StartTime), len("15:04"))]
	default:
		t = now.Format("15:04")
	}
	return garminapi.MealEntry{Date: string(date), Meal: meal, Time: t, LoggedAt: now}, nil
}

func findMeal(meals []garminapi.Meal, name, date string) (garminapi.Meal, error) {
	if len(meals) == 0 {
		return garminapi.Meal{}, fmt.Errorf("no meals on %s: set up nutrition in Garmin Connect first (it needs a Garmin Connect+ subscription)", date)
	}
	want := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(name)), "_", "-")
	var names []string
	for _, m := range meals {
		if mealName(m) == want {
			return m, nil
		}
		names = append(names, mealName(m))
	}
	return garminapi.Meal{}, fmt.Errorf("no meal %q on %s: choose %s", name, date, orList(names))
}

func orList(words []string) string {
	if len(words) < 2 {
		return strings.Join(words, "")
	}
	return strings.Join(words[:len(words)-1], ", ") + " or " + words[len(words)-1]
}

// mealName is the name of m as --meal takes it, such as snacks or
// custom-meal-1.
func mealName(m garminapi.Meal) string {
	return strings.ReplaceAll(strings.ToLower(m.Name), "_", "-")
}

// logIDNote names the entry that a log write added, found in the day log
// that Garmin answers by the time of the write.
func logIDNote(dayLog []byte, e garminapi.MealEntry) string {
	foods, err := garminapi.ParseFoodLog(dayLog)
	if err != nil {
		return ""
	}
	var ids []string
	for _, f := range foods {
		if f.LogTimestamp == garminapi.LogTimestamp(e.LoggedAt) {
			ids = append(ids, f.LogID)
		}
	}
	if len(ids) != 1 {
		return ""
	}
	return " (log ID: " + ids[0] + ")"
}
