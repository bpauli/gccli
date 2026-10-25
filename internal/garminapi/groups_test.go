package garminapi

import "testing"

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
		{"groups.leave", map[string]string{"groupID": "1001", "profileID": "3141592"}, "DELETE /group-service/group/1001/member/3141592"},	})
}
