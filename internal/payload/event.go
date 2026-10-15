package payload

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// Event is the input of a new calendar event. Date is already resolved to
// YYYY-MM-DD. Distance takes values such as "10km" and Goal values such as
// "1h30m".
type Event struct {
	Name     string
	Date     string
	Type     string
	Race     bool
	Location string
	Time     string
	Timezone string
	Distance string
	Goal     string
	Primary  bool
	Training bool
	Private  bool
	Note     string
	URL      string
}

// JSON builds the body of the event.
func (e Event) JSON() (json.RawMessage, error) {
	body := map[string]any{
		"eventName": e.Name,
		"date":      e.Date,
		"eventType": e.Type,
		"race":      e.Race,
	}

	if e.Location != "" {
		body["location"] = e.Location
	}
	if e.Note != "" {
		body["note"] = e.Note
	}
	if e.URL != "" {
		body["url"] = e.URL
	}
	if e.Private {
		body["eventPrivacy"] = map[string]string{"label": "PRIVATE"}
	}

	if e.Time != "" {
		tl := map[string]string{"startTimeHhMm": e.Time}
		if e.Timezone != "" {
			tl["timeZoneId"] = e.Timezone
		}
		body["eventTimeLocal"] = tl
	}

	if e.Distance != "" {
		val, unit, err := parseDistance(e.Distance)
		if err != nil {
			return nil, err
		}
		body["completionTarget"] = map[string]any{
			"value":    val,
			"unit":     unit,
			"unitType": "distance",
		}
	}

	if e.Goal != "" || e.Primary || e.Training {
		cust := map[string]any{
			"isPrimaryEvent":  e.Primary,
			"isTrainingEvent": e.Training,
		}
		if e.Goal != "" {
			secs, err := parseGoalDuration(e.Goal)
			if err != nil {
				return nil, err
			}
			cust["customGoal"] = map[string]any{
				"value":    secs,
				"unit":     "second",
				"unitType": "time",
			}
		}
		body["eventCustomization"] = cust
	}

	return json.Marshal(body)
}

var distanceRe = regexp.MustCompile(`^([0-9]+(?:\.[0-9]+)?)\s*(km|mi|m)$`)

func parseDistance(s string) (float64, string, error) {
	m := distanceRe.FindStringSubmatch(s)
	if m == nil {
		return 0, "", fmt.Errorf("invalid distance %q (use e.g. 10km, 26.2mi, 400m)", s)
	}
	val, _ := strconv.ParseFloat(m[1], 64)
	units := map[string]string{"km": "kilometer", "mi": "mile", "m": "meter"}
	return val, units[m[2]], nil
}

func parseGoalDuration(s string) (int, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid goal duration %q (use e.g. 50m, 1h30m, 2400s)", s)
	}
	secs := int(d.Seconds())
	if secs <= 0 {
		return 0, fmt.Errorf("goal duration must be positive")
	}
	return secs, nil
}
