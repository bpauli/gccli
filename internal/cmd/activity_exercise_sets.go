package cmd

import (
	"fmt"
	"strconv"

	"github.com/bpauli/gccli/internal/payload"
)

// ActivityExerciseSetsSetCmd sets exercise sets for a strength training activity.
type ActivityExerciseSetsSetCmd struct {
	ID       string   `arg:"" help:"Activity ID."`
	Exercise []string `help:"Exercise set in format CATEGORY/NAME:reps@weightkg[:dSECS][:rSECS] (e.g. BENCH_PRESS/BARBELL_BENCH_PRESS:12@20:d30:r60)." short:"e" required:""`
}

func (c *ActivityExerciseSetsSetCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	activityID, err := strconv.ParseInt(c.ID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid activity ID: %w", err)
	}

	body, err := payload.ExerciseSets(activityID, c.Exercise)
	if err != nil {
		return err
	}

	data, err := client.SetExerciseSets(g.Context, c.ID, body)
	if err != nil {
		return fmt.Errorf("set exercise sets: %w", err)
	}
	return reportWrite(g, data, "Exercise sets updated for activity %s", c.ID)
}
