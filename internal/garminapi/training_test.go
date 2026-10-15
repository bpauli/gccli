package garminapi

import "testing"

func TestTrainingEndpoints(t *testing.T) {
	testEndpoints(t, trainingEndpoints, []endpointCase{
		{"training.plans", map[string]string{"locale": "de"}, "GET /trainingplan-service/trainingplan/plans?locale=de"},
		{"training.plans", map[string]string{"locale": ""}, "GET /trainingplan-service/trainingplan/plans?locale=en"},
		{"training.plans", nil, "GET /trainingplan-service/trainingplan/plans?locale=en"},
		{"training.plan", map[string]string{"planID": "42"}, "GET /trainingplan-service/trainingplan/plan/42"},
	})
}
