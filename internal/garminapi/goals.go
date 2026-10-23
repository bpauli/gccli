package garminapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
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
	{Name: "challenges.show", Method: http.MethodGet, Host: ConnectAPI, Path: "/badgechallenge-service/badgeChallenge/{uuid}"},
	{Name: "challenges.join", Method: http.MethodPost, Host: ConnectAPI, Path: "/badgechallenge-service/badgeChallenge/{uuid}/optIn/{date}"},
	{Name: "challenges.leave", Method: http.MethodPost, Host: ConnectAPI, Path: "/badgechallenge-service/badgeChallenge/{uuid}/optOut"},
	{Name: "challenges.adhoc-historical", Method: http.MethodGet, Host: ConnectAPI, Path: "/adhocchallenge-service/adHocChallenge/historical"},
	{Name: "challenges.adhoc-show", Method: http.MethodGet, Host: ConnectAPI, Path: "/adhocchallenge-service/adHocChallenge/{uuid}?todayDate={todayDate}"},
	{Name: "challenges.adhoc-invites", Method: http.MethodGet, Host: ConnectAPI, Path: "/adhocchallenge-service/adHocChallenge/{uuid}/invite"},
	{Name: "challenges.adhoc-accept", Method: http.MethodPut, Host: ConnectAPI, Path: "/adhocchallenge-service/adHocChallenge/invite/{inviteID}/accept"},
	{Name: "challenges.adhoc-decline", Method: http.MethodPut, Host: ConnectAPI, Path: "/adhocchallenge-service/adHocChallenge/invite/{inviteID}/decline"},
	{Name: "challenges.adhoc-leave", Method: http.MethodDelete, Host: ConnectAPI, Path: "/adhocchallenge-service/adHocChallenge/{uuid}/player/{profileID}"},
	{Name: "challenges.adhoc-delete", Method: http.MethodDelete, Host: ConnectAPI, Path: "/adhocchallenge-service/adHocChallenge/{uuid}"},
	{Name: "challenges.adhoc-create", Method: http.MethodPost, Host: ConnectAPI, Path: "/adhocchallenge-service/adHocChallenge", Body: true},
	{Name: "challenges.adhoc-invite", Method: http.MethodPost, Host: ConnectAPI, Path: "/adhocchallenge-service/adHocChallenge/invite", Body: true},
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

// ChallengeType is the activity that an ad-hoc challenge counts, as
// Garmin's socialChallengeActivityTypeId.
type ChallengeType int

// The activities that the web app offers for a new ad-hoc challenge.
const (
	ChallengeWalking         ChallengeType = 0
	ChallengeRunning         ChallengeType = 1
	ChallengeCycling         ChallengeType = 2
	ChallengeSwimming        ChallengeType = 3
	ChallengeSteps           ChallengeType = 4
	ChallengePushes          ChallengeType = 10
	ChallengePushingDistance ChallengeType = 11
)

// ChallengeDuration is the length of an ad-hoc challenge, as Garmin's
// durationTypeId.
type ChallengeDuration int

const (
	ChallengeWeek    ChallengeDuration = 0
	ChallengeMonth   ChallengeDuration = 1
	ChallengeCustom  ChallengeDuration = 2
	ChallengeWeekend ChallengeDuration = 3
	ChallengeDay     ChallengeDuration = 4
)

// AdHocChallenge is a new ad-hoc challenge. It runs from the start of the
// local day Start to the end of the local day End. Invite holds the display
// names of the people to invite.
type AdHocChallenge struct {
	Name        string
	Description string
	Type        ChallengeType
	Duration    ChallengeDuration
	Start, End  time.Time
	Invite      []string
}

type adHocPlayer struct {
	UserProfileID json.Number `json:"userProfileId"`
	DisplayName   string      `json:"displayName"`
}

