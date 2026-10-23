package garminapi

import "net/http"

var socialEndpoints = []Endpoint{
	{Name: "connections.list", Method: http.MethodGet, Host: ConnectAPI, Path: "/connection-service/connection/v2/connections/pagination?start={start}&limit={limit}"},
	{Name: "followers.list", Method: http.MethodGet, Host: ConnectAPI, Path: "/follower-service/follow/followers/{displayName}"},
	{Name: "followers.remove", Method: http.MethodDelete, Host: ConnectAPI, Path: "/follower-service/follow/followers/{displayName}"},
	{Name: "following.list", Method: http.MethodGet, Host: ConnectAPI, Path: "/follower-service/follow/followings/{displayName}"},
	{Name: "followers.mutuals", Method: http.MethodGet, Host: ConnectAPI, Path: "/follower-service/follow/mutuals"},
	{Name: "following.add", Method: http.MethodPost, Host: ConnectAPI, Path: "/follower-service/follow/followings/{displayName}"},
	{Name: "following.remove", Method: http.MethodDelete, Host: ConnectAPI, Path: "/follower-service/follow/followings/{displayName}"},
	{Name: "follow.status", Method: http.MethodGet, Host: ConnectAPI, Path: "/follower-service/follow/{displayName}/profileStatus"},
	{Name: "follow-requests.list", Method: http.MethodGet, Host: ConnectAPI, Path: "/follower-service/follow/pending"},
	// The pending list has no request IDs. The inbox has them, and req=8
	// keeps only the follow requests.
	{Name: "follow-requests.inbox", Method: http.MethodGet, Host: ConnectAPI, Path: "/web-gateway/inbox/v2/requests?req=8&cursor={cursor}", Params: []Param{{Name: "cursor", Optional: true}}},
	{Name: "follow-requests.accept", Method: http.MethodPut, Host: ConnectAPI, Path: "/follower-service/follow/requests/{requestID}/accept"},
	{Name: "follow-requests.decline", Method: http.MethodPut, Host: ConnectAPI, Path: "/follower-service/follow/requests/{requestID}/decline"},
	{Name: "muted.list", Method: http.MethodGet, Host: ConnectAPI, Path: "/follower-service/muteuser"},
	{Name: "muted.add", Method: http.MethodPost, Host: ConnectAPI, Path: "/follower-service/muteuser/{displayName}"},
	{Name: "muted.remove", Method: http.MethodDelete, Host: ConnectAPI, Path: "/follower-service/muteuser/{displayName}"},
	{Name: "connections.pending", Method: http.MethodGet, Host: ConnectAPI, Path: "/connection-service/connection/pending"},
	{Name: "newsfeed.list", Method: http.MethodGet, Host: ConnectAPI, Path: "/web-gateway/newsfeed/v3"},
	// Garmin takes the search only as a POST form. A GET answers 405 and a
	// JSON body 415.
	{Name: "users.search", Method: http.MethodPost, Host: ConnectAPI, Path: "/usersearch-service/search/v3?displayMutedStatus=true", Body: true, Form: true, Read: true},
}
