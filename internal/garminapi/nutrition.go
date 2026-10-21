package garminapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

var nutritionEndpoints = []Endpoint{
	{Name: "nutrition.daily-food-log", Method: http.MethodGet, Host: ConnectAPI, Path: "/nutrition-service/food/logs/{date}"},
	{Name: "nutrition.daily-meals", Method: http.MethodGet, Host: ConnectAPI, Path: "/nutrition-service/meals/{date}"},
	{Name: "nutrition.daily-settings", Method: http.MethodGet, Host: ConnectAPI, Path: "/nutrition-service/settings/{date}"},
	{Name: "nutrition.food-search", Method: http.MethodGet, Host: ConnectAPI, Path: "/nutrition-service/food/search?searchExpression={query}&start={start}&limit={limit}", Requires: ConnectPlus},
	{Name: "nutrition.recent-foods", Method: http.MethodGet, Host: ConnectAPI, Path: "/nutrition-service/food/recent/{date}?start={start}&limit={limit}", Requires: ConnectPlus},
	{Name: "nutrition.favorite-foods", Method: http.MethodGet, Host: ConnectAPI, Path: "/nutrition-service/favorite?start={start}&limit={limit}&includeContent=true", Requires: ConnectPlus},
	{Name: "nutrition.add-favorite-food", Method: http.MethodPut, Host: ConnectAPI, Path: "/nutrition-service/favorite/food", Body: true, Requires: ConnectPlus},
	{Name: "nutrition.remove-favorite-food", Method: http.MethodDelete, Host: ConnectAPI, Path: "/nutrition-service/favorite/food/{foodId}", Requires: ConnectPlus},
	{Name: "nutrition.log-food", Method: http.MethodPut, Host: ConnectAPI, Path: "/nutrition-service/food/logs", Body: true, Requires: ConnectPlus},
	{Name: "nutrition.delete-food-logs", Method: http.MethodDelete, Host: ConnectAPI, Path: "/nutrition-service/food/logs/{date}", Body: true, Requires: ConnectPlus},
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

// Food is a food of the Garmin food database with the fields that a log
// entry and a favorite need.
type Food struct {
	ID           string    `json:"foodId"`
	Name         string    `json:"foodName"`
	Source       string    `json:"source"`
	RegionCode   string    `json:"regionCode"`
	LanguageCode string    `json:"languageCode"`
	Servings     []Serving `json:"-"`
}

// Serving is one serving size of a food.
type Serving struct {
	ID   string `json:"servingId"`
	Unit string `json:"servingUnit"`
}

const (
	foodSearchPage  = 50
	foodSearchPages = 5
)

// FindFood returns the food with ID id from the results of a food search
// for query. Garmin has no read of one food by ID, and a log entry needs
// the source, region, language and serving of the food. ok is false when
// the first 250 results do not have it.
func (c *Client) FindFood(ctx context.Context, query, id string) (food Food, ok bool, err error) {
	for page := range foodSearchPages {
		args := map[string]string{"query": query, "start": strconv.Itoa(page * foodSearchPage), "limit": strconv.Itoa(foodSearchPage)}
		data, err := c.Call(ctx, "nutrition.food-search", args, nil)
		if err != nil {
			return Food{}, false, err
		}
		var res struct {
			More    bool `json:"moreDataAvailable"`
			Results []struct {
				Food     Food      `json:"foodMetaData"`
				Servings []Serving `json:"nutritionContents"`
			} `json:"results"`
		}
		if len(data) > 0 {
			if err := json.Unmarshal(data, &res); err != nil {
				return Food{}, false, fmt.Errorf("parse foods: %w", err)
			}
		}
		for _, r := range res.Results {
			if r.Food.ID == id {
				r.Food.Servings = r.Servings
				return r.Food, true, nil
			}
		}
		if !res.More {
			break
		}
	}
	return Food{}, false, nil
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
	Time     string
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

// FoodLogEntry is a quantity of one serving of a food logged in a meal.
type FoodLogEntry struct {
	MealEntry
	Food     Food
	Serving  Serving
	Quantity float64
}

// LogFood logs e and returns the food log of its day.
func (c *Client) LogFood(ctx context.Context, e FoodLogEntry) (json.RawMessage, error) {
	type item struct {
		LogID        *string     `json:"logId"`
		LogTimestamp string      `json:"logTimestamp"`
		LogSource    string      `json:"logSource"`
		LogCategory  string      `json:"logCategory"`
		MealTime     string      `json:"mealTime"`
		Action       string      `json:"action"`
		MealID       json.Number `json:"mealId"`
		FoodID       string      `json:"foodId"`
		ServingID    string      `json:"servingId"`
		Source       string      `json:"source"`
		RegionCode   string      `json:"regionCode"`
		LanguageCode string      `json:"languageCode"`
		ServingQty   float64     `json:"servingQty"`
		CustomMealID *string     `json:"customMealId"`
	}
	it := item{
		LogTimestamp: LogTimestamp(e.LoggedAt), LogSource: "GCW", LogCategory: "REGULAR_LOG",
		MealTime: e.mealTime(), Action: "ADD", MealID: e.Meal.ID,
		FoodID: e.Food.ID, ServingID: e.Serving.ID, Source: e.Food.Source,
		RegionCode: e.Food.RegionCode, LanguageCode: e.Food.LanguageCode, ServingQty: e.Quantity,
	}
	body, err := json.Marshal(map[string]any{"mealDate": e.Date, "foodLogItems": []item{it}})
	if err != nil {
		return nil, err
	}
	return c.Call(ctx, "nutrition.log-food", nil, body)
}

// AddFavoriteFood saves a quantity of one serving of food as a favorite.
func (c *Client) AddFavoriteFood(ctx context.Context, food Food, serving Serving, quantity float64) (json.RawMessage, error) {
	favorite := struct {
		FoodID       string  `json:"foodId"`
		ServingID    string  `json:"servingId"`
		Source       string  `json:"source"`
		ServingQty   float64 `json:"servingQty"`
		RegionCode   string  `json:"regionCode"`
		LanguageCode string  `json:"languageCode"`
	}{food.ID, serving.ID, food.Source, quantity, food.RegionCode, food.LanguageCode}
	body, err := json.Marshal(favorite)
	if err != nil {
		return nil, err
	}
	return c.Call(ctx, "nutrition.add-favorite-food", nil, body)
}

const (
	favoritePage  = 50
	favoritePages = 100
)

// IsFavoriteFood reports whether the food with ID id is a favorite food.
func (c *Client) IsFavoriteFood(ctx context.Context, id string) (bool, error) {
	for page := range favoritePages {
		args := map[string]string{"start": strconv.Itoa(page * favoritePage), "limit": strconv.Itoa(favoritePage)}
		data, err := c.Call(ctx, "nutrition.favorite-foods", args, nil)
		if err != nil {
			return false, err
		}
		var res struct {
			HasMore     bool `json:"hasMore"`
			Consumables []struct {
				Food Food `json:"foodMetaData"`
			} `json:"consumables"`
		}
		if len(data) > 0 {
			if err := json.Unmarshal(data, &res); err != nil {
				return false, fmt.Errorf("parse favorite foods: %w", err)
			}
		}
		for _, f := range res.Consumables {
			if f.Food.ID == id {
				return true, nil
			}
		}
		if !res.HasMore {
			break
		}
	}
	return false, nil
}
