package garminapi

import (
	"context"
	"encoding/json"
	"net/http"
)

var goalsEndpoints = []Endpoint{
	{
		Name: "goals.list", Method: http.MethodGet, Host: ConnectAPI,
		Path:   "/goal-service/goal/goals?status={status}",
		Params: []Param{{Name: "status", Enum: []string{"active", "future", "past"}}},
	},
	{Name: "goals.effective-weight-goal", Method: http.MethodGet, Host: ConnectAPI, Path: "/goal-service/goal/user/effective/weightgoal/{startDate}/{endDate}"},
	{
		Name: "goals.set-effective", Method: http.MethodPut, Host: ConnectAPI,
		Path: "/goal-service/goal/user/effective/{goalType}/{date}", Body: true,
		Params: []Param{{Name: "goalType", Enum: []string{"floors-climbed", "intensity-minutes", "hydration"}}},
	},
	{Name: "goals.steps", Method: http.MethodGet, Host: ConnectAPI, Path: "/wellness-service/wellness/wellness-goals/consolidated/steps/{date}"},
	{Name: "goals.set-steps", Method: http.MethodPut, Host: ConnectAPI, Path: "/wellness-service/wellness/wellness-goals/consolidated/steps/{date}", Body: true},
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

// StepGoal is a daily steps goal: ManualStepGoal or AutoStepGoal.
type StepGoal struct {
	steps    int
	category string
}

// ManualStepGoal is a fixed goal of steps per day.
func ManualStepGoal(steps int) StepGoal { return StepGoal{steps: steps, category: "manual"} }

// AutoStepGoal is Garmin's automatic goal. Garmin stores it as goal 0 with
// the category my-auto.
var AutoStepGoal = StepGoal{steps: 0, category: "my-auto"}

// SetStepGoal reads the steps goal of date, sets the goal and its category
// and sends the full object back, as the web app does.
func (c *Client) SetStepGoal(ctx context.Context, date string, goal StepGoal) (json.RawMessage, error) {
	params := map[string]string{"date": date}
	data, err := c.Call(ctx, "goals.steps", params, nil)
	if err != nil {
		return nil, err
	}
	body, err := setFields(data, "steps goal", map[string]any{"userGoal": goal.steps, "userGoalCategory": goal.category})
	if err != nil {
		return nil, err
	}
	return c.Call(ctx, "goals.set-steps", params, body)
}
