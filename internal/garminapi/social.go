package garminapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
)

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
}

// SearchUsers finds Garmin Connect users by name. Garmin takes the search
// as a form, not JSON, and counts start from 1.
func (c *Client) SearchUsers(ctx context.Context, keyword string, offset, limit int) (json.RawMessage, error) {
	form := url.Values{"keyword": {keyword}, "start": {strconv.Itoa(offset + 1)}, "limit": {strconv.Itoa(limit)}}
	resp, err := c.do(ctx, request{
		kind: kindJSON, method: http.MethodPost, path: "/usersearch-service/search/v3?displayMutedStatus=true",
		body: []byte(form.Encode()), contentType: "application/x-www-form-urlencoded; charset=UTF-8",
	})
	if err != nil {
		return nil, err
	}
	return jsonResult(resp), nil
}
