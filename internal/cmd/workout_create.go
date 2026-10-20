package cmd

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/alecthomas/kong"

	"github.com/bpauli/gccli/internal/payload"
)

// WorkoutCreateCmd creates a workout with configurable sport type and targets.
type WorkoutCreateCmd struct {
	Name  FreeText `arg:"" help:"Workout name."`
	Type  string   `help:"Sport type." required:"" enum:"run,bike,swim,strength,cardio,hiit,yoga,pilates,mobility,multisport,custom"`
	Steps []Step   `help:"Workout step (type:duration[@target:values])." required:"" short:"s" name:"step"`
	Unit  string   `help:"Pace unit: km or mi." default:"km" enum:"km,mi"`
}

// Step is the Kong type of a workout step. Kong decodes it through the
// payload parser at parse time, so a bad step is a usage error before the
// login.
type Step payload.Step

func (s *Step) Decode(ctx *kong.DecodeContext) error {
	raw, err := popString(ctx, "step")
	if err != nil {
		return err
	}
	v, err := payload.ParseStep(raw)
	if err != nil {
		return err
	}
	*s = Step(v)
	return nil
}

// Validate rejects --step "", which Kong splits into no step at all.
func (c *WorkoutCreateCmd) Validate() error {
	if len(c.Steps) == 0 {
		return errors.New(`invalid step "": expected type:duration[@target:values]`)
	}
	return nil
}

func (c *WorkoutCreateCmd) Run(g *Globals) error {
	steps := make([]payload.Step, len(c.Steps))
	for i, s := range c.Steps {
		steps[i] = payload.Step(s)
	}
	body, err := payload.Workout(string(c.Name), c.Type, c.Unit, steps)
	if err != nil {
		return err
	}

	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "workouts.upload", nil, body)
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
