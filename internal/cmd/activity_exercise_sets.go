package cmd

import (
	"errors"
	"fmt"

	"github.com/alecthomas/kong"

	"github.com/bpauli/gccli/internal/payload"
)

// ActivityExerciseSetsSetCmd sets exercise sets for a strength training activity.
type ActivityExerciseSetsSetCmd struct {
	ID       NumericID  `arg:"" help:"Activity ID."`
	Exercise []Exercise `help:"Exercise set in format CATEGORY/NAME:reps@weightkg[:dSECS][:rSECS] (e.g. BENCH_PRESS/BARBELL_BENCH_PRESS:12@20:d30:r60)." short:"e" required:""`
}

// Exercise is the Kong type of an exercise set. Kong decodes it through the
// payload parser at parse time, so a bad value is a usage error before the
// login.
type Exercise payload.Exercise

func (e *Exercise) Decode(ctx *kong.DecodeContext) error {
	raw, err := popString(ctx, "exercise")
	if err != nil {
		return err
	}
	v, err := payload.ParseExercise(raw)
	if err != nil {
		return fmt.Errorf("invalid exercise %q: %w", raw, err)
	}
	*e = Exercise(v)
	return nil
}

// Validate rejects --exercise "", which Kong splits into an empty list that
// passes the required check. Kong leaves the list nil when the flag is not
// given, and its own missing-flag error covers that case.
func (c *ActivityExerciseSetsSetCmd) Validate() error {
	if c.Exercise != nil && len(c.Exercise) == 0 {
		return errors.New("--exercise must not be empty")
	}
	return nil
}

func (c *ActivityExerciseSetsSetCmd) Run(g *Globals) error {
	exercises := make([]payload.Exercise, len(c.Exercise))
	for i, e := range c.Exercise {
		exercises[i] = payload.Exercise(e)
	}
	body, err := payload.ExerciseSets(int64(c.ID), exercises)
	if err != nil {
		return err
	}

	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "activities.set-exercise-sets", map[string]string{"activityID": c.ID.String()}, body)
	if err != nil {
		return fmt.Errorf("set exercise sets: %w", err)
	}
	return reportWrite(g, data, "Exercise sets updated for activity %s", c.ID)
}
