package garminapi

import (
	"slices"
	"testing"
)

func TestNutritionEndpoints(t *testing.T) {
	day := map[string]string{"date": "2024-01-15"}
	testEndpoints(t, nutritionEndpoints, []endpointCase{
		{"nutrition.daily-food-log", day, "GET /nutrition-service/food/logs/2024-01-15"},
		{"nutrition.daily-meals", day, "GET /nutrition-service/meals/2024-01-15"},
		{"nutrition.daily-settings", day, "GET /nutrition-service/settings/2024-01-15"},
		{"nutrition.food-search", map[string]string{"query": "apple pie", "start": "0", "limit": "20"}, "GET /nutrition-service/food/search?searchExpression=apple+pie&start=0&limit=20"},
		{"nutrition.recent-foods", map[string]string{"date": "2024-01-15", "start": "0", "limit": "20"}, "GET /nutrition-service/food/recent/2024-01-15?start=0&limit=20"},
	})
}

// TestNutritionConnectPlusRows pins which nutrition rows Garmin gates behind
// Connect+ (live 2026-10-21): the day log, meals and settings reads pass
// without it, and the food database, recent foods and favorites do not.
func TestNutritionConnectPlusRows(t *testing.T) {
	gated := []string{"nutrition.food-search", "nutrition.recent-foods"}
	for _, e := range nutritionEndpoints {
		if want := slices.Contains(gated, e.Name); (e.Requires == ConnectPlus) != want {
			t.Errorf("%s: Requires = %v, want Connect+ %v", e.Name, e.Requires, want)
		}
	}
}
