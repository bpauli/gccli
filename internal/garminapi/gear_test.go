package garminapi

import (
	"testing"
)

func TestGearEndpoints(t *testing.T) {
	link := map[string]string{"gearUUID": "abc-123", "activityID": "456"}
	testEndpoints(t, gearEndpoints, []endpointCase{
		{"gear.list", map[string]string{"userProfilePK": "12345"}, "GET /gear-service/gear/filterGear?userProfilePk=12345"},
		{"gear.stats", map[string]string{"gearUUID": "abc-123"}, "GET /gear-service/gear/stats/abc-123"},
		{"gear.activities", map[string]string{"gearUUID": "abc-123", "limit": "20"}, "GET /activitylist-service/activities/abc-123/gear?limit=20"},
		{"gear.defaults", map[string]string{"userProfileNumber": "12345"}, "GET /gear-service/gear/user/12345/activityTypes"},
		{"gear.link", link, "PUT /gear-service/gear/link/abc-123/activity/456"},
		{"gear.unlink", link, "PUT /gear-service/gear/unlink/abc-123/activity/456"},
	})
}
