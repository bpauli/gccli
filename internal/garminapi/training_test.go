package garminapi

import "testing"

func TestTrainingEndpoints(t *testing.T) {
	testEndpoints(t, trainingEndpoints, []endpointCase{
		{"training.plans", map[string]string{"locale": "de"}, "GET /trainingplan-service/trainingplan/plans?locale=de"},
		{"training.plans", map[string]string{"locale": ""}, "GET /trainingplan-service/trainingplan/plans?locale=en"},
		{"training.plans", nil, "GET /trainingplan-service/trainingplan/plans?locale=en"},
		{"training.plan", map[string]string{"planID": "42"}, "GET /trainingplan-service/trainingplan/plan/42"},
		{"training.active-plans", map[string]string{"startDate": "2024-01-15"}, "GET /trainingplan-service/trainingplan/active-plans?startDate=2024-01-15"},
	})
}
