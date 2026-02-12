package cmd

import (
	"fmt"
)

// GoalsCmd groups goal subcommands.
type GoalsCmd struct {
	List GoalsListCmd `cmd:"" default:"withargs" help:"List goals."`
}

// GoalsListCmd lists user goals.
type GoalsListCmd struct {
	Status string `help:"Filter by goal status (e.g. active, completed)." default:""`
}

func (c *GoalsListCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.GetGoals(g.Context, c.Status)
	if err != nil {
		return fmt.Errorf("get goals: %w", err)
	}

	return writeHealthJSON(g, data)
}

// BadgesCmd groups badge subcommands.
type BadgesCmd struct {
	Earned     BadgesEarnedCmd     `cmd:"" default:"withargs" help:"Show earned badges."`
	Available  BadgesAvailableCmd  `cmd:"" help:"Show available badges."`
	InProgress BadgesInProgressCmd `cmd:"" name:"in-progress" help:"Show badges in progress."`
}

// BadgesEarnedCmd shows earned badges.
type BadgesEarnedCmd struct{}

func (c *BadgesEarnedCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.GetBadgesEarned(g.Context)
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

	data, err := client.GetBadgesAvailable(g.Context)
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

	data, err := client.GetBadgesInProgress(g.Context)
	if err != nil {
		return fmt.Errorf("get in-progress badges: %w", err)
	}

	return writeHealthJSON(g, data)
}

// ChallengesCmd groups challenge subcommands.
type ChallengesCmd struct {
	List  ChallengesListCmd  `cmd:"" default:"withargs" help:"Show joined challenges."`
	Badge ChallengesBadgeCmd `cmd:"" help:"Show badge challenges."`
}

// ChallengesListCmd shows joined challenges.
type ChallengesListCmd struct{}

func (c *ChallengesListCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.GetChallenges(g.Context)
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

	data, err := client.GetBadgeChallenges(g.Context)
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

	displayName := client.Tokens().DisplayName
	if displayName == "" {
		displayName = client.Tokens().Email
	}

	data, err := client.GetPersonalRecords(g.Context, displayName)
	if err != nil {
		return fmt.Errorf("get personal records: %w", err)
	}

	return writeHealthJSON(g, data)
}