// CreateAdHocChallenge creates an ad-hoc challenge with the body of the web
// app. Garmin invites each player of the body. Garmin refuses an empty
// description, so the name is sent as the description when it is empty.
func (c *Client) CreateAdHocChallenge(ctx context.Context, ch AdHocChallenge) (json.RawMessage, error) {
	players := []adHocPlayer{}
	if len(ch.Invite) > 0 {
		var err error
		if players, err = c.mutualFollowers(ctx, ch.Invite); err != nil {
			return nil, err
		}
	}
	owner, err := c.GetProfilePK(ctx)
	if err != nil {
		return nil, err
	}
	desc := ch.Description
	if desc == "" {
		desc = ch.Name
	}
	body, err := json.Marshal(struct {
		Name     string            `json:"adHocChallengeName"`
		Desc     string            `json:"adHocChallengeDesc"`
		Type     ChallengeType     `json:"socialChallengeActivityTypeId"`
		Duration ChallengeDuration `json:"durationTypeId"`
		Start    string            `json:"startDate"`
		End      string            `json:"endDate"`
		Ranking  int               `json:"userRanking"`
		Status   int               `json:"socialChallengeStatusId"`
		Owner    json.Number       `json:"ownerUserProfileId"`
		Players  []adHocPlayer     `json:"players"`
	}{
		Name:     ch.Name,
		Desc:     desc,
		Type:     ch.Type,
		Duration: ch.Duration,
		Start:    ch.Start.Format("2006-01-02") + "T00:00:00.00",
		End:      ch.End.Format("2006-01-02") + "T23:59:59.999",
		Owner:    json.Number(owner),
		Players:  players,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal ad-hoc challenge: %w", err)
	}
	return c.Call(ctx, "challenges.adhoc-create", nil, body)
}

// InviteToAdHocChallenge invites the people with the display names to the
// ad-hoc challenge uuid.
func (c *Client) InviteToAdHocChallenge(ctx context.Context, uuid string, displayNames []string) (json.RawMessage, error) {
	players, err := c.mutualFollowers(ctx, displayNames)
	if err != nil {
		return nil, err
	}
	owner, err := c.GetProfilePK(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]json.Number, len(players))
	for i, p := range players {
		ids[i] = p.UserProfileID
	}
	body, err := json.Marshal(struct {
		UUID     string        `json:"adHocChallengeUuid"`
		Invitor  json.Number   `json:"invitorId"`
		Invitees []json.Number `json:"inviteeIds"`
	}{uuid, json.Number(owner), ids})
	if err != nil {
		return nil, fmt.Errorf("marshal ad-hoc challenge invite: %w", err)
	}
	return c.Call(ctx, "challenges.adhoc-invite", nil, body)
}

// mutualFollowers returns the player of each display name, in the order
// given. Garmin invites only people who follow you and whom you follow, so
// a name that is not a mutual follower fails before any write.
func (c *Client) mutualFollowers(ctx context.Context, displayNames []string) ([]adHocPlayer, error) {
	data, err := c.Call(ctx, "followers.mutuals", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("get mutual followers: %w", err)
	}
	var mutuals []struct {
		UserID      json.Number `json:"userId"`
		DisplayName string      `json:"displayName"`
	}
	if len(bytes.TrimSpace(data)) > 0 {
		if err := json.Unmarshal(data, &mutuals); err != nil {
			return nil, fmt.Errorf("parse mutual followers: %w", err)
		}
	}
	byName := map[string]json.Number{}
	for _, m := range mutuals {
		byName[m.DisplayName] = m.UserID
	}
	players := make([]adHocPlayer, 0, len(displayNames))
	var missing []string
	for _, name := range displayNames {
		id, ok := byName[name]
		if !ok {
			missing = append(missing, strconv.Quote(name))
			continue
		}
		players = append(players, adHocPlayer{UserProfileID: id, DisplayName: name})
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("cannot invite %s: Garmin invites only people you follow who also follow you", strings.Join(missing, ", "))
	}
	return players, nil
}
