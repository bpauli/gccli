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
		{"gear.types", nil, "GET /gear-service/gear/types"},
		{"gear.collections", nil, "GET /gear-service/collection"},
		{"gear.get", map[string]string{"gearUUID": "abc-123"}, "GET /gear-service/gear/v2/abc-123"},
		{"gear.create", nil, "POST /gear-service/gear/v2"},
		{"gear.update", map[string]string{"gearUUID": "abc-123"}, "PUT /gear-service/gear/v2/abc-123"},
		{"gear.retire", map[string]string{"gearUUID": "abc-123"}, "PUT /gear-service/gear/v2/abc-123/status/RETIRED"},
		{"gear.unretire", map[string]string{"gearUUID": "abc-123"}, "PUT /gear-service/gear/v2/abc-123/status/ACTIVE"},
		{"gear.delete", map[string]string{"gearUUID": "abc-123"}, "DELETE /gear-service/gear/v2/abc-123"},
		{"gear.collection.get", map[string]string{"collectionUUID": "c-1"}, "GET /gear-service/collection/c-1"},
		{"gear.collection.create", nil, "POST /gear-service/collection"},
		{"gear.collection.update", map[string]string{"collectionUUID": "c-1"}, "PUT /gear-service/collection/c-1"},
		{"gear.collection.add-gear", map[string]string{"collectionUUID": "c-1", "gearUUID": "abc-123"}, "POST /gear-service/collection/c-1/gear/abc-123"},
		{"gear.collection.remove-gear", map[string]string{"collectionUUID": "c1", "gearUUID": "g1"}, "DELETE /gear-service/collection/c1/gear/g1"},
		{"gear.models", map[string]string{"keywords": "brooks ghost"}, "GET /gear-service/gear/makemodels?keywords=brooks+ghost"},
		{"gear.collection.delete", map[string]string{"collectionUUID": "c-1"}, "DELETE /gear-service/collection/c-1"},
	})
}
