package garminapi

import "net/http"

var gearEndpoints = []Endpoint{
	{Name: "gear.list", Method: http.MethodGet, Host: ConnectAPI, Path: "/gear-service/gear/filterGear?userProfilePk={userProfilePK}"},
	{Name: "gear.stats", Method: http.MethodGet, Host: ConnectAPI, Path: "/gear-service/gear/stats/{gearUUID}"},
	{Name: "gear.activities", Method: http.MethodGet, Host: ConnectAPI, Path: "/activitylist-service/activities/{gearUUID}/gear?limit={limit}"},
	{Name: "gear.defaults", Method: http.MethodGet, Host: ConnectAPI, Path: "/gear-service/gear/user/{userProfileNumber}/activityTypes"},
	{Name: "gear.link", Method: http.MethodPut, Host: ConnectAPI, Path: "/gear-service/gear/link/{gearUUID}/activity/{activityID}"},
	{Name: "gear.unlink", Method: http.MethodPut, Host: ConnectAPI, Path: "/gear-service/gear/unlink/{gearUUID}/activity/{activityID}"},
}
