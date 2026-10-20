package cmd

import (
	"encoding/json"
	"fmt"
	"time"
)

// ActivityCreateCmd creates a manual activity entry.
type ActivityCreateCmd struct {
	Name     FreeText      `help:"Activity name." required:""`
	Type     FreeText      `help:"Activity type key (e.g. running, cycling, swimming)." required:""`
	Distance float64       `help:"Distance in meters."`
	Duration time.Duration `help:"Duration (e.g. 30m, 1h15m)." required:""`
	Date     DateTime      `help:"Start date and time in local time: YYYY-MM-DDTHH:MM:SS, YYYY-MM-DDTHH:MM or YYYY-MM-DD, with T or a space. Defaults to now."`
}

func (c *ActivityCreateCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	start := c.Date.orNow()
	data, err := client.CreateManualActivity(
		g.Context,
		string(c.Name),
		string(c.Type),
		start.Location().String(),
		c.Distance,
		c.Duration.Seconds(),
		start.Format("2006-01-02T15:04:05.000"),
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
