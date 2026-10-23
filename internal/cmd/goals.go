package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/bpauli/gccli/internal/garminapi"
	"github.com/bpauli/gccli/internal/outfmt"
)

var goalsGroup = Group{Name: "goals", Help: "View and set goals.", Children: []Node{
	Leaf{
		Name: "list", Help: "List goals.", Default: true,
		Spec: argsOf(&struct {
			Status string `help:"Goal status: active, future, past." default:"active" enum:"active,future,past"`
		}{}),
		Endpoint: "goals.list",
		Params:   Params{"status": In("Status")},
		Verb:     "get goals", Output: JSON{List: true},
	},
	Leaf{
		Name: "weight", Help: "Show the effective weight goal for a date range.",
		Spec: argsOf(&struct {
			StartDate Date `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today." name:"start"`
			EndDate   Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today." name:"end"`
		}{}),
		Endpoint: "goals.effective-weight-goal",
		Params:   Params{"startDate": In("StartDate").Or(Today), "endDate": In("EndDate").Or(Today)},
		Verb:     "get weight goal", Output: JSON{},
	},
	Custom{Name: "set", Help: "Set the daily goal for floors climbed, intensity minutes or hydration.", Cmd: &GoalsSetCmd{}},
	Group{Name: "steps", Help: "Set the daily steps goal.", Children: []Node{
		Custom{Name: "set", Help: "Set a fixed daily steps goal.", Cmd: &GoalsStepsSetCmd{}},
		Custom{Name: "auto", Help: "Set Garmin's automatic steps goal.", Cmd: &GoalsStepsAutoCmd{}},
	}},
}}

// GoalsSetCmd sets the daily goal of one type.
type GoalsSetCmd struct {
	Type  string `arg:"" help:"Goal type: floors-climbed, intensity-minutes or hydration." enum:"floors-climbed,intensity-minutes,hydration"`
	Value int    `arg:"" help:"Daily goal: floors for floors-climbed, minutes for intensity-minutes, milliliters for hydration."`
	Date  Date   `help:"First day of the goal (YYYY-MM-DD, today, yesterday, 3d). The goal stays for the days after it." default:"today"`
}

func (c *GoalsSetCmd) Validate() error {
	if c.Value < 1 {
		return errors.New("goal must be 1 or more")
	}
	return nil
}

func (c *GoalsSetCmd) Run(g *Globals) error {
	body, err := json.Marshal(map[string]any{
		"goalType":  strings.ToUpper(strings.ReplaceAll(c.Type, "-", "_")),
		"goalValue": c.Value,
	})
	if err != nil {
		return err
	}
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "goals.set-effective", map[string]string{"goalType": c.Type, "date": string(c.Date)}, body)
	if err != nil {
		return fmt.Errorf("set goal: %w", err)
	}
	return reportWrite(g, data, "Set %s goal to %d from %s", c.Type, c.Value, c.Date)
}

// GoalsStepsSetCmd sets a fixed daily steps goal.
type GoalsStepsSetCmd struct {
	Steps int  `arg:"" help:"Steps per day."`
	Date  Date `help:"First day of the goal (YYYY-MM-DD, today, yesterday, 3d). The goal stays for the days after it." default:"today"`
}

func (c *GoalsStepsSetCmd) Validate() error {
	if c.Steps < 1 {
		return errors.New("steps goal must be 1 or more")
	}
	return nil
}

func (c *GoalsStepsSetCmd) Run(g *Globals) error {
	data, err := setStepGoal(g, c.Date, garminapi.ManualStepGoal(c.Steps))
	if err != nil {
		return err
	}
	return reportWrite(g, data, "Set steps goal to %d from %s", c.Steps, c.Date)
}

// GoalsStepsAutoCmd sets Garmin's automatic steps goal.
type GoalsStepsAutoCmd struct {
	Date Date `help:"First day of the goal (YYYY-MM-DD, today, yesterday, 3d). The goal stays for the days after it." default:"today"`
}

func (c *GoalsStepsAutoCmd) Run(g *Globals) error {
	data, err := setStepGoal(g, c.Date, garminapi.AutoStepGoal)
	if err != nil {
		return err
	}
	return reportWrite(g, data, "Set automatic steps goal from %s", c.Date)
}

func setStepGoal(g *Globals, date Date, goal garminapi.StepGoal) (json.RawMessage, error) {
	client, err := resolveClient(g)
	if err != nil {
		return nil, err
	}
	data, err := client.SetStepGoal(g.Context, string(date), goal)
	if err != nil {
		return nil, fmt.Errorf("set steps goal: %w", err)
	}
	return data, nil
}

var badgesGroup = Group{Name: "badges", Help: "View badges.", Children: []Node{
	Leaf{Name: "earned", Help: "Show earned badges.", Default: true, Endpoint: "badges.earned", Verb: "get earned badges", Output: JSON{List: true}},
	Leaf{Name: "available", Help: "Show available badges.", Endpoint: "badges.available", Verb: "get available badges", Output: JSON{List: true}},
	Custom{Name: "in-progress", Help: "Show badges in progress.", Cmd: &BadgesInProgressCmd{}},
	Leaf{
		Name: "detail", Help: "Show details of a badge.",
		Spec: argsOf(&struct {
			ID ID `arg:"" help:"Badge ID."`
		}{}),
		Endpoint: "badges.detail",
		Params:   Params{"badgeID": In("ID")},
		Verb:     "get badge detail", Output: JSON{},
	},
	Leaf{
		Name: "compare", Help: "Compare your badges with a connection.",
		Spec: argsOf(&struct {
			DisplayName ID `arg:"" help:"Display name of the connection."`
		}{}),
		Endpoint: "badges.compare",
		Params:   Params{"displayName": In("DisplayName")},
		Verb:     "compare badges", Output: JSON{},
	},
	Leaf{Name: "leaderboard", Help: "Show the badge points leaderboard of your connections.", Endpoint: "badges.leaderboard", Verb: "get badge leaderboard", Output: JSON{}},
}}

