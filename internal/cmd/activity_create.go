package cmd

import (
	"encoding/json"
	"fmt"
	"time"
)

// ActivityCreateCmd creates a manual activity entry.
type ActivityCreateCmd struct {
	Name     string        `help:"Activity name." required:""`
	Type     string        `help:"Activity type key (e.g. running, cycling, swimming)." required:""`
	Distance float64       `help:"Distance in meters."`
	Duration time.Duration `help:"Duration (e.g. 30m, 1h15m)." required:""`
	Date     string        `help:"Start date/time (e.g. 2024-06-15T07:30:00). Defaults to now."`
}

func (c *ActivityCreateCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	var startTime string
	var timezone string
	if c.Date != "" {
		t, err := parseDateTime(c.Date)
		if err != nil {
			return fmt.Errorf("parse date: %w", err)
		}
		startTime = t.Format("2006-01-02T15:04:05.000")
		timezone = t.Location().String()
	} else {
		now := nowFn()
		startTime = now.Format("2006-01-02T15:04:05.000")
		timezone = now.Location().String()
	}

	data, err := client.CreateManualActivity(
		g.Context,
		c.Name,
		c.Type,
		timezone,
		c.Distance,
		c.Duration.Seconds(),
		startTime,
	)
	if err != nil {
		return fmt.Errorf("create activity: %w", err)
	}

	var result map[string]any
	_ = json.Unmarshal(data, &result)
	if id := jsonString(result, "activityId"); id != "" {
		return reportWrite(g, data, "Created activity %s", id)
	}
	return reportWrite(g, data, "Activity created")
}

// parseDateTime parses a date/time string in various formats.
func parseDateTime(s string) (time.Time, error) {
	formats := []string{
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.ParseInLocation(f, s, nowFn().Location()); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported date format %q (use YYYY-MM-DDTHH:MM:SS)", s)
}
