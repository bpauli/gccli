package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/bpauli/gccli/internal/payload"
)

// WorkoutCreateCmd creates a workout with configurable sport type and targets.
type WorkoutCreateCmd struct {
	Name  string   `arg:"" help:"Workout name."`
	Type  string   `help:"Sport type." required:"" enum:"run,bike,swim,strength,cardio,hiit,yoga,pilates,mobility,multisport,custom"`
	Steps []string `help:"Workout step (type:duration[@target:values])." required:"" short:"s" name:"step"`
	Unit  string   `help:"Pace unit: km or mi." default:"km" enum:"km,mi"`
}

func (c *WorkoutCreateCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	body, err := payload.Workout(c.Name, c.Type, c.Unit, c.Steps)
	if err != nil {
		return err
	}

	data, err := client.UploadWorkout(g.Context, body)
	if err != nil {
		return fmt.Errorf("create workout: %w", err)
	}

	var resp map[string]any
	_ = json.Unmarshal(data, &resp)
	if id := jsonString(resp, "workoutId"); id != "" {
		return reportWrite(g, data, "Created workout %q (ID: %s)", c.Name, id)
	}
	return reportWrite(g, data, "Created workout %q", c.Name)
}
