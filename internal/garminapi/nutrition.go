package garminapi

import "net/http"

var nutritionEndpoints = []Endpoint{
	{Name: "nutrition.daily-food-log", Method: http.MethodGet, Host: ConnectAPI, Path: "/nutrition-service/food/logs/{date}"},
	{Name: "nutrition.daily-meals", Method: http.MethodGet, Host: ConnectAPI, Path: "/nutrition-service/meals/{date}"},
	{Name: "nutrition.daily-settings", Method: http.MethodGet, Host: ConnectAPI, Path: "/nutrition-service/settings/{date}"},
}
