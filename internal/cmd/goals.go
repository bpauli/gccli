package cmd

import (
	"fmt"
)

var goalsGroup = Group{Name: "goals", Help: "View goals.", Children: []Node{
	Custom{Name: "list", Help: "List goals.", Default: true, Cmd: &GoalsListCmd{}},
}}

// GoalsListCmd lists user goals.
type GoalsListCmd struct {
	Status string `help:"Goal status: active, future, past." default:"active" enum:"active,future,past"`
}

func (c *GoalsListCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "goals.list", map[string]string{"status": c.Status}, nil)
	if err != nil {
		return fmt.Errorf("get goals: %w", err)
	}

	return writeHealthJSON(g, data)
}

var badgesGroup = Group{Name: "badges", Help: "View badges.", Children: []Node{
	Custom{Name: "earned", Help: "Show earned badges.", Default: true, Cmd: &BadgesEarnedCmd{}},
	Custom{Name: "available", Help: "Show available badges.", Cmd: &BadgesAvailableCmd{}},
	Custom{Name: "in-progress", Help: "Show badges in progress.", Cmd: &BadgesInProgressCmd{}},
}}

// BadgesEarnedCmd shows earned badges.
type BadgesEarnedCmd struct{}

func (c *BadgesEarnedCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "badges.earned", nil, nil)
	if err != nil {
		return fmt.Errorf("get earned badges: %w", err)
	}

	return writeHealthJSON(g, data)
}

// BadgesAvailableCmd shows available badges.
type BadgesAvailableCmd struct{}

func (c *BadgesAvailableCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "badges.available", nil, nil)
	if err != nil {
		return fmt.Errorf("get available badges: %w", err)
	}

	return writeHealthJSON(g, data)
}

// BadgesInProgressCmd shows badges in progress.
type BadgesInProgressCmd struct{}

func (c *BadgesInProgressCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "badges.in-progress", nil, nil)
	if err != nil {
		return fmt.Errorf("get in-progress badges: %w", err)
	}

	return writeHealthJSON(g, data)
}

var challengesGroup = Group{Name: "challenges", Help: "View challenges.", Children: []Node{
	Custom{Name: "list", Help: "Show joined challenges.", Default: true, Cmd: &ChallengesListCmd{}},
	Custom{Name: "badge", Help: "Show badge challenges.", Cmd: &ChallengesBadgeCmd{}},
}}

// ChallengesListCmd shows joined challenges.
type ChallengesListCmd struct{}

func (c *ChallengesListCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "challenges.joined", nil, nil)
	if err != nil {
		return fmt.Errorf("get challenges: %w", err)
	}

	return writeHealthJSON(g, data)
}

// ChallengesBadgeCmd shows badge challenges.
type ChallengesBadgeCmd struct{}

func (c *ChallengesBadgeCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "badges.challenges", nil, nil)
	if err != nil {
		return fmt.Errorf("get badge challenges: %w", err)
	}

	return writeHealthJSON(g, data)
}

// RecordsCmd shows personal records.
type RecordsCmd struct{}

func (c *RecordsCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	displayName, err := resolveDisplayName(g, client)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "records.list", map[string]string{"displayName": displayName}, nil)
	if err != nil {
		return fmt.Errorf("get personal records: %w", err)
	}

	return writeHealthJSON(g, data)
}
