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
	// groups.leaderboard needs start and limit, or Garmin answers 400. start
	// is a 1-based index. The wellness leaderboards answer 400 without
	// todayDate, and any date is accepted. They answer an empty metrics map
	// for an account that is not a member of the group.
	{
		Name: "groups.leaderboard", Method: http.MethodGet, Host: ConnectAPI,
		Path:   "/groupchallenge-service/leaderboard/{userType}/group/{groupID}?start={start}&limit={limit}&todayDate={todayDate}&startDate={startDate}&endDate={endDate}&metricId={metricID}&actTypeId={actTypeID}",
		Params: []Param{{Name: "userType", Enum: []string{"wellness", "activity"}}, {Name: "actTypeID", Optional: true}},
	},
	{Name: "groups.requests", Method: http.MethodGet, Host: ConnectAPI, Path: "/group-service/private/group/request/user/{displayName}"},
	// groups.join adds the account to a public group at once. A private
	// group needs a join request, which is a different request.
	{Name: "groups.join", Method: http.MethodPost, Host: ConnectAPI, Path: "/group-service/group/{groupID}/member/{profileID}", Body: true},
	{Name: "groups.leave", Method: http.MethodDelete, Host: ConnectAPI, Path: "/group-service/group/{groupID}/member/{profileID}"},
}
