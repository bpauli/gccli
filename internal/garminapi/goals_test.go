package garminapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestGoalsEndpoints(t *testing.T) {
	testEndpoints(t, goalsEndpoints, []endpointCase{
		{"goals.list", map[string]string{"status": "past"}, "GET /goal-service/goal/goals?status=past"},
		{"goals.effective-weight-goal", map[string]string{"startDate": "2026-01-01", "endDate": "2026-06-30"}, "GET /goal-service/goal/user/effective/weightgoal/2026-01-01/2026-06-30"},
		{"goals.set-effective", map[string]string{"goalType": "intensity-minutes", "date": "2026-10-19"}, "PUT /goal-service/goal/user/effective/intensity-minutes/2026-10-19"},
		{"goals.steps", map[string]string{"date": "2026-10-19"}, "GET /wellness-service/wellness/wellness-goals/consolidated/steps/2026-10-19"},
		{"goals.set-steps", map[string]string{"date": "2026-10-19"}, "PUT /wellness-service/wellness/wellness-goals/consolidated/steps/2026-10-19"},
		{"badges.earned", nil, "GET /badge-service/badge/earned"},
		{"badges.available", nil, "GET /badge-service/badge/available"},
		{"badges.available", map[string]string{"showExclusiveBadge": "true"}, "GET /badge-service/badge/available?showExclusiveBadge=true"},
		{"badges.detail", map[string]string{"badgeID": "42"}, "GET /badge-service/badge/detail/v2/42"},
		{"badges.compare", map[string]string{"displayName": "jo/b"}, "GET /badge-service/badge/compare/jo%2Fb"},
		{"badges.leaderboard", nil, "GET /badge-service/badge/leaderboard"},
		{"challenges.auto-active", nil, "GET /challenge-service/challenge/active"},
		{"challenges.group-non-completed", nil, "GET /groupchallenge-service/groupchallenge/nonCompleted"},
		{"challenges.adhoc-non-completed", nil, "GET /adhocchallenge-service/adHocChallenge/nonCompleted"},
		{"challenges.vivokid-adhoc-non-completed", nil, "GET /vivokidchallenge-service/adHocChallenge/nonCompleted"},
		{"challenges.vivokid-team-non-completed", nil, "GET /vivokidchallenge-service/teamChallenge/nonCompleted"},
		{"challenges.non-completed", nil, "GET /badgechallenge-service/badgeChallenge/non-completed?desc=true&start=1&limit=10000"},
		{"challenges.available", nil, "GET /badgechallenge-service/badgeChallenge/available"},
		{"challenges.completed", nil, "GET /badgechallenge-service/badgeChallenge/completed"},
		{"challenges.virtual-available", nil, "GET /badgechallenge-service/virtualChallenge/available"},
		{"challenges.virtual-completed", nil, "GET /badgechallenge-service/virtualChallenge/completed"},
		{"challenges.virtual-in-progress", nil, "GET /badgechallenge-service/virtualChallenge/inProgress"},
		{"challenges.show", map[string]string{"uuid": "0123456789ABCDEF0123456789ABCDEF"}, "GET /badgechallenge-service/badgeChallenge/0123456789ABCDEF0123456789ABCDEF"},
		{"challenges.join", map[string]string{"uuid": "0123456789ABCDEF0123456789ABCDEF", "date": "2026-10-22"}, "POST /badgechallenge-service/badgeChallenge/0123456789ABCDEF0123456789ABCDEF/optIn/2026-10-22"},
		{"challenges.leave", map[string]string{"uuid": "0123456789ABCDEF0123456789ABCDEF"}, "POST /badgechallenge-service/badgeChallenge/0123456789ABCDEF0123456789ABCDEF/optOut"},
		{"challenges.adhoc-historical", nil, "GET /adhocchallenge-service/adHocChallenge/historical"},
		{"challenges.adhoc-show", map[string]string{"uuid": "0123456789ABCDEF0123456789ABCDEF", "todayDate": "2026-10-22"}, "GET /adhocchallenge-service/adHocChallenge/0123456789ABCDEF0123456789ABCDEF?todayDate=2026-10-22"},
		{"challenges.adhoc-invites", map[string]string{"uuid": "0123456789ABCDEF0123456789ABCDEF"}, "GET /adhocchallenge-service/adHocChallenge/0123456789ABCDEF0123456789ABCDEF/invite"},
		{"challenges.adhoc-accept", map[string]string{"inviteID": "1001"}, "PUT /adhocchallenge-service/adHocChallenge/invite/1001/accept"},
		{"challenges.adhoc-decline", map[string]string{"inviteID": "1001"}, "PUT /adhocchallenge-service/adHocChallenge/invite/1001/decline"},
		{"challenges.adhoc-leave", map[string]string{"uuid": "0123456789ABCDEF0123456789ABCDEF", "profileID": "1002"}, "DELETE /adhocchallenge-service/adHocChallenge/0123456789ABCDEF0123456789ABCDEF/player/1002"},
		{"challenges.adhoc-delete", map[string]string{"uuid": "0123456789ABCDEF0123456789ABCDEF"}, "DELETE /adhocchallenge-service/adHocChallenge/0123456789ABCDEF0123456789ABCDEF"},
		{"challenges.adhoc-create", nil, "POST /adhocchallenge-service/adHocChallenge"},
		{"challenges.adhoc-invite", nil, "POST /adhocchallenge-service/adHocChallenge/invite"},
		{"records.list", map[string]string{"displayName": "jo/b"}, "GET /personalrecord-service/personalrecord/prs/jo%2Fb"},
		{"records.types", map[string]string{"displayName": "jo/b"}, "GET /personalrecord-service/personalrecordtype/prtypes/jo%2Fb"},
		{"records.activity", map[string]string{"activityID": "42"}, "GET /personalrecord-service/personalrecord/prByActivityId/42"},
	})
}

