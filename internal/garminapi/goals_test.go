package garminapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"net/http"
	"slices"
	"testing"
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
