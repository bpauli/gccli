package garminapi

import "testing"

func TestSocialEndpoints(t *testing.T) {
	testEndpoints(t, socialEndpoints, []endpointCase{
		{"connections.list", map[string]string{"start": "40", "limit": "20"}, "GET /connection-service/connection/v2/connections/pagination?start=40&limit=20"},
		{"followers.list", map[string]string{"displayName": "tester"}, "GET /follower-service/follow/followers/tester"},
		{"followers.list", map[string]string{"displayName": "a b/c"}, "GET /follower-service/follow/followers/a%20b%2Fc"},
		{"following.list", map[string]string{"displayName": "tester"}, "GET /follower-service/follow/followings/tester"},
		{"following.list", map[string]string{"displayName": "a b/c"}, "GET /follower-service/follow/followings/a%20b%2Fc"},
		{"followers.mutuals", nil, "GET /follower-service/follow/mutuals"},
		{"connections.pending", nil, "GET /connection-service/connection/pending"},
		{"newsfeed.list", nil, "GET /web-gateway/newsfeed/v3"},
	})
}
