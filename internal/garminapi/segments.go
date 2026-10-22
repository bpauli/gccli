package garminapi

import "net/http"

var segmentsEndpoints = []Endpoint{
	{Name: "segments.search", Method: http.MethodPost, Host: ConnectAPI, Path: "/segment-service/segment/filterSegments", Body: true, Read: true},
	{Name: "segments.get", Method: http.MethodGet, Host: ConnectAPI, Path: "/segment-service/segment/{uuid}"},
	{Name: "segments.leaderboard", Method: http.MethodPost, Host: ConnectAPI, Path: "/segment-service/leaderboard/search/", Body: true, Read: true},
	{Name: "segments.rank", Method: http.MethodGet, Host: ConnectAPI, Path: "/segment-service/leaderboard/segment/{uuid}/rank/{displayName}"},
	{Name: "segments.classifications", Method: http.MethodGet, Host: ConnectAPI, Path: "/segment-service/segment/classifications"},
	{Name: "segments.surface-types", Method: http.MethodGet, Host: ConnectAPI, Path: "/segment-service/segment/surfaceTypes"},
	{Name: "segments.activity", Method: http.MethodGet, Host: ConnectAPI, Path: "/segment-service/segment/list/{activityID}"},
}
