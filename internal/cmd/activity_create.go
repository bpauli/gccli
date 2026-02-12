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
}

func (c *ActivityCreateCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	startTime := time.Now().UTC().Format("2006-01-02T15:04:05.000")

	data, err := client.CreateManualActivity(
		g.Context,
		c.Name,
		c.Type,
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
