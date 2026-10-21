package garminapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

var nutritionEndpoints = []Endpoint{
	{Name: "nutrition.daily-food-log", Method: http.MethodGet, Host: ConnectAPI, Path: "/nutrition-service/food/logs/{date}"},
	{Name: "nutrition.daily-meals", Method: http.MethodGet, Host: ConnectAPI, Path: "/nutrition-service/meals/{date}"},
	{Name: "nutrition.daily-settings", Method: http.MethodGet, Host: ConnectAPI, Path: "/nutrition-service/settings/{date}"},
	{Name: "nutrition.food-search", Method: http.MethodGet, Host: ConnectAPI, Path: "/nutrition-service/food/search?searchExpression={query}&start={start}&limit={limit}", Requires: ConnectPlus},
	{Name: "nutrition.recent-foods", Method: http.MethodGet, Host: ConnectAPI, Path: "/nutrition-service/food/recent/{date}?start={start}&limit={limit}", Requires: ConnectPlus},
	{Name: "nutrition.favorite-foods", Method: http.MethodGet, Host: ConnectAPI, Path: "/nutrition-service/favorite?start={start}&limit={limit}&includeContent=true", Requires: ConnectPlus},
	{Name: "nutrition.quick-add", Method: http.MethodPut, Host: ConnectAPI, Path: "/nutrition-service/food/logs/quickAdd", Body: true, Requires: ConnectPlus},
}

// Meal is one meal of a day in the nutrition settings of the user.
type Meal struct {
	ID json.Number `json:"mealId"`
	// Name is BREAKFAST, LUNCH, DINNER, SNACKS or CUSTOM_MEAL_<n>.
	Name string `json:"mealName"`
	// StartTime is HH:MM:SS, or empty for a meal without a time window,
	// such as SNACKS.
	StartTime string `json:"startTime"`
	Enabled   bool   `json:"enabled"`
}

// Meals returns the enabled meals of date. A user who has not set up
// nutrition has none.
func (c *Client) Meals(ctx context.Context, date string) ([]Meal, error) {
	data, err := c.Call(ctx, "nutrition.daily-meals", map[string]string{"date": date}, nil)
	if err != nil {
		return nil, err
	}
	var day struct {
		Meals []Meal `json:"meals"`
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &day); err != nil {
			return nil, fmt.Errorf("parse meals: %w", err)
		}
	}
	var enabled []Meal
	for _, m := range day.Meals {
		if m.Enabled {
			enabled = append(enabled, m)
		}
	}
	return enabled, nil
}

// LoggedFood is one entry of the food log of a day.
type LoggedFood struct {
	LogID        string `json:"logId"`
	LogTimestamp string `json:"logTimestamp"`
}

// ParseFoodLog returns the entries of a day food log, the answer of
// nutrition.daily-food-log and of each log write.
func ParseFoodLog(data json.RawMessage) ([]LoggedFood, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var day struct {
		MealDetails []struct {
			LoggedFoods []LoggedFood `json:"loggedFoods"`
		} `json:"mealDetails"`
	}
	if err := json.Unmarshal(data, &day); err != nil {
		return nil, fmt.Errorf("parse food log: %w", err)
	}
	var foods []LoggedFood
	for _, m := range day.MealDetails {
		foods = append(foods, m.LoggedFoods...)
	}
	return foods, nil
}

// LogTimestamp is the time of a log write as Garmin stores it: ISO 8601 in
// UTC with milliseconds, as the web app sends it. Garmin reads a number as
// epoch seconds and refuses a time without a zone (live 2026-10-21).
func LogTimestamp(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000Z") }

// MealEntry places a log entry in a meal of a day.
type MealEntry struct {
	Date string
	Meal Meal
	// Time is the meal time, HH:MM.
	Time string
	// LoggedAt is the time of the write.
	LoggedAt time.Time
}

func (e MealEntry) mealTime() string { return e.Time + ":00" }

// QuickAddEntry is calories and macros logged in a meal without a food.
// A nil macro and an empty name are sent as null.
type QuickAddEntry struct {
	MealEntry
	Name     string
	Calories float64
	Carbs    *float64
	Protein  *float64
	Fat      *float64
}

// QuickAdd logs e and returns the food log of its day.
func (c *Client) QuickAdd(ctx context.Context, e QuickAddEntry) (json.RawMessage, error) {
	type item struct {
		Name         *string     `json:"name"`
		LogID        *string     `json:"logId"`
		LogTimestamp string      `json:"logTimestamp"`
		LogSource    string      `json:"logSource"`
		LogCategory  string      `json:"logCategory"`
		MealTime     string      `json:"mealTime"`
		MealID       json.Number `json:"mealId"`
		Action       string      `json:"action"`
		Calories     float64     `json:"calories"`
		Carbs        *float64    `json:"carbs"`
		Protein      *float64    `json:"protein"`
		Fat          *float64    `json:"fat"`
	}
	it := item{
		LogTimestamp: LogTimestamp(e.LoggedAt), LogSource: "GCW", LogCategory: "QUICK_ADD",
		MealTime: e.mealTime(), MealID: e.Meal.ID, Action: "ADD",
		Calories: e.Calories, Carbs: e.Carbs, Protein: e.Protein, Fat: e.Fat,
	}
	if e.Name != "" {
		it.Name = &e.Name
	}
	body, err := json.Marshal(map[string]any{"mealDate": e.Date, "quickAddItems": []item{it}})
	if err != nil {
		return nil, err
	}
	return c.Call(ctx, "nutrition.quick-add", nil, body)
}