func TestGoalsListRefusesUnknownStatus(t *testing.T) {
	_, client := testServer(t, http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("request sent: %s %s", r.Method, r.URL.RequestURI())
	}))
	_, err := client.Call(context.Background(), "goals.list", map[string]string{"status": "bogus"}, nil)
	var epErr *EndpointError
	if !errors.As(err, &epErr) || epErr.Param != "status" {
		t.Errorf("err = %v, want EndpointError for status", err)
	}
}

func TestGoalsSetEffectiveRefusesUnknownType(t *testing.T) {
	_, client := testServer(t, http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("request sent: %s %s", r.Method, r.URL.RequestURI())
	}))
	_, err := client.Call(context.Background(), "goals.set-effective", map[string]string{"goalType": "steps", "date": "2026-10-19"}, []byte(`{}`))
	var epErr *EndpointError
	if !errors.As(err, &epErr) || epErr.Param != "goalType" {
		t.Errorf("err = %v, want EndpointError for goalType", err)
	}
}

func TestSetStepGoal(t *testing.T) {
	const saved = `{"userGoal":0,"deviceGoal":7500,"userGoalCategory":"my-auto","effective":true,"syncedToDevice":true}`
	tests := []struct {
		name string
		goal StepGoal
		want map[string]any
	}{
		{"manual", ManualStepGoal(9000), map[string]any{"userGoal": 9000.0, "userGoalCategory": "manual"}},
		{"auto", AutoStepGoal, map[string]any{"userGoal": 0.0, "userGoalCategory": "my-auto"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var requests []string
			var put map[string]any
			_, client := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests = append(requests, r.Method+" "+r.URL.EscapedPath())
				if r.Method == http.MethodGet {
					_, _ = w.Write([]byte(saved))
					return
				}
				b, _ := io.ReadAll(r.Body)
				if err := json.Unmarshal(b, &put); err != nil {
					t.Errorf("PUT body %q: %v", b, err)
				}
				_, _ = w.Write([]byte(`{"userGoal":1}`))
			}))
			data, err := client.SetStepGoal(context.Background(), "2026-10-19", tc.goal)
			if err != nil {
				t.Fatalf("SetStepGoal: %v", err)
			}
			if string(data) != `{"userGoal":1}` {
				t.Errorf("data = %s, want the PUT response", data)
			}
			const path = "/wellness-service/wellness/wellness-goals/consolidated/steps/2026-10-19"
			if want := []string{"GET " + path, "PUT " + path}; !slices.Equal(requests, want) {
				t.Errorf("requests = %q, want %q", requests, want)
			}
			var want map[string]any
			_ = json.Unmarshal([]byte(saved), &want)
			maps.Copy(want, tc.want)
			if !maps.Equal(put, want) {
				t.Errorf("PUT body = %v, want the saved object with %v", put, tc.want)
			}
		})
	}
}

func TestSetStepGoal_GetError(t *testing.T) {
	var puts int
	_, client := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			puts++
		}
		w.WriteHeader(http.StatusBadRequest)
	}))
	_, err := client.SetStepGoal(context.Background(), "2026-10-19", AutoStepGoal)
	var apiErr *GarminAPIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("err = %v, want a GarminAPIError with status 400", err)
	}
	if puts != 0 {
		t.Errorf("%d writes sent after the GET failed, want 0", puts)
	}
}

