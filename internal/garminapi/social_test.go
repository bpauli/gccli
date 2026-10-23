package garminapi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
)

func TestSocialEndpoints(t *testing.T) {
	dn := map[string]string{"displayName": "tester"}
	odd := map[string]string{"displayName": "a b/c"}
	testEndpoints(t, socialEndpoints, []endpointCase{
		{"connections.list", map[string]string{"start": "40", "limit": "20"}, "GET /connection-service/connection/v2/connections/pagination?start=40&limit=20"},
		{"followers.list", dn, "GET /follower-service/follow/followers/tester"},
		{"followers.list", odd, "GET /follower-service/follow/followers/a%20b%2Fc"},
		{"followers.remove", dn, "DELETE /follower-service/follow/followers/tester"},
		{"following.list", dn, "GET /follower-service/follow/followings/tester"},
		{"following.list", odd, "GET /follower-service/follow/followings/a%20b%2Fc"},
		{"following.add", dn, "POST /follower-service/follow/followings/tester"},
		{"following.remove", odd, "DELETE /follower-service/follow/followings/a%20b%2Fc"},
		{"follow.status", dn, "GET /follower-service/follow/tester/profileStatus"},
		{"follow-requests.list", nil, "GET /follower-service/follow/pending"},
		{"follow-requests.inbox", map[string]string{"cursor": ""}, "GET /web-gateway/inbox/v2/requests?req=8"},
		{"follow-requests.inbox", map[string]string{"cursor": "c 2"}, "GET /web-gateway/inbox/v2/requests?req=8&cursor=c+2"},
		{"follow-requests.accept", map[string]string{"requestID": "4242"}, "PUT /follower-service/follow/requests/4242/accept"},
		{"follow-requests.decline", map[string]string{"requestID": "4242"}, "PUT /follower-service/follow/requests/4242/decline"},
		{"muted.list", nil, "GET /follower-service/muteuser"},
		{"muted.add", dn, "POST /follower-service/muteuser/tester"},
		{"muted.remove", odd, "DELETE /follower-service/muteuser/a%20b%2Fc"},
		{"followers.mutuals", nil, "GET /follower-service/follow/mutuals"},
		{"connections.pending", nil, "GET /connection-service/connection/pending"},
		{"newsfeed.list", nil, "GET /web-gateway/newsfeed/v3"},
	})
}

func TestSearchUsers(t *testing.T) {
	var method, uri, contentType, body string
	_, client := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, uri, contentType = r.Method, r.URL.RequestURI(), r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		_, _ = w.Write([]byte(`{"profileList":[]}`))
	}))
	data, err := client.SearchUsers(context.Background(), "anna b&c", 20, 5)
	if err != nil {
		t.Fatalf("SearchUsers: %v", err)
	}
	if string(data) != `{"profileList":[]}` {
		t.Errorf("data = %s, want the response body", data)
	}
	if method != http.MethodPost || uri != "/usersearch-service/search/v3?displayMutedStatus=true" {
		t.Errorf("request = %s %s, want POST /usersearch-service/search/v3?displayMutedStatus=true: a GET answers 405", method, uri)
	}
	if contentType != "application/x-www-form-urlencoded; charset=UTF-8" {
		t.Errorf("Content-Type = %q, want a form: Garmin answers JSON with 415", contentType)
	}
	if want := "keyword=anna+b%26c&limit=5&start=21"; body != want {
		t.Errorf("body = %q, want %q: start counts from 1", body, want)
	}
}

func TestSearchUsers_ServerError(t *testing.T) {
	_, client := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	_, err := client.SearchUsers(context.Background(), "a", 0, 20)
	var apiErr *GarminAPIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("err = %v, want a GarminAPIError with status 400", err)
	}
}
