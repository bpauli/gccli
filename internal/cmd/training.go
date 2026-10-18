package cmd

import (
	"fmt"
)

var trainingGroup = Group{Name: "training", Help: "View training plans.", Children: []Node{
	Custom{Name: "plans", Help: "List training plans.", Default: true, Cmd: &TrainingPlansCmd{}},
	Custom{Name: "plan", Help: "Show a specific training plan.", Cmd: &TrainingPlanCmd{}},
}}

// TrainingPlansCmd lists training plans.
type TrainingPlansCmd struct {
	Locale string `help:"Locale for training plans (e.g. en, de, fr)." default:"en"`
}

func (c *TrainingPlansCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "training.plans", map[string]string{"locale": c.Locale}, nil)
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

	data, err := client.Call(g.Context, "training.plan", map[string]string{"planID": c.ID}, nil)
	if err != nil {
		return fmt.Errorf("get training plan: %w", err)
	}

	return writeHealthJSON(g, data)
}
