package garminapi

import "net/http"

var groupsEndpoints = []Endpoint{
	{Name: "groups.list", Method: http.MethodGet, Host: ConnectAPI, Path: "/group-service/groups/{displayName}?includeMembership=true"},
}
