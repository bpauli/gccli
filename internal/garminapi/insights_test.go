package garminapi

import "testing"

func TestInsightsEndpoints(t *testing.T) {
	compare := func(extra map[string]string) map[string]string {
		args := map[string]string{"age": "40-44", "type": "RUNNING", "field": "VO2_MAX_VALUE", "year": "2025"}
		for k, v := range extra {
			args[k] = v
		}
		return args
	}
	testEndpoints(t, insightsEndpoints, []endpointCase{
		{"insights.selections", nil, "GET /gcs-insights/api/demographics/selections?lang=en&localeUnit=METRIC"},
		{"insights.selections", map[string]string{"lang": "de", "localeUnit": "STATUTE"}, "GET /gcs-insights/api/demographics/selections?lang=de&localeUnit=STATUTE"},
		{"insights.compare", compare(nil), "GET /gcs-insights/api/demographics/compare?gender=ALL&age=40-44&type=RUNNING&field=VO2_MAX_VALUE&lang=en&localeUnit=METRIC&year=2025"},
		{
			"insights.compare", compare(map[string]string{"gender": "F", "age": "75+", "localeUnit": "STATUTE", "month": "3"}),
			"GET /gcs-insights/api/demographics/compare?gender=F&age=75%2B&type=RUNNING&field=VO2_MAX_VALUE&lang=en&localeUnit=STATUTE&year=2025&month=3",
		},
		{"insights.preferences", nil, "GET /userpreference-service/demographicComparisonPreferences"},
	})
}
