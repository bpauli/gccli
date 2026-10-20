package cmd

// courseActivityTypes maps each --type value of courses import to Garmin's
// activity type PK. The enum tag of the flag lists the same keys, which
// TestCourseActivityTypesMatchEnum checks.
var courseActivityTypes = map[string]int{
	"running":         1,
	"cycling":         2,
	"hiking":          3,
	"other":           4,
	"mountain_biking": 5,
	"trail_running":   6,
	"street_running":  7,
	"walking":         9,
	"road_biking":     10,
	"gravel_cycling":  143,
	"e_bike_mountain": 175,
	"e_bike_fitness":  176,
	"ultra_run":       181,
}
