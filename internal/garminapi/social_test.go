package garminapi

import (
	"context"
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
		{"users.search", nil, "POST /usersearch-service/search/v3?displayMutedStatus=true"},
	})
}

// Garmin answers 405 to a GET search and 415 to a JSON body.
func TestUserSearchIsAForm(t *testing.T) {
	var contentType, body string
	_, client := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		_, _ = w.Write([]byte(`{}`))
	}))
	if _, err := client.Call(context.Background(), "users.search", nil, []byte("keyword=anna&limit=5&start=21")); err != nil {
		t.Fatal(err)
	}
	if contentType != "application/x-www-form-urlencoded" {
		t.Errorf("Content-Type = %q, want application/x-www-form-urlencoded", contentType)
	}
	if body != "keyword=anna&limit=5&start=21" {
		t.Errorf("body = %q, want the form as given", body)
	}
}