func TestInProgressBadges(t *testing.T) {
	tests := []struct {
		name              string
		earned, available []string
		want              []string
	}{
		{"progress short of the target", nil, []string{`{"badgeId":1,"badgeProgressValue":5,"badgeTargetValue":10}`}, []string{`{"badgeId":1,"badgeProgressValue":5,"badgeTargetValue":10}`}},
		{"fraction of the target", nil, []string{`{"badgeId":1,"badgeProgressValue":0.5,"badgeTargetValue":1}`}, []string{`{"badgeId":1,"badgeProgressValue":0.5,"badgeTargetValue":1}`}},
		{"progress without a target", nil, []string{`{"badgeId":1,"badgeProgressValue":5}`}, []string{`{"badgeId":1,"badgeProgressValue":5}`}},
		{"progress past the target", nil, []string{`{"badgeId":1,"badgeProgressValue":12,"badgeTargetValue":10}`}, []string{`{"badgeId":1,"badgeProgressValue":12,"badgeTargetValue":10}`}},
		{"no progress", nil, []string{`{"badgeId":1,"badgeTargetValue":10}`}, nil},
		{"null progress", nil, []string{`{"badgeId":1,"badgeProgressValue":null,"badgeTargetValue":10}`}, nil},
		{"zero progress", nil, []string{`{"badgeId":1,"badgeProgressValue":0,"badgeTargetValue":10}`}, nil},
		{"target reached without a limit", []string{`{"badgeId":1,"badgeProgressValue":10,"badgeTargetValue":10}`}, nil, nil},
		{"target reached with a null limit", []string{`{"badgeId":1,"badgeProgressValue":10,"badgeTargetValue":10,"badgeLimitCount":null,"badgeEarnedNumber":1}`}, nil, nil},
		{"target reached, can be earned again", []string{`{"badgeId":1,"badgeProgressValue":1,"badgeTargetValue":1,"badgeLimitCount":5,"badgeEarnedNumber":2}`}, nil, []string{`{"badgeId":1,"badgeProgressValue":1,"badgeTargetValue":1,"badgeLimitCount":5,"badgeEarnedNumber":2}`}},
		{"target reached, earned to the limit", []string{`{"badgeId":1,"badgeProgressValue":1,"badgeTargetValue":1,"badgeLimitCount":5,"badgeEarnedNumber":5}`}, nil, nil},
		{"target reached with a limit, never earned", nil, []string{`{"badgeId":1,"badgeProgressValue":1,"badgeTargetValue":1,"badgeLimitCount":1}`}, []string{`{"badgeId":1,"badgeProgressValue":1,"badgeTargetValue":1,"badgeLimitCount":1}`}},
		{
			"a badge in both lists keeps the earned place and the available entry",
			[]string{`{"badgeId":1,"badgeProgressValue":5,"badgeTargetValue":10,"from":"earned"}`, `{"badgeId":2,"badgeProgressValue":3,"badgeTargetValue":10}`},
			[]string{`{"badgeId":3,"badgeProgressValue":1,"badgeTargetValue":10}`, `{"badgeId":1,"badgeProgressValue":6,"badgeTargetValue":10,"from":"available"}`},
			[]string{`{"badgeId":1,"badgeProgressValue":6,"badgeTargetValue":10,"from":"available"}`, `{"badgeId":2,"badgeProgressValue":3,"badgeTargetValue":10}`, `{"badgeId":3,"badgeProgressValue":1,"badgeTargetValue":10}`},
		},
		{
			"an available badge that is not in progress leaves the earned one",
			[]string{`{"badgeId":1,"badgeProgressValue":5,"badgeTargetValue":10}`},
			[]string{`{"badgeId":1,"badgeProgressValue":0,"badgeTargetValue":10}`},
			[]string{`{"badgeId":1,"badgeProgressValue":5,"badgeTargetValue":10}`},
		},
	}
	raws := func(list []string) []json.RawMessage {
		var out []json.RawMessage
		for _, s := range list {
			out = append(out, json.RawMessage(s))
		}
		return out
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := inProgressBadges(raws(tc.earned), raws(tc.available))
			if err != nil {
				t.Fatal(err)
			}
			var gotS []string
			for _, b := range got {
				gotS = append(gotS, string(b))
			}
			if !slices.Equal(gotS, tc.want) {
				t.Errorf("kept %q, want %q", gotS, tc.want)
			}
		})
	}
}

// adHocServer answers the reads that the ad-hoc challenge writes send, and
// records each request with its body.
func adHocServer(t *testing.T, writeAnswer string) (*[]string, *Client) {
	t.Helper()
	var requests []string
	_, client := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		requests = append(requests, strings.TrimSpace(r.Method+" "+r.URL.Path+" "+string(b)))
		switch r.URL.Path {
		case "/follower-service/follow/mutuals":
			_, _ = w.Write([]byte(`[{"displayName":"ann","userId":2001,"fullName":"Ann"},{"displayName":"bob","userId":2002}]`))
		case "/userprofile-service/userprofile/settings", "/userprofile-service/socialProfile/tester":
			_, _ = w.Write([]byte(`{"displayName":"tester","profileId":3141592}`))
		default:
			if writeAnswer == "" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			_, _ = w.Write([]byte(writeAnswer))
		}
	}))
	return &requests, client
}

