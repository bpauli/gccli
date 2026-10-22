package garminapi

import "net/http"

var localeUnits = []string{"METRIC", "STATUTE"}

var insightsEndpoints = []Endpoint{
	{
		Name: "insights.selections", Method: http.MethodGet, Host: ConnectAPI,
		Path: "/gcs-insights/api/demographics/selections?lang={lang}&localeUnit={localeUnit}",
		Params: []Param{
			{Name: "lang", Default: "en"},
			{Name: "localeUnit", Default: "METRIC", Enum: localeUnits},
		},
	},
	{
		Name: "insights.compare", Method: http.MethodGet, Host: ConnectAPI,
		Path: "/gcs-insights/api/demographics/compare?gender={gender}&age={age}&type={type}&field={field}&lang={lang}&localeUnit={localeUnit}&year={year}&month={month}",
		Params: []Param{
			{Name: "gender", Default: "ALL", Enum: []string{"ALL", "M", "F"}},
			{Name: "type", Enum: []string{"STEPS", "SLEEP", "FLOORS", "RUNNING", "CYCLING", "LAP_SWIMMING", "OPEN_WATER_SWIMMING"}},
			{Name: "field", Enum: []string{"AVG_SPEED", "VO2_MAX_VALUE", "AVG_POWER", "WEEKLY_AVG_DISTANCE", "WEEKLY_AVG_DURATION", "STEPS", "SLEEP", "FLOORS", "COUNT"}},
			{Name: "lang", Default: "en"},
			{Name: "localeUnit", Default: "METRIC", Enum: localeUnits},
			{Name: "month", Optional: true},
		},
	},
	{Name: "insights.preferences", Method: http.MethodGet, Host: ConnectAPI, Path: "/userpreference-service/demographicComparisonPreferences"},
}
