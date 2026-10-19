package garminapi

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestGoalsEndpoints(t *testing.T) {
	testEndpoints(t, goalsEndpoints, []endpointCase{
		{"goals.list", map[string]string{"status": "past"}, "GET /goal-service/goal/goals?status=past"},
		{"goals.effective-weight-goal", map[string]string{"startDate": "2026-01-01", "endDate": "2026-06-30"}, "GET /goal-service/goal/user/effective/weightgoal/2026-01-01/2026-06-30"},
		{"badges.earned", nil, "GET /badge-service/badge/earned"},
		{"badges.available", nil, "GET /badge-service/badge/available"},
		{"badges.in-progress", nil, "GET /badge-service/badge/in-progress"},
		{"badges.detail", map[string]string{"badgeID": "42"}, "GET /badge-service/badge/detail/v2/42"},
		{"badges.challenges", nil, "GET /badge-service/badge/challenges"},
		{"challenges.joined", nil, "GET /challenge-service/challenge/joined"},
		{"challenges.available", nil, "GET /badgechallenge-service/badgeChallenge/available"},
		{"challenges.completed", nil, "GET /badgechallenge-service/badgeChallenge/completed"},
		{"records.list", map[string]string{"displayName": "jo/b"}, "GET /personalrecord-service/personalrecord/prs/jo%2Fb"},
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
