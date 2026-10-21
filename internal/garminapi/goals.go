package garminapi

import (
	"context"
	"encoding/json"
	"fmt"
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
	{
		Name: "badges.available", Method: http.MethodGet, Host: ConnectAPI,
		Path:   "/badge-service/badge/available?showExclusiveBadge={showExclusiveBadge}",
		Params: []Param{{Name: "showExclusiveBadge", Optional: true, Enum: []string{"true"}}},
	},
	{Name: "badges.detail", Method: http.MethodGet, Host: ConnectAPI, Path: "/badge-service/badge/detail/v2/{badgeID}"},
	{Name: "badges.compare", Method: http.MethodGet, Host: ConnectAPI, Path: "/badge-service/badge/compare/{displayName}"},
	{Name: "badges.leaderboard", Method: http.MethodGet, Host: ConnectAPI, Path: "/badge-service/badge/leaderboard"},
	{Name: "challenges.auto-active", Method: http.MethodGet, Host: ConnectAPI, Path: "/challenge-service/challenge/active"},
	{Name: "challenges.group-non-completed", Method: http.MethodGet, Host: ConnectAPI, Path: "/groupchallenge-service/groupchallenge/nonCompleted"},
	{Name: "challenges.adhoc-non-completed", Method: http.MethodGet, Host: ConnectAPI, Path: "/adhocchallenge-service/adHocChallenge/nonCompleted"},
	{Name: "challenges.vivokid-adhoc-non-completed", Method: http.MethodGet, Host: ConnectAPI, Path: "/vivokidchallenge-service/adHocChallenge/nonCompleted"},
	{Name: "challenges.vivokid-team-non-completed", Method: http.MethodGet, Host: ConnectAPI, Path: "/vivokidchallenge-service/teamChallenge/nonCompleted"},
	{Name: "challenges.non-completed", Method: http.MethodGet, Host: ConnectAPI, Path: "/badgechallenge-service/badgeChallenge/non-completed?desc=true&start=1&limit=10000"},
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

// CurrentChallenges sends the seven reads that the web app joins into its
// list of current challenges, and returns one object with each answer under
// the key that the web app gives it. An empty answer is {} for
// autoChallenge and [] for the lists.
func (c *Client) CurrentChallenges(ctx context.Context) (json.RawMessage, error) {
	var (
		cur struct {
			AutoChallenge              json.RawMessage `json:"autoChallenge"`
			GroupChallenges            json.RawMessage `json:"groupChallenges"`
			AdHocChallenges            json.RawMessage `json:"adHocChallenges"`
			VivokidChallenges          json.RawMessage `json:"vivokidChallenges"`
			TeamStepsVivokidChallenges json.RawMessage `json:"teamStepsVivokidChallenges"`
			BadgeChallenges            json.RawMessage `json:"badgeChallenges"`
			VirtualChallenges          json.RawMessage `json:"virtualChallenges"`
		}
		err error
	)
	if cur.AutoChallenge, err = c.Call(ctx, "challenges.auto-active", nil, nil); err != nil {
		return nil, err
	}
	if cur.GroupChallenges, err = c.Call(ctx, "challenges.group-non-completed", nil, nil); err != nil {
		return nil, err
	}
	if cur.AdHocChallenges, err = c.Call(ctx, "challenges.adhoc-non-completed", nil, nil); err != nil {
		return nil, err
	}
	if cur.VivokidChallenges, err = c.Call(ctx, "challenges.vivokid-adhoc-non-completed", nil, nil); err != nil {
		return nil, err
	}
	if cur.TeamStepsVivokidChallenges, err = c.Call(ctx, "challenges.vivokid-team-non-completed", nil, nil); err != nil {
		return nil, err
	}
	if cur.BadgeChallenges, err = c.Call(ctx, "challenges.non-completed", nil, nil); err != nil {
		return nil, err
	}
	if cur.VirtualChallenges, err = c.Call(ctx, "challenges.virtual-in-progress", nil, nil); err != nil {
		return nil, err
	}
	if len(cur.AutoChallenge) == 0 {
		cur.AutoChallenge = json.RawMessage("{}")
	}
	for _, l := range []*json.RawMessage{&cur.GroupChallenges, &cur.AdHocChallenges, &cur.VivokidChallenges, &cur.TeamStepsVivokidChallenges, &cur.BadgeChallenges, &cur.VirtualChallenges} {
		if len(*l) == 0 {
			*l = json.RawMessage("[]")
		}
	}
	return json.Marshal(cur)
}

// InProgressBadges returns the earned and available badges that are in
// progress. Garmin has no read for them, so it keeps the badges that
// python-garminconnect keeps.
func (c *Client) InProgressBadges(ctx context.Context) (json.RawMessage, error) {
	earnedData, err := c.Call(ctx, "badges.earned", nil, nil)
	if err != nil {
		return nil, err
	}
	availableData, err := c.Call(ctx, "badges.available", map[string]string{"showExclusiveBadge": "true"}, nil)
	if err != nil {
		return nil, err
	}
	var earned, available []json.RawMessage
	if len(earnedData) > 0 {
		if err := json.Unmarshal(earnedData, &earned); err != nil {
			return nil, fmt.Errorf("parse earned badges: %w", err)
		}
	}
	if len(availableData) > 0 {
		if err := json.Unmarshal(availableData, &available); err != nil {
			return nil, fmt.Errorf("parse available badges: %w", err)
		}
	}
	kept, err := inProgressBadges(earned, available)
	if err != nil {
		return nil, err
	}
	return json.Marshal(kept)
}

// inProgressBadges keeps each badge that has progress short of its target,
// or that reached its target and can be earned again. A badge in both lists
// keeps the place of its earned entry and the content of its available one.
func inProgressBadges(earned, available []json.RawMessage) ([]json.RawMessage, error) {
	kept := []json.RawMessage{}
	at := map[string]int{}
	for _, list := range [][]json.RawMessage{earned, available} {
		for _, raw := range list {
			var b struct {
				ID           json.RawMessage `json:"badgeId"`
				Progress     *float64        `json:"badgeProgressValue"`
				Target       *float64        `json:"badgeTargetValue"`
				LimitCount   *float64        `json:"badgeLimitCount"`
				EarnedNumber float64         `json:"badgeEarnedNumber"`
			}
			if err := json.Unmarshal(raw, &b); err != nil {
				return nil, fmt.Errorf("parse badge: %w", err)
			}
			switch {
			case b.Progress == nil || *b.Progress == 0:
				continue
			case b.Target != nil && *b.Progress == *b.Target && (b.LimitCount == nil || b.EarnedNumber >= *b.LimitCount):
				continue
			}
			if i, ok := at[string(b.ID)]; ok {
				kept[i] = raw
				continue
			}
			at[string(b.ID)] = len(kept)
			kept = append(kept, raw)
		}
	}
	return kept, nil
}
