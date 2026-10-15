package garminapi

import "net/http"

var goalsEndpoints = []Endpoint{
	{
		Name: "goals.list", Method: http.MethodGet, Host: ConnectAPI,
		Path:   "/goal-service/goal/goals?status={status}",
		Params: []Param{{Name: "status", Enum: []string{"active", "future", "past"}}},
	},
	{Name: "badges.earned", Method: http.MethodGet, Host: ConnectAPI, Path: "/badge-service/badge/earned"},
	{Name: "badges.available", Method: http.MethodGet, Host: ConnectAPI, Path: "/badge-service/badge/available"},
	{Name: "badges.in-progress", Method: http.MethodGet, Host: ConnectAPI, Path: "/badge-service/badge/in-progress"},
	{Name: "badges.challenges", Method: http.MethodGet, Host: ConnectAPI, Path: "/badge-service/badge/challenges"},
	{Name: "challenges.joined", Method: http.MethodGet, Host: ConnectAPI, Path: "/challenge-service/challenge/joined"},
	{Name: "records.list", Method: http.MethodGet, Host: ConnectAPI, Path: "/personalrecord-service/personalrecord/prs/{displayName}"},
}
