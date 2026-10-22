package garminapi

import "testing"

func TestGolfEndpoints(t *testing.T) {
	page := map[string]string{"perPage": "20", "page": "1"}
	testEndpoints(t, golfEndpoints, []endpointCase{
		{"golf.scorecards", page, "GET /gcs-golfcommunity/api/v2/scorecard/summary?per-page=20&page=1&user-locale=en"},
		{"golf.courses", map[string]string{"perPage": "5", "page": "2"}, "GET /gcs-golfcommunity/api/v2/course-snapshot/player?per-page=5&page=2&user-locale=en"},
		{"golf.stats", nil, "GET /gcs-golfcommunity/api/v2/player/stats"},
		{"golf.shot-stats", map[string]string{"kind": "putt"}, "GET /gcs-golfcommunity/api/v2/shot/stats/putt"},
		{"golf.clubs", nil, "GET /gcs-golfcommunity/api/v2/club/player?per-page=1000&include-stats=true"},
		{"golf.club-types", nil, "GET /gcs-golfcommunity/api/v2/club/types"},
		{"golf.leaderboards", page, "GET /gcs-golfcommunity/api/v2/leaderboard/player/played?page=1&per-page=20&user-locale=en"},
	})
}
