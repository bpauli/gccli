package garminapi

import "net/http"

var trainingEndpoints = []Endpoint{
	{
		Name: "training.plans", Method: http.MethodGet, Host: ConnectAPI,
		Path:   "/trainingplan-service/trainingplan/plans?locale={locale}",
		Params: []Param{{Name: "locale", Default: "en"}},
	},
	{Name: "training.plan", Method: http.MethodGet, Host: ConnectAPI, Path: "/trainingplan-service/trainingplan/plan/{planID}"},
}
