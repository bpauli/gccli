package garminapi

import "testing"

func TestNutritionEndpoints(t *testing.T) {
	day := map[string]string{"date": "2024-01-15"}
	testEndpoints(t, nutritionEndpoints, []endpointCase{
		{"nutrition.daily-food-log", day, "GET /nutrition-service/food/logs/2024-01-15"},
		{"nutrition.daily-meals", day, "GET /nutrition-service/meals/2024-01-15"},
		{"nutrition.daily-settings", day, "GET /nutrition-service/settings/2024-01-15"},
	})
}
