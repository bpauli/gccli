package payload

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// Event is the input of a new calendar event. Date is already resolved to
// YYYY-MM-DD, and Time is HH:MM or empty.
type Event struct {
	Name     string
	Date     string
	Type     string
	Race     bool
	Location string
	Time     string
	Timezone string
	Distance Distance
	Goal     Goal
	Primary  bool
	Training bool
	Private  bool
	Note     string
	URL      string
}

// Distance is the completion target of an event, with Garmin's unit name.
// The zero value means no target.
type Distance struct {
	Value float64
	Unit  string
}

// Goal is the time goal of an event in whole seconds. Zero means no goal.
type Goal int

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
		if !timeRe.MatchString(e.Time) {
			return nil, fmt.Errorf("invalid time %q (use HH:MM, e.g. 07:30)", e.Time)
		}
		tl := map[string]string{"startTimeHhMm": e.Time}
		if e.Timezone != "" {
			tl["timeZoneId"] = e.Timezone
		}
		body["eventTimeLocal"] = tl
	}

	if e.Distance != (Distance{}) {
		body["completionTarget"] = map[string]any{
			"value":    e.Distance.Value,
			"unit":     e.Distance.Unit,
			"unitType": "distance",
		}
	}

	if e.Goal != 0 || e.Primary || e.Training {
		cust := map[string]any{
			"isPrimaryEvent":  e.Primary,
			"isTrainingEvent": e.Training,
		}
		if e.Goal != 0 {
			cust["customGoal"] = map[string]any{
				"value":    int(e.Goal),
				"unit":     "second",
				"unitType": "time",
			}
		}
		body["eventCustomization"] = cust
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("build event payload: %w", err)
	}
	return raw, nil
}

var (
	distanceRe = regexp.MustCompile(`^([0-9]+(?:\.[0-9]+)?)\s*(km|mi|m)$`)
	timeRe     = regexp.MustCompile(`^([01]?[0-9]|2[0-3]):[0-5][0-9]$`)
)

// ParseDistance parses a target distance such as "10km", "26.2mi" or "400m".
func ParseDistance(s string) (Distance, error) {
	m := distanceRe.FindStringSubmatch(s)
	if m == nil {
		return Distance{}, fmt.Errorf("invalid distance %q (use e.g. 10km, 26.2mi, 400m)", s)
	}
	val, _ := strconv.ParseFloat(m[1], 64)
	if !finite(val) {
		return Distance{}, fmt.Errorf("invalid distance %q: out of range", s)
	}
	units := map[string]string{"km": "kilometer", "mi": "mile", "m": "meter"}
	return Distance{Value: val, Unit: units[m[2]]}, nil
}

// ParseGoal parses a time goal such as "50m", "1h30m" or "2400s" into whole
// seconds. A goal under one second is not positive.
func ParseGoal(s string) (Goal, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid goal duration %q (use e.g. 50m, 1h30m, 2400s)", s)
	}
	secs := int(d.Seconds())
	if secs <= 0 {
		return 0, fmt.Errorf("goal duration must be positive")
	}
	return Goal(secs), nil
}
