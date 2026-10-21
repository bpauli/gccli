package cmd

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
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
	Custom{Name: "add", Help: "Log a food of the Garmin food database in a meal. Needs Garmin Connect+.", Cmd: &NutritionAddCmd{}},
	Custom{Name: "quick-add", Help: "Log calories and macros in a meal without a food. Needs Garmin Connect+.", Cmd: &NutritionQuickAddCmd{}},
	Custom{Name: "delete", Help: "Delete food log entries of a day. Needs Garmin Connect+.", Cmd: &NutritionDeleteCmd{}},
	Group{Name: "favorites", Help: "List, add and remove favorite foods. Needs Garmin Connect+.", Children: []Node{
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
		Custom{Name: "add", Help: "Add a food to your favorite foods. Needs Garmin Connect+.", Cmd: &NutritionFavoritesAddCmd{}},
		Custom{Name: "remove", Help: "Remove a food from your favorite foods. Needs Garmin Connect+.", Cmd: &NutritionFavoritesRemoveCmd{}},
	}},
}}

// NutritionAddCmd logs a food of the Garmin food database in a meal.
type NutritionAddCmd struct {
	Date     Date     `arg:"" help:"Date of the meal (YYYY-MM-DD, today, yesterday, 3d)."`
	FoodID   ID       `arg:"" name:"food-id" help:"Food ID, as nutrition search prints it."`
	Query    FreeText `help:"Search text that finds the food, as you gave it to nutrition search. Garmin has no read of one food by ID." required:"" short:"q" placeholder:"TEXT"`
	Meal     FreeText `help:"Meal: breakfast, lunch, dinner, snacks, or a custom meal you enabled." required:"" short:"m" placeholder:"MEAL"`
	Serving  string   `help:"Serving unit, such as g or oz, or serving ID, as nutrition search --json lists them under nutritionContents. Defaults to the first serving of the food." placeholder:"UNIT"`
	Servings float64  `help:"Number of servings." default:"1"`
	Time     Clock    `help:"Meal time (HH:MM). Defaults to the start time of the meal, or to the current time for a meal without one, such as snacks." placeholder:"HH:MM"`
}

func (c *NutritionAddCmd) Validate() error {
	if c.Servings <= 0 {
		return fmt.Errorf("--servings must be more than 0, got %g", c.Servings)
	}
	return nil
}

func (c *NutritionAddCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	food, serving, err := findFood(g, client, string(c.Query), string(c.FoodID), c.Serving)
	if err != nil {
		return err
	}
	entry, err := mealEntry(g, client, c.Date, c.Meal, c.Time)
	if err != nil {
		return err
	}
	data, err := client.LogFood(g.Context, garminapi.FoodLogEntry{MealEntry: entry, Food: food, Serving: serving, Quantity: c.Servings})
	if err != nil {
		return fmt.Errorf("log food: %w", err)
	}
	return reportWrite(g, data, "Added %s (%g x %s) to %s on %s%s", food.Name, c.Servings, serving.Unit, mealName(entry.Meal), c.Date, logIDNote(data, entry))
}

// findFood finds the food with ID id in the results of a search for query,
// and its serving with the unit or ID serving, or its first serving.
func findFood(g *Globals, client *garminapi.Client, query, id, serving string) (garminapi.Food, garminapi.Serving, error) {
	food, ok, err := client.FindFood(g.Context, query, id)
	if err != nil {
		return food, garminapi.Serving{}, fmt.Errorf("find food: %w", err)
	}
	if !ok {
		return food, garminapi.Serving{}, fmt.Errorf("no food %s in the search results for %q: give --query the search text with which nutrition search lists it", id, query)
	}
	if len(food.Servings) == 0 {
		return food, garminapi.Serving{}, fmt.Errorf("food %s has no serving", id)
	}
	if serving == "" {
		return food, food.Servings[0], nil
	}
	var units []string
	for _, s := range food.Servings {
		if s.ID == serving || strings.EqualFold(s.Unit, strings.TrimSpace(serving)) {
			return food, s, nil
		}
		units = append(units, strconv.Quote(s.Unit))
	}
	return food, garminapi.Serving{}, fmt.Errorf("food %s has no serving %q: choose %s", id, serving, orList(units))
}

