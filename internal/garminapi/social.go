package garminapi

import "net/http"

var socialEndpoints = []Endpoint{
	{Name: "connections.list", Method: http.MethodGet, Host: ConnectAPI, Path: "/connection-service/connection/v2/connections/pagination?start={start}&limit={limit}"},
	{Name: "followers.list", Method: http.MethodGet, Host: ConnectAPI, Path: "/follower-service/follow/followers/{displayName}"},
}
