package garminapi

import "testing"

func TestSegmentsEndpoints(t *testing.T) {
	uuid := map[string]string{"uuid": "a1b2c3d4-0000-4000-8000-000000000001"}
	testEndpoints(t, segmentsEndpoints, []endpointCase{
		{"segments.search", nil, "POST /segment-service/segment/filterSegments"},
		{"segments.get", uuid, "GET /segment-service/segment/a1b2c3d4-0000-4000-8000-000000000001"},
		{"segments.leaderboard", nil, "POST /segment-service/leaderboard/search/"},
		{
			"segments.rank",
			map[string]string{"uuid": "a1b2c3d4-0000-4000-8000-000000000001", "displayName": "runner one"},
			"GET /segment-service/leaderboard/segment/a1b2c3d4-0000-4000-8000-000000000001/rank/runner%20one",
		},
		{"segments.classifications", nil, "GET /segment-service/segment/classifications"},
		{"segments.surface-types", nil, "GET /segment-service/segment/surfaceTypes"},
		{"segments.activity", map[string]string{"activityID": "123456"}, "GET /segment-service/segment/list/123456"},
		{"segments.track", map[string]string{"activityID": "123456"}, "GET /segment-service/segment/activity/123456"},
		{"segments.create", nil, "POST /segment-service/segment/"},
		{"segments.delete", uuid, "DELETE /segment-service/segment/a1b2c3d4-0000-4000-8000-000000000001"},
		{"segments.favorite", map[string]string{"segmentPK": "900001"}, "PUT /segment-service/segment/user/900001"},
		{"segments.unfavorite", map[string]string{"segmentPK": "900001"}, "DELETE /segment-service/segment/user/900001"},
	})
}
