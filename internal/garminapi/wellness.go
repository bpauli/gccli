package garminapi

import "net/http"

var wellnessEndpoints = []Endpoint{
	{Name: "wellness.menstrual-cycle-data", Method: http.MethodGet, Host: ConnectAPI, Path: "/periodichealth-service/menstrualcycle/dayview/{startDate}/{endDate}"},
	{Name: "wellness.menstrual-cycle-summary", Method: http.MethodGet, Host: ConnectAPI, Path: "/periodichealth-service/menstrualcycle/summary/{startDate}/{endDate}"},
	{Name: "wellness.pregnancy-summary", Method: http.MethodGet, Host: ConnectAPI, Path: "/periodichealth-service/pregnancy/summary"},
	{Name: "wellness.request-reload", Method: http.MethodGet, Host: ConnectAPI, Path: "/wellness-service/wellness/epoch/request/{date}"},
}
