package garminapi

import "net/http"

var nutritionEndpoints = []Endpoint{
	{Name: "nutrition.daily-food-log", Method: http.MethodGet, Host: ConnectAPI, Path: "/nutrition-service/food/logs/{date}"},
	{Name: "nutrition.daily-meals", Method: http.MethodGet, Host: ConnectAPI, Path: "/nutrition-service/meals/{date}"},
	{Name: "nutrition.daily-settings", Method: http.MethodGet, Host: ConnectAPI, Path: "/nutrition-service/settings/{date}"},
	{Name: "nutrition.food-search", Method: http.MethodGet, Host: ConnectAPI, Path: "/nutrition-service/food/search?searchExpression={query}&start={start}&limit={limit}", Requires: ConnectPlus},
	{Name: "nutrition.recent-foods", Method: http.MethodGet, Host: ConnectAPI, Path: "/nutrition-service/food/recent/{date}?start={start}&limit={limit}", Requires: ConnectPlus},
}
