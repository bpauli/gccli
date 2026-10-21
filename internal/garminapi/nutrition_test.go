package garminapi

import (
	"context"
	"net/http"
	"slices"
	"testing"
	"time"
)

func TestNutritionEndpoints(t *testing.T) {
	day := map[string]string{"date": "2024-01-15"}
	testEndpoints(t, nutritionEndpoints, []endpointCase{
		{"nutrition.daily-food-log", day, "GET /nutrition-service/food/logs/2024-01-15"},
		{"nutrition.daily-meals", day, "GET /nutrition-service/meals/2024-01-15"},
		{"nutrition.daily-settings", day, "GET /nutrition-service/settings/2024-01-15"},
		{"nutrition.food-search", map[string]string{"query": "apple pie", "start": "0", "limit": "20"}, "GET /nutrition-service/food/search?searchExpression=apple+pie&start=0&limit=20"},
		{"nutrition.recent-foods", map[string]string{"date": "2024-01-15", "start": "0", "limit": "20"}, "GET /nutrition-service/food/recent/2024-01-15?start=0&limit=20"},
		{"nutrition.favorite-foods", map[string]string{"start": "0", "limit": "20"}, "GET /nutrition-service/favorite?start=0&limit=20&includeContent=true"},
		{"nutrition.quick-add", nil, "PUT /nutrition-service/food/logs/quickAdd"},
	})
}

// TestNutritionConnectPlusRows pins which nutrition rows Garmin gates behind
// Connect+ (live 2026-10-21): the day log, meals and settings reads pass
// without it, and the food database, recent foods, favorites and every
// write do not.
func TestNutritionConnectPlusRows(t *testing.T) {
	gated := []string{"nutrition.food-search", "nutrition.recent-foods", "nutrition.favorite-foods", "nutrition.quick-add"}
	for _, e := range nutritionEndpoints {
		if want := slices.Contains(gated, e.Name); (e.Requires == ConnectPlus) != want {
			t.Errorf("%s: Requires = %v, want Connect+ %v", e.Name, e.Requires, want)
		}
	}
}

func TestMealsKeepsTheEnabledMeals(t *testing.T) {
	_, client := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"meals":[{"mealId":1,"mealName":"BREAKFAST","startTime":"07:00:00","enabled":true},{"mealId":2,"mealName":"CUSTOM_MEAL_1","enabled":false},{"mealId":3,"mealName":"SNACKS","startTime":null,"enabled":true}]}`))
	}))
	meals, err := client.Meals(context.Background(), "2026-10-15")
	if err != nil {
		t.Fatal(err)
	}
	want := []Meal{{ID: "1", Name: "BREAKFAST", StartTime: "07:00:00", Enabled: true}, {ID: "3", Name: "SNACKS", Enabled: true}}
	if !slices.Equal(meals, want) {
		t.Errorf("Meals = %+v, want %+v", meals, want)
	}
}

func TestMealsWithoutNutritionSetup(t *testing.T) {
	for name, body := range map[string]string{"empty list": `{"meals":[]}`, "no body": ""} {
		t.Run(name, func(t *testing.T) {
			_, client := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if body == "" {
					w.WriteHeader(http.StatusNoContent)
				}
				_, _ = w.Write([]byte(body))
			}))
			meals, err := client.Meals(context.Background(), "2026-10-15")
			if err != nil || len(meals) != 0 {
				t.Errorf("Meals = %v, %v, want no meals and no error", meals, err)
			}
		})
	}
}

func TestParseFoodLogJoinsTheMeals(t *testing.T) {
	foods, err := ParseFoodLog([]byte(`{"mealDetails":[{"loggedFoods":[{"logId":"a","logTimestamp":"t1"}]},{"loggedFoods":[]},{"loggedFoods":[{"logId":"b","logTimestamp":"t2"},{"logId":"c","logTimestamp":"t3"}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	want := []LoggedFood{{"a", "t1"}, {"b", "t2"}, {"c", "t3"}}
	if !slices.Equal(foods, want) {
		t.Errorf("ParseFoodLog = %+v, want %+v", foods, want)
	}
	if _, err := ParseFoodLog([]byte(`[]`)); err == nil {
		t.Error("ParseFoodLog([]) = nil error, want a parse error")
	}
}

func TestLogTimestampIsUTCWithMilliseconds(t *testing.T) {
	at := time.Date(2026, 10, 21, 15, 5, 9, 123456789, time.FixedZone("CEST", 2*60*60))
	if got, want := LogTimestamp(at), "2026-10-21T13:05:09.123Z"; got != want {
		t.Errorf("LogTimestamp = %q, want %q", got, want)
	}
}
