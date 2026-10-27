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
		{"groups.leaderboard", map[string]string{"userType": "activity", "groupID": "1001", "start": "1", "limit": "20", "todayDate": "2026-10-21", "startDate": "2026-10-19", "endDate": "2026-10-25", "metricID": "17", "actTypeID": "1"}, "GET /groupchallenge-service/leaderboard/activity/group/1001?start=1&limit=20&todayDate=2026-10-21&startDate=2026-10-19&endDate=2026-10-25&metricId=17&actTypeId=1"},
		{"groups.leaderboard", map[string]string{"userType": "wellness", "groupID": "1001", "start": "3", "limit": "2", "todayDate": "2026-10-21", "startDate": "2026-10-19", "endDate": "2026-10-25", "metricID": "29"}, "GET /groupchallenge-service/leaderboard/wellness/group/1001?start=3&limit=2&todayDate=2026-10-21&startDate=2026-10-19&endDate=2026-10-25&metricId=29"},
		{"groups.requests", map[string]string{"displayName": "tester"}, "GET /group-service/private/group/request/user/tester"},
		{"groups.join", map[string]string{"groupID": "1001", "profileID": "3141592"}, "POST /group-service/group/1001/member/3141592"},
		{"groups.leave", map[string]string{"groupID": "1001", "profileID": "3141592"}, "DELETE /group-service/group/1001/member/3141592"},
	})
}