var profileReads = []string{"GET /userprofile-service/userprofile/settings", "GET /userprofile-service/socialProfile/tester"}

func TestCreateAdHocChallenge(t *testing.T) {
	day := func(d int) time.Time { return time.Date(2026, time.October, d, 0, 0, 0, 0, time.Local) }
	tests := []struct {
		name string
		ch   AdHocChallenge
		want []string
	}{
		{
			"invites and description",
			AdHocChallenge{Name: "Step up", Description: "Who walks most", Type: ChallengeSteps, Duration: ChallengeWeek, Start: day(15), End: day(21), Invite: []string{"bob", "ann"}},
			slices.Concat([]string{"GET /follower-service/follow/mutuals"}, profileReads, []string{
				`POST /adhocchallenge-service/adHocChallenge {"adHocChallengeName":"Step up","adHocChallengeDesc":"Who walks most","socialChallengeActivityTypeId":4,"durationTypeId":0,"startDate":"2026-10-15T00:00:00.00","endDate":"2026-10-21T23:59:59.999","userRanking":0,"socialChallengeStatusId":0,"ownerUserProfileId":3141592,"players":[{"userProfileId":2002,"displayName":"bob"},{"userProfileId":2001,"displayName":"ann"}]}`,
			}),
		},
		{
			"no invite skips the mutuals and the name is the description",
			AdHocChallenge{Name: "Ride", Type: ChallengePushingDistance, Duration: ChallengeDay, Start: day(15), End: day(15)},
			slices.Concat(profileReads, []string{
				`POST /adhocchallenge-service/adHocChallenge {"adHocChallengeName":"Ride","adHocChallengeDesc":"Ride","socialChallengeActivityTypeId":11,"durationTypeId":4,"startDate":"2026-10-15T00:00:00.00","endDate":"2026-10-15T23:59:59.999","userRanking":0,"socialChallengeStatusId":0,"ownerUserProfileId":3141592,"players":[]}`,
			}),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			requests, client := adHocServer(t, `{"uuid":"ABC"}`)
			data, err := client.CreateAdHocChallenge(context.Background(), tc.ch)
			if err != nil {
				t.Fatalf("CreateAdHocChallenge: %v", err)
			}
			if string(data) != `{"uuid":"ABC"}` {
				t.Errorf("data = %s, want the POST answer", data)
			}
			if !slices.Equal(*requests, tc.want) {
				t.Errorf("requests:\n%s\nwant:\n%s", strings.Join(*requests, "\n"), strings.Join(tc.want, "\n"))
			}
		})
	}
}

func TestInviteToAdHocChallenge(t *testing.T) {
	requests, client := adHocServer(t, "")
	if _, err := client.InviteToAdHocChallenge(context.Background(), "ABC", []string{"ann", "bob"}); err != nil {
		t.Fatalf("InviteToAdHocChallenge: %v", err)
	}
	want := slices.Concat([]string{"GET /follower-service/follow/mutuals"}, profileReads, []string{
		`POST /adhocchallenge-service/adHocChallenge/invite {"adHocChallengeUuid":"ABC","invitorId":3141592,"inviteeIds":[2001,2002]}`,
	})
	if !slices.Equal(*requests, want) {
		t.Errorf("requests:\n%s\nwant:\n%s", strings.Join(*requests, "\n"), strings.Join(want, "\n"))
	}
}

// TestAdHocInviteNeedsMutualFollowers pins that a display name that is not
// a mutual follower fails after the mutuals read and before any other
// request, and that the error names each such person. Names match exactly.
func TestAdHocInviteNeedsMutualFollowers(t *testing.T) {
	const want = `cannot invite "carol", "Ann": Garmin invites only people you follow who also follow you`
	writes := map[string]func(*Client) error{
		"create": func(c *Client) error {
			_, err := c.CreateAdHocChallenge(context.Background(), AdHocChallenge{Name: "x", Invite: []string{"ann", "carol", "Ann"}})
			return err
		},
		"invite": func(c *Client) error {
			_, err := c.InviteToAdHocChallenge(context.Background(), "ABC", []string{"ann", "carol", "Ann"})
			return err
		},
	}
	for name, write := range writes {
		t.Run(name, func(t *testing.T) {
			requests, client := adHocServer(t, `{}`)
			if err := write(client); err == nil || err.Error() != want {
				t.Errorf("err = %v, want %q", err, want)
			}
			if want := []string{"GET /follower-service/follow/mutuals"}; !slices.Equal(*requests, want) {
				t.Errorf("requests = %q, want %q", *requests, want)
			}
		})
	}
}
