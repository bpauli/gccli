package garminapi

import (
	"context"
	"io"
	"net/http"
	"testing"
)

func TestGroupsEndpoints(t *testing.T) {
	testEndpoints(t, groupsEndpoints, []endpointCase{
		{"groups.list", map[string]string{"displayName": "tester"}, "GET /group-service/groups/tester?includeMembership=true"},
		{"groups.list", map[string]string{"displayName": "a b/c"}, "GET /group-service/groups/a%20b%2Fc?includeMembership=true"},
		{"groups.search", nil, "POST /group-service/keyword"},
		{"groups.get", map[string]string{"groupID": "1001"}, "GET /group-service/group/1001?includeMembership=true"},
		{"groups.members", map[string]string{"groupID": "1001"}, "GET /group-service/group/1001/members?supportOwner=true&orderByRelationship=true"},
		{"groups.feed", map[string]string{"groupID": "1001"}, "GET /web-gateway/group/v2/1001/groupFeed"},
		{"groups.activities", map[string]string{"groupID": "1001"}, "GET /web-gateway/group/v2/1001/activityFeed"},
		{"groups.challenges", map[string]string{"groupID": "1001"}, "GET /groupchallenge-service/group/1001/groupchallenge/nonCompleted"},
		{"groups.join", map[string]string{"groupID": "1001", "profileID": "3141592"}, "POST /group-service/group/1001/member/3141592"},
		{"groups.leave", map[string]string{"groupID": "1001", "profileID": "3141592"}, "DELETE /group-service/group/1001/member/3141592"},
	})
}

func TestGroupsSearchIsAForm(t *testing.T) {
	var contentType, body string
	_, client := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		_, _ = w.Write([]byte(`{}`))
	}))
	if _, err := client.Call(context.Background(), "groups.search", nil, []byte("keyword=running&limit=20&start=1")); err != nil {
		t.Fatal(err)
	}
	if contentType != "application/x-www-form-urlencoded" {
		t.Errorf("Content-Type = %q, want application/x-www-form-urlencoded", contentType)
	}
	if body != "keyword=running&limit=20&start=1" {
		t.Errorf("body = %q, want the form as given", body)
	}
}
