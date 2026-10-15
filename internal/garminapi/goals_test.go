package garminapi

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestGoalsEndpoints(t *testing.T) {
	tests := []struct {
		name string
		args map[string]string
		want string
	}{
		{"goals.list", map[string]string{"status": "past"}, "GET /goal-service/goal/goals?status=past"},
		{"badges.earned", nil, "GET /badge-service/badge/earned"},
		{"badges.available", nil, "GET /badge-service/badge/available"},
		{"badges.in-progress", nil, "GET /badge-service/badge/in-progress"},
		{"badges.challenges", nil, "GET /badge-service/badge/challenges"},
		{"challenges.joined", nil, "GET /challenge-service/challenge/joined"},
		{"records.list", map[string]string{"displayName": "jo/b"}, "GET /personalrecord-service/personalrecord/prs/jo%2Fb"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got string
			_, client := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got = r.Method + " " + r.URL.RequestURI()
				_, _ = w.Write([]byte(`[{"id":1}]`))
			}))
			data, err := client.Call(context.Background(), tc.name, tc.args, nil)
			if err != nil || got != tc.want || string(data) != `[{"id":1}]` {
				t.Errorf("Call = %q, %v; sent %q, want %q", data, err, got, tc.want)
			}
		})
	}
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
