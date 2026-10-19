package garminapi

import "net/http"

var goalsEndpoints = []Endpoint{
	{
		Name: "goals.list", Method: http.MethodGet, Host: ConnectAPI,
		Path:   "/goal-service/goal/goals?status={status}",
		Params: []Param{{Name: "status", Enum: []string{"active", "future", "past"}}},
	},
	{Name: "goals.effective-weight-goal", Method: http.MethodGet, Host: ConnectAPI, Path: "/goal-service/goal/user/effective/weightgoal/{startDate}/{endDate}"},
	{Name: "badges.earned", Method: http.MethodGet, Host: ConnectAPI, Path: "/badge-service/badge/earned"},
	{Name: "badges.available", Method: http.MethodGet, Host: ConnectAPI, Path: "/badge-service/badge/available"},
	{Name: "badges.in-progress", Method: http.MethodGet, Host: ConnectAPI, Path: "/badge-service/badge/in-progress"},
	{Name: "badges.detail", Method: http.MethodGet, Host: ConnectAPI, Path: "/badge-service/badge/detail/v2/{badgeID}"},
	{Name: "badges.compare", Method: http.MethodGet, Host: ConnectAPI, Path: "/badge-service/badge/compare/{displayName}"},
	{Name: "badges.leaderboard", Method: http.MethodGet, Host: ConnectAPI, Path: "/badge-service/badge/leaderboard"},
	{Name: "badges.challenges", Method: http.MethodGet, Host: ConnectAPI, Path: "/badge-service/badge/challenges"},
	{Name: "challenges.joined", Method: http.MethodGet, Host: ConnectAPI, Path: "/challenge-service/challenge/joined"},
	{Name: "challenges.available", Method: http.MethodGet, Host: ConnectAPI, Path: "/badgechallenge-service/badgeChallenge/available"},
	{Name: "challenges.completed", Method: http.MethodGet, Host: ConnectAPI, Path: "/badgechallenge-service/badgeChallenge/completed"},
	{Name: "challenges.virtual-available", Method: http.MethodGet, Host: ConnectAPI, Path: "/badgechallenge-service/virtualChallenge/available"},
	{Name: "challenges.virtual-completed", Method: http.MethodGet, Host: ConnectAPI, Path: "/badgechallenge-service/virtualChallenge/completed"},
	{Name: "challenges.virtual-in-progress", Method: http.MethodGet, Host: ConnectAPI, Path: "/badgechallenge-service/virtualChallenge/inProgress"},
	{Name: "records.list", Method: http.MethodGet, Host: ConnectAPI, Path: "/personalrecord-service/personalrecord/prs/{displayName}"},
	{Name: "records.types", Method: http.MethodGet, Host: ConnectAPI, Path: "/personalrecord-service/personalrecordtype/prtypes/{displayName}"},
	{Name: "records.activity", Method: http.MethodGet, Host: ConnectAPI, Path: "/personalrecord-service/personalrecord/prByActivityId/{activityID}"},
}
