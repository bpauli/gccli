package garminapi

import "net/http"

var groupsEndpoints = []Endpoint{
	{Name: "groups.list", Method: http.MethodGet, Host: ConnectAPI, Path: "/group-service/groups/{displayName}?includeMembership=true"},
	// groups.search takes keyword, start and limit as a form body. start is
	// a 1-based item index: start=0 and start=1 give the same first group.
	{Name: "groups.search", Method: http.MethodPost, Host: ConnectAPI, Path: "/group-service/keyword", Body: true, Form: true, Read: true},
	{Name: "groups.get", Method: http.MethodGet, Host: ConnectAPI, Path: "/group-service/group/{groupID}?includeMembership=true"},
	{Name: "groups.members", Method: http.MethodGet, Host: ConnectAPI, Path: "/group-service/group/{groupID}/members?supportOwner=true&orderByRelationship=true"},
	// groups.feed answers 207 with the feed, also for a member.
	{Name: "groups.feed", Method: http.MethodGet, Host: ConnectAPI, Path: "/web-gateway/group/v2/{groupID}/groupFeed"},
	{Name: "groups.activities", Method: http.MethodGet, Host: ConnectAPI, Path: "/web-gateway/group/v2/{groupID}/activityFeed"},
	// groups.challenges answers 403 ForbiddenException to an account that is
	// not a member of the group.
	{Name: "groups.challenges", Method: http.MethodGet, Host: ConnectAPI, Path: "/groupchallenge-service/group/{groupID}/groupchallenge/nonCompleted"},
}
