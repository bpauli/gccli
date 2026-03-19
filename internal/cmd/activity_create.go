package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/bpauli/gccli/internal/outfmt"
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
		now := time.Now()
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

	if outfmt.IsJSON(g.Context) {
		return outfmt.WriteJSON(os.Stdout, json.RawMessage(data))
	}

	// Try to extract the created activity ID for user feedback.
	var result map[string]any
	if err := json.Unmarshal(data, &result); err == nil {
		if id, ok := result["activityId"]; ok {
			g.UI.Successf("Created activity %v", id)
			return nil
		}
	}

	g.UI.Successf("Activity created")
	return nil
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
		if t, err := time.ParseInLocation(f, s, time.Now().Location()); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported date format %q (use YYYY-MM-DDTHH:MM:SS)", s)
}
