package garminapi

import "net/http"

var trainingEndpoints = []Endpoint{
	{
		Name: "training.plans", Method: http.MethodGet, Host: ConnectAPI,
		Path:   "/trainingplan-service/trainingplan/plans?locale={locale}",
		Params: []Param{{Name: "locale", Default: "en"}},
	},
	{Name: "training.plan", Method: http.MethodGet, Host: ConnectAPI, Path: "/trainingplan-service/trainingplan/{planID}"},
	{Name: "training.active-plans", Method: http.MethodGet, Host: ConnectAPI, Path: "/trainingplan-service/trainingplan/active-plans?startDate={startDate}"},
	// training.search takes the filter in the query and an empty form body.
	// Garmin's start and limit counts are not consistent, so the row asks
	// for the first 1000 plans, as the web app does.
	{
		Name: "training.search", Method: http.MethodPost, Host: ConnectAPI,
		Path: "/trainingplan-service/trainingplan/search?start=1&limit=1000&locale={locale}&types={sport}&subTypes={goal}&levels={level}&version={target}",
		Params: []Param{
			{Name: "locale", Default: "en"},
			{Name: "sport", Optional: true},
			{Name: "goal", Optional: true},
			{Name: "level", Optional: true},
			{Name: "target", Optional: true},
		},
		Body: true, Form: true, Read: true,
	},
	{
		Name: "training.schedule", Method: http.MethodGet, Host: ConnectAPI,
		Path:   "/trainingplan-service/trainingplan/tasks/{planID}?localeKey={locale}",
		Params: []Param{{Name: "locale", Default: "en"}},
	},
	// training.join is a GET, but it adds a copy of the plan to the account
	// and schedules its workouts from startDate.
	{Name: "training.join", Method: http.MethodGet, Host: ConnectAPI, Path: "/trainingplan-service/trainingplan/schedule/{planID}?startDate={startDate}"},
	{Name: "training.leave", Method: http.MethodDelete, Host: ConnectAPI, Path: "/trainingplan-service/trainingplan/trainingplan/{planID}"},
	{Name: "training.move-task", Method: http.MethodPost, Host: ConnectAPI, Path: "/trainingplan-service/trainingplan/moveWorkoutTask/{planID}", Body: true, Form: true},
	{Name: "training.delete-task", Method: http.MethodDelete, Host: ConnectAPI, Path: "/trainingplan-service/trainingplan/workoutTask/{planID}/{taskID}"},
	{Name: "training.adaptive-coaching-workouts", Method: http.MethodGet, Host: ConnectAPI, Path: "/userprofile-service/userprofile/adaptive-coaching-workouts"},
}
