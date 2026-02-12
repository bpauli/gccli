package cmd

import (
	"fmt"
)

// TrainingCmd groups training subcommands.
type TrainingCmd struct {
	Plans TrainingPlansCmd `cmd:"" default:"withargs" help:"List training plans."`
	Plan  TrainingPlanCmd  `cmd:"" help:"Show a specific training plan."`
}

// TrainingPlansCmd lists training plans.
type TrainingPlansCmd struct {
	Locale string `help:"Locale for training plans (e.g. en, de, fr)." default:"en"`
}

func (c *TrainingPlansCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.GetTrainingPlans(g.Context, c.Locale)
	if err != nil {
		return fmt.Errorf("get training plans: %w", err)
	}

	return writeHealthJSON(g, data)
}

// TrainingPlanCmd shows a specific training plan.
type TrainingPlanCmd struct {
	ID string `arg:"" help:"Training plan ID."`
}

func (c *TrainingPlanCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.GetTrainingPlan(g.Context, c.ID)
	if err != nil {
		return fmt.Errorf("get training plan: %w", err)
	}

	return writeHealthJSON(g, data)
}