var challengesGroup = Group{Name: "challenges", Help: "View challenges.", Children: []Node{
	Custom{Name: "list", Help: "Show current challenges.", Default: true, Cmd: &ChallengesListCmd{}},
	Leaf{Name: "badge", Help: "Show joined badge challenges that are not completed.", Endpoint: "challenges.non-completed", Verb: "get badge challenges", Output: JSON{List: true}},
	Leaf{Name: "available", Help: "Show available badge challenges.", Endpoint: "challenges.available", Verb: "get available challenges", Output: JSON{List: true}},
	Leaf{Name: "completed", Help: "Show completed badge challenges.", Endpoint: "challenges.completed", Verb: "get completed challenges", Output: JSON{List: true}},
	Group{Name: "virtual", Help: "View virtual challenges.", Children: []Node{
		Leaf{Name: "available", Help: "Show available virtual challenges.", Endpoint: "challenges.virtual-available", Verb: "get available virtual challenges", Output: JSON{List: true}},
		Leaf{Name: "completed", Help: "Show completed virtual challenges.", Endpoint: "challenges.virtual-completed", Verb: "get completed virtual challenges", Output: JSON{List: true}},
		Leaf{Name: "in-progress", Help: "Show virtual challenges in progress.", Endpoint: "challenges.virtual-in-progress", Verb: "get in-progress virtual challenges", Output: JSON{List: true}},
	}},
}}

// BadgesInProgressCmd shows the badges in progress.
type BadgesInProgressCmd struct{}

func (c *BadgesInProgressCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.InProgressBadges(g.Context)
	if err != nil {
		return fmt.Errorf("get in-progress badges: %w", err)
	}
	return writeAnswer(g, data, "[]")
}

// ChallengesListCmd shows the current challenges as the web app lists them.
type ChallengesListCmd struct{}

func (c *ChallengesListCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.CurrentChallenges(g.Context)
	if err != nil {
		return fmt.Errorf("get challenges: %w", err)
	}
	return outfmt.WriteJSON(g.Stdout, data)
}

var recordsGroup = Group{Name: "records", Help: "View, hide and delete personal records.", Children: []Node{
	Leaf{
		Name: "list", Help: "View personal records.", Default: true,
		Endpoint: "records.list",
		Params:   Params{"displayName": DisplayName},
		Verb:     "get personal records", Output: JSON{List: true},
	},
	Leaf{
		Name: "types", Help: "List personal record types.",
		Endpoint: "records.types",
		Params:   Params{"displayName": DisplayName},
		Verb:     "get personal record types", Output: JSON{List: true},
	},
	Leaf{
		Name: "activity", Help: "List the personal records set in an activity.",
		Spec: argsOf(&struct {
			ID ID `arg:"" help:"Activity ID."`
		}{}),
		Endpoint: "records.activity",
		Params:   Params{"activityID": In("ID")},
		Verb:     "get activity records", Output: JSON{List: true},
	},
	Leaf{
		Name: "candidates", Help: "List the personal record candidates that wait for your review.",
		Endpoint: "records.candidates",
		Params:   Params{"displayName": DisplayName},
		Verb:     "get personal record candidates", Output: JSON{List: true},
	},
	Custom{Name: "hide", Help: "Hide a personal record type on the records page.", Cmd: &RecordsHideCmd{}},
	Custom{Name: "unhide", Help: "Show a hidden personal record type on the records page again.", Cmd: &RecordsUnhideCmd{}},
	Leaf{
		Name: "delete", Help: "Delete a personal record. This cannot be undone.",
		Spec: argsOf(&struct {
			ID NumericID `arg:"" help:"Personal record ID, as records list prints it."`
		}{}),
		Endpoint: "records.delete",
		Params:   Params{"recordID": In("ID")},
		Verb:     "delete personal record",
		Output:   Write{Done: "Deleted personal record {ID}", Confirm: "Delete personal record {ID}?"},
	},
}}

// RecordsHideCmd hides a personal record type on the records page.
type RecordsHideCmd struct {
	TypeID NumericID `arg:"" help:"Record type ID, as records types prints it."`
}

func (c *RecordsHideCmd) Run(g *Globals) error {
	data, err := setRecordTypeVisible(g, c.TypeID, false)
	if err != nil {
		return err
	}
	return reportWrite(g, data, "Hid personal record type %s", c.TypeID)
}

// RecordsUnhideCmd shows a hidden personal record type again.
type RecordsUnhideCmd struct {
	TypeID NumericID `arg:"" help:"Record type ID, as records types prints it."`
}

func (c *RecordsUnhideCmd) Run(g *Globals) error {
	data, err := setRecordTypeVisible(g, c.TypeID, true)
	if err != nil {
		return err
	}
	return reportWrite(g, data, "Unhid personal record type %s", c.TypeID)
}

func setRecordTypeVisible(g *Globals, id NumericID, visible bool) (json.RawMessage, error) {
	verb := "hide"
	if visible {
		verb = "unhide"
	}
	body, err := json.Marshal(map[string]any{"id": id, "visible": visible})
	if err != nil {
		return nil, err
	}
	client, err := resolveClient(g)
	if err != nil {
		return nil, err
	}
	data, err := client.Call(g.Context, "records.type.update", map[string]string{"typeID": id.String()}, body)
	if err != nil {
		return nil, fmt.Errorf("%s personal record type: %w", verb, err)
	}
	return data, nil
}
