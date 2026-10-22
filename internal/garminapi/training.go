package garminapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

var trainingEndpoints = []Endpoint{
	{
		Name: "training.plans", Method: http.MethodGet, Host: ConnectAPI,
		Path:   "/trainingplan-service/trainingplan/plans?locale={locale}",
		Params: []Param{{Name: "locale", Default: "en"}},
	},
	{Name: "training.plan", Method: http.MethodGet, Host: ConnectAPI, Path: "/trainingplan-service/trainingplan/{planID}"},
	{Name: "training.active-plans", Method: http.MethodGet, Host: ConnectAPI, Path: "/trainingplan-service/trainingplan/active-plans?startDate={startDate}"},
	{
		Name: "training.schedule", Method: http.MethodGet, Host: ConnectAPI,
		Path:   "/trainingplan-service/trainingplan/tasks/{planID}?localeKey={locale}",
		Params: []Param{{Name: "locale", Default: "en"}},
	},
	// training.join is a GET, but it adds a copy of the plan to the account
	// and schedules its workouts from startDate.
	{Name: "training.join", Method: http.MethodGet, Host: ConnectAPI, Path: "/trainingplan-service/trainingplan/schedule/{planID}?startDate={startDate}"},
	{Name: "training.leave", Method: http.MethodDelete, Host: ConnectAPI, Path: "/trainingplan-service/trainingplan/trainingplan/{planID}"},
	{Name: "training.delete-task", Method: http.MethodDelete, Host: ConnectAPI, Path: "/trainingplan-service/trainingplan/workoutTask/{planID}/{taskID}"},
	{Name: "training.adaptive-coaching-workouts", Method: http.MethodGet, Host: ConnectAPI, Path: "/userprofile-service/userprofile/adaptive-coaching-workouts"},
}

// formContentType is the type of the training plan requests that Garmin
// reads as a form. It answers 415 to the same request without it.
const formContentType = "application/x-www-form-urlencoded"

// TrainingPlanFilter selects plans from the Garmin catalog. Each field takes
// Garmin's own value, such as Running, 5k, Beginner or HeartRate. An empty
// field does not filter.
type TrainingPlanFilter struct {
	Sport, Goal, Level, Target string
	Locale                     string
}

// SearchTrainingPlans lists the catalog plans that match f. The search is a
// POST with the filter in the query and an empty form body. It asks for the
// first 1000 plans, as the web app does.
func (c *Client) SearchTrainingPlans(ctx context.Context, f TrainingPlanFilter) (json.RawMessage, error) {
	q := url.Values{"start": {"1"}, "limit": {"1000"}, "locale": {"en"}}
	for key, v := range map[string]string{"locale": f.Locale, "types": f.Sport, "subTypes": f.Goal, "levels": f.Level, "version": f.Target} {
		if v != "" {
			q.Set(key, v)
		}
	}
	return c.sendForm(ctx, "/trainingplan-service/trainingplan/search?"+q.Encode(), nil)
}

// MoveTrainingPlanTask moves a workout of a joined plan to date. Garmin
// gives the moved workout a new task ID, which the answer holds.
func (c *Client) MoveTrainingPlanTask(ctx context.Context, planID, taskID, date string) (json.RawMessage, error) {
	form := url.Values{"workoutSchedulePk": {taskID}, "scheduleDate": {date}}
	return c.sendForm(ctx, "/trainingplan-service/trainingplan/moveWorkoutTask/"+url.PathEscape(planID), form)
}

func (c *Client) sendForm(ctx context.Context, path string, form url.Values) (json.RawMessage, error) {
	resp, err := c.do(ctx, request{
		kind: kindJSON, method: http.MethodPost, path: path,
		body: []byte(form.Encode()), contentType: formContentType,
	})
	if err != nil {
		return nil, err
	}
	return jsonResult(resp), nil
}
