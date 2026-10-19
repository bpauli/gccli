package garminapi

import "net/http"

var socialEndpoints = []Endpoint{
	{Name: "connections.list", Method: http.MethodGet, Host: ConnectAPI, Path: "/connection-service/connection/v2/connections/pagination?start={start}&limit={limit}"},
	{Name: "followers.list", Method: http.MethodGet, Host: ConnectAPI, Path: "/follower-service/follow/followers/{displayName}"},
	{Name: "following.list", Method: http.MethodGet, Host: ConnectAPI, Path: "/follower-service/follow/followings/{displayName}"},
	{Name: "connections.pending", Method: http.MethodGet, Host: ConnectAPI, Path: "/connection-service/connection/pending"},
	{Name: "newsfeed.list", Method: http.MethodGet, Host: ConnectAPI, Path: "/web-gateway/newsfeed/v3"},
}