// NutritionFavoritesAddCmd saves a food of the Garmin food database as a
// favorite.
type NutritionFavoritesAddCmd struct {
	FoodID   ID       `arg:"" name:"food-id" help:"Food ID, as nutrition search prints it."`
	Query    FreeText `help:"Search text that finds the food, as you gave it to nutrition search. Garmin has no read of one food by ID." required:"" short:"q" placeholder:"TEXT"`
	Serving  string   `help:"Serving unit, such as g or oz, or serving ID, as nutrition search --json lists them under nutritionContents. Defaults to the first serving of the food." placeholder:"UNIT"`
	Servings float64  `help:"Number of servings." default:"1"`
}

func (c *NutritionFavoritesAddCmd) Validate() error {
	if c.Servings <= 0 {
		return fmt.Errorf("--servings must be more than 0, got %g", c.Servings)
	}
	return nil
}

func (c *NutritionFavoritesAddCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	food, serving, err := findFood(g, client, string(c.Query), string(c.FoodID), c.Serving)
	if err != nil {
		return err
	}
	data, err := client.AddFavoriteFood(g.Context, food, serving, c.Servings)
	if err != nil {
		return fmt.Errorf("add favorite food: %w", err)
	}
	return reportWrite(g, data, "Added %s (%g x %s) to favorite foods", food.Name, c.Servings, serving.Unit)
}

// NutritionFavoritesRemoveCmd removes a food from the favorite foods.
type NutritionFavoritesRemoveCmd struct {
	FoodID ID `arg:"" name:"food-id" help:"Food ID, as nutrition favorites prints it."`
}

func (c *NutritionFavoritesRemoveCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	// A DELETE of a food that is not a favorite also answers 204, so the
	// favorites tell whether there is one to remove.
	ok, err := client.IsFavoriteFood(g.Context, string(c.FoodID))
	if err != nil {
		return fmt.Errorf("list favorite foods: %w", err)
	}
	if !ok {
		return fmt.Errorf("food %s is not a favorite food", c.FoodID)
	}
	data, err := client.Call(g.Context, "nutrition.remove-favorite-food", map[string]string{"foodId": string(c.FoodID)}, nil)
	if err != nil {
		return fmt.Errorf("remove favorite food: %w", err)
	}
	return reportWrite(g, data, "Removed food %s from favorite foods", c.FoodID)
}

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

// NutritionDeleteCmd deletes food log entries of a day.
type NutritionDeleteCmd struct {
	Date   Date `arg:"" help:"Date of the entries (YYYY-MM-DD, today, yesterday, 3d)."`
	LogIDs []ID `arg:"" name:"log-id" help:"Log ID of an entry, as the add commands print it or nutrition food-log --json lists it."`
	Force  bool `help:"Skip confirmation prompt." short:"f"`
}

func (c *NutritionDeleteCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	date := map[string]string{"date": string(c.Date)}
	// A DELETE of a log ID that does not exist also answers 204, so the day
	// log tells whether each entry is there.
	data, err := client.Call(g.Context, "nutrition.daily-food-log", date, nil)
	if err != nil {
		return fmt.Errorf("get food log: %w", err)
	}
	logged, err := garminapi.ParseFoodLog(data)
	if err != nil {
		return err
	}
	var ids []string
	for _, id := range c.LogIDs {
		if !slices.ContainsFunc(logged, func(f garminapi.LoggedFood) bool { return f.LogID == string(id) }) {
			return fmt.Errorf("no food log entry %s on %s", id, c.Date)
		}
		if !slices.Contains(ids, string(id)) {
			ids = append(ids, string(id))
		}
	}
	what := "food log entry " + ids[0]
	if len(ids) > 1 {
		what = strconv.Itoa(len(ids)) + " food log entries"
	}
	ok, err := confirm(g.Stderr, fmt.Sprintf("Delete %s on %s?", what, c.Date), c.Force)
	if err != nil {
		return err
	}
	if !ok {
		g.UI.Infof("Cancelled")
		return nil
	}
	body, err := json.Marshal(map[string][]string{"logIds": ids})
	if err != nil {
		return err
	}
	data, err = client.Call(g.Context, "nutrition.delete-food-logs", date, body)
	if err != nil {
		return fmt.Errorf("delete food log entries: %w", err)
	}
	return reportWrite(g, data, "Deleted %s on %s", what, c.Date)
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
