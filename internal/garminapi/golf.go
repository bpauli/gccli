package garminapi

import "net/http"

var golfEndpoints = []Endpoint{
	{
		Name: "golf.scorecards", Method: http.MethodGet, Host: ConnectAPI,
		Path:   "/gcs-golfcommunity/api/v2/scorecard/summary?per-page={perPage}&page={page}&user-locale={locale}",
		Params: []Param{{Name: "locale", Default: "en"}},
	},
	{
		Name: "golf.courses", Method: http.MethodGet, Host: ConnectAPI,
		Path:   "/gcs-golfcommunity/api/v2/course-snapshot/player?per-page={perPage}&page={page}&user-locale={locale}",
		Params: []Param{{Name: "locale", Default: "en"}},
	},
	{
		Name: "golf.course", Method: http.MethodGet, Host: ConnectAPI,
		Path:   "/gcs-golfcommunity/api/v2/course-snapshot?course-snapshot-ids={id}&user-locale={locale}",
		Params: []Param{{Name: "locale", Default: "en"}},
	},
	{Name: "golf.stats", Method: http.MethodGet, Host: ConnectAPI, Path: "/gcs-golfcommunity/api/v2/player/stats"},
	{
		Name: "golf.shot-stats", Method: http.MethodGet, Host: ConnectAPI,
		Path:   "/gcs-golfcommunity/api/v2/shot/stats/{kind}",
		Params: []Param{{Name: "kind", Enum: []string{"drive", "approach", "chip", "putt"}}},
	},
	{Name: "golf.clubs", Method: http.MethodGet, Host: ConnectAPI, Path: "/gcs-golfcommunity/api/v2/club/player?per-page=1000&include-stats=true"},
	{Name: "golf.club-types", Method: http.MethodGet, Host: ConnectAPI, Path: "/gcs-golfcommunity/api/v2/club/types"},
	{Name: "golf.club-flex-types", Method: http.MethodGet, Host: ConnectAPI, Path: "/gcs-golfcommunity/api/v2/club/flex-types"},
	{
		Name: "golf.leaderboards", Method: http.MethodGet, Host: ConnectAPI,
		Path:   "/gcs-golfcommunity/api/v2/leaderboard/player/played?page={page}&per-page={perPage}&user-locale={locale}",
		Params: []Param{{Name: "locale", Default: "en"}},
	},
}
