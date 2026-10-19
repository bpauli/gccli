package garminapi

import "testing"

func TestGroupsEndpoints(t *testing.T) {
	testEndpoints(t, groupsEndpoints, []endpointCase{
		{"groups.list", map[string]string{"displayName": "tester"}, "GET /group-service/groups/tester?includeMembership=true"},
		{"groups.list", map[string]string{"displayName": "a b/c"}, "GET /group-service/groups/a%20b%2Fc?includeMembership=true"},
	})
}
