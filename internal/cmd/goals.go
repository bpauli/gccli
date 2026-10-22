package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/bpauli/gccli/internal/garminapi"
	"github.com/bpauli/gccli/internal/outfmt"
	"github.com/bpauli/gccli/internal/units"
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
	Custom{Name: "add", Help: "Add an activity goal.", Cmd: &GoalsAddCmd{}},
	Custom{Name: "update", Help: "Change an activity goal.", Cmd: &GoalsUpdateCmd{}},
	Leaf{
		Name: "delete", Help: "Delete an activity goal.",
		Spec: argsOf(&struct {
			ID ID `arg:"" help:"Goal ID, as goals list prints it."`
		}{}),
		Endpoint: "goals.delete",
		Params:   Params{"goalID": In("ID")},
		Verb:     "delete goal",
		Output:   Write{Done: "Deleted goal {ID}", Confirm: "Delete goal {ID}?"},
	},
	Leaf{
		Name: "effective", Help: "Show the daily goal of one type that applies on a date.",
		Spec: argsOf(&struct {
			Type string `arg:"" help:"Goal type: floors-climbed, intensity-minutes, hydration, steps or weight-grams." enum:"floors-climbed,intensity-minutes,hydration,steps,weight-grams"`
			Date Date   `help:"Date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today."`
		}{}),
		Endpoint: "goals.effective",
		Params:   Params{"goalType": In("Type"), "date": In("Date").Or(Today)},
		Verb:     "get goal", Output: JSON{},
	},
	Group{Name: "weight", Help: "View, set and delete the weight goal.", Children: []Node{
		Leaf{
			Name: "view", Help: "Show the effective weight goal for a date range.", Default: true,
			Spec: argsOf(&struct {
				StartDate Date `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today." name:"start"`
				EndDate   Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today." name:"end"`
			}{}),
			Endpoint: "goals.effective-weight-goal",
			Params:   Params{"startDate": In("StartDate").Or(Today), "endDate": In("EndDate").Or(Today)},
			Verb:     "get weight goal", Output: JSON{},
		},
		Custom{Name: "set", Help: "Set the weight goal.", Cmd: &GoalsWeightSetCmd{}},
		Custom{Name: "delete", Help: "Delete the current weight goal. An earlier weight goal then applies again.", Cmd: &GoalsWeightDeleteCmd{}},
	}},
	Custom{Name: "set", Help: "Set the daily goal for floors climbed, intensity minutes or hydration.", Cmd: &GoalsSetCmd{}},
	Group{Name: "steps", Help: "Set the daily steps goal.", Children: []Node{
		Custom{Name: "set", Help: "Set a fixed daily steps goal.", Cmd: &GoalsStepsSetCmd{}},
		Custom{Name: "auto", Help: "Set Garmin's automatic steps goal.", Cmd: &GoalsStepsAutoCmd{}},
	}},
}}

// goalTarget holds the four target flags of an activity goal. A goal counts
// exactly one of them.
type goalTarget struct {
	Distance   EventDistance `help:"Distance to cover (e.g. 50km, 30mi, 5000m)." placeholder:"DISTANCE"`
	Duration   Goal          `help:"Time to spend (e.g. 10h, 90m)." placeholder:"DURATION"`
	Calories   int           `help:"Kilocalories to burn." placeholder:"KCAL"`
	Activities int           `help:"Number of activities to do." placeholder:"N"`
}

var metersPer = map[string]float64{"kilometer": 1000, "mile": units.MetersPerMile, "meter": 1}

// target returns the one target that was given, or nil for none.
func (t goalTarget) target() (*garminapi.GoalTarget, error) {
	var given []garminapi.GoalTarget
	if t.Distance != (EventDistance{}) {
		meters := math.Round(t.Distance.Value * metersPer[t.Distance.Unit])
		if !(meters >= 1) {
			return nil, errors.New("--distance must be 1 m or more")
		}
		given = append(given, garminapi.DistanceGoal(meters))
	}
	if t.Duration != 0 {
		given = append(given, garminapi.DurationGoal(int(t.Duration)))
	}
	if t.Calories != 0 {
		if t.Calories < 0 {
			return nil, errors.New("--calories must be 1 or more")
		}
		given = append(given, garminapi.CalorieGoal(t.Calories))
	}
	if t.Activities != 0 {
		if t.Activities < 0 {
			return nil, errors.New("--activities must be 1 or more")
		}
		given = append(given, garminapi.ActivityCountGoal(t.Activities))
	}
	switch len(given) {
	case 0:
		return nil, nil
	case 1:
		return &given[0], nil
	}
	return nil, errors.New("give only one of --distance, --duration, --calories and --activities")
}

// goalPeriod is the Garmin form of a --period value: seven-days is seven_days.
func goalPeriod(p string) string { return strings.ReplaceAll(p, "-", "_") }

// checkMonthStart refuses a one_month goal that does not start on the first
// day of a month, which Garmin answers with 400.
func checkMonthStart(period string, start Date) error {
	if period == "one_month" && start != "" && !strings.HasSuffix(string(start), "-01") {
		return fmt.Errorf("a one-month goal starts on the first day of a month, not %s", start)
	}
	return nil
}

// checkGoalName refuses a name of 50 characters or more, as the web app does.
func checkGoalName(name FreeText) error {
	if n := len([]rune(string(name))); n > 49 {
		return fmt.Errorf("goal name has %d characters: the limit is 49", n)
	}
	return nil
}

// GoalsAddCmd adds an activity goal.
type GoalsAddCmd struct {
	Name         FreeText `arg:"" help:"Goal name, 1 to 49 characters."`
	Period       string   `help:"Goal period: seven-days, thirty-days, one-month or annual." required:"" enum:"seven-days,thirty-days,one-month,annual"`
	ActivityType string   `help:"Activity type key, as activities types prints it (e.g. running, cycling), or all." default:"all" name:"activity-type"`
	Start        Date     `help:"First day of the goal (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today, and to the first day of this month for a one-month goal."`
	goalTarget
}

func (c *GoalsAddCmd) Validate() error {
	if err := checkGoalName(c.Name); err != nil {
		return err
	}
	t, err := c.target()
	if err != nil {
		return err
	}
	if t == nil {
		return errors.New("give one of --distance, --duration, --calories and --activities")
	}
	return checkMonthStart(goalPeriod(c.Period), c.Start)
}

func (c *GoalsAddCmd) Run(g *Globals) error {
	t, err := c.target()
	if err != nil {
		return err
	}
	start := string(c.Start)
	if start == "" {
		today := nowFn()
		if c.Period == "one-month" {
			today = today.AddDate(0, 0, 1-today.Day())
		}
		start = today.Format("2006-01-02")
	}
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.CreateGoal(g.Context, garminapi.Goal{
		Name:         string(c.Name),
		ActivityType: c.ActivityType,
		Period:       goalPeriod(c.Period),
		StartDate:    start,
		Target:       *t,
	})
	if err != nil {
		return fmt.Errorf("add goal: %w", err)
	}
	var resp map[string]any
	_ = json.Unmarshal(data, &resp)
	if id := jsonString(resp, "id"); id != "" {
		return reportWrite(g, data, "Added goal %q (ID: %s)", c.Name, id)
	}
	return reportWrite(g, data, "Added goal %q", c.Name)
}

// GoalsUpdateCmd changes an activity goal.
type GoalsUpdateCmd struct {
	ID           ID       `arg:"" help:"Goal ID, as goals list prints it."`
	Name         FreeText `help:"New name, 1 to 49 characters."`
	Period       *string  `help:"New period: seven-days, thirty-days, one-month or annual." enum:"seven-days,thirty-days,one-month,annual"`
	ActivityType FreeText `help:"New activity type key, as activities types prints it (e.g. running, cycling), or all." name:"activity-type"`
	Start        Date     `help:"New first day of the goal (YYYY-MM-DD, today, yesterday, 3d, +30d)."`
	goalTarget
}

func (c *GoalsUpdateCmd) edit() (garminapi.GoalEdit, error) {
	t, err := c.target()
	if err != nil {
		return garminapi.GoalEdit{}, err
	}
	e := garminapi.GoalEdit{Name: string(c.Name), ActivityType: string(c.ActivityType), StartDate: string(c.Start), Target: t}
	if c.Period != nil {
		e.Period = goalPeriod(*c.Period)
	}
	return e, nil
}

func (c *GoalsUpdateCmd) Validate() error {
	e, err := c.edit()
	if err != nil {
		return err
	}
	if e == (garminapi.GoalEdit{}) {
		return errors.New("nothing to change: give --name, --period, --activity-type, --start, --distance, --duration, --calories or --activities")
	}
	if err := checkGoalName(c.Name); err != nil {
		return err
	}
	return checkMonthStart(e.Period, c.Start)
}

func (c *GoalsUpdateCmd) Run(g *Globals) error {
	e, err := c.edit()
	if err != nil {
		return err
	}
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.UpdateGoal(g.Context, string(c.ID), e)
	if err != nil {
		return fmt.Errorf("update goal: %w", err)
	}
	return reportWrite(g, data, "Updated goal %s", c.ID)
}

var gramsPer = map[string]float64{"kg": 1000, "lbs": units.GramsPerPound}

// GoalsWeightSetCmd sets the weight goal.
type GoalsWeightSetCmd struct {
	Weight float64 `arg:"" help:"Goal weight."`
	Unit   string  `help:"Unit of the weight: kg or lbs." default:"kg" enum:"kg,lbs"`
	Date   Date    `help:"First day of the goal (YYYY-MM-DD, today, yesterday, 3d). The goal stays for the days after it." default:"today"`
}

func (c *GoalsWeightSetCmd) Validate() error {
	// The negated check also refuses NaN.
	if !(c.Weight > 0 && c.Weight < 1000) {
		return fmt.Errorf("goal weight must be more than 0 and less than 1000 %s", c.Unit)
	}
	return nil
}

func (c *GoalsWeightSetCmd) Run(g *Globals) error {
	body, err := json.Marshal(map[string]any{
		"goalType":  "WEIGHT_GRAMS",
		"goalValue": math.Round(c.Weight * gramsPer[c.Unit]),
	})
	if err != nil {
		return err
	}
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "goals.set-effective", map[string]string{"goalType": "weight-grams", "date": string(c.Date)}, body)
	if err != nil {
		return fmt.Errorf("set weight goal: %w", err)
	}
	return reportWrite(g, data, "Set weight goal to %g %s from %s", c.Weight, c.Unit, c.Date)
}

// GoalsWeightDeleteCmd deletes the weight goal that Garmin returns as the
// current one. A DELETE of a goal that does not exist also answers 204, so
// the goal is read first: no weight goal is an error.
type GoalsWeightDeleteCmd struct {
	Force bool `help:"Skip confirmation prompt." short:"f"`
}

func (c *GoalsWeightDeleteCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "goals.weight-goal", nil, nil)
	if err != nil {
		return fmt.Errorf("get weight goal: %w", err)
	}
	if len(data) == 0 {
		return errors.New("no weight goal to delete")
	}
	var goal map[string]any
	if err := json.Unmarshal(data, &goal); err != nil {
		return fmt.Errorf("parse weight goal: %w", err)
	}
	pk := jsonString(goal, "userGoalPk")
	if pk == "" {
		return errors.New("parse weight goal: no userGoalPk")
	}
	ok, err := confirm(g.Stderr, fmt.Sprintf("Delete weight goal of %g kg from %s?", jsonFloat(goal, "goalValue")/1000, jsonString(goal, "startDate")), c.Force)
	if err != nil {
		return err
	}
	if !ok {
		g.UI.Infof("Cancelled")
		return nil
	}
	data, err = client.Call(g.Context, "goals.delete-user-goal", map[string]string{"userGoalPK": pk}, nil)
	if err != nil {
		return fmt.Errorf("delete weight goal: %w", err)
	}
	return reportWrite(g, data, "Deleted weight goal from %s", jsonString(goal, "startDate"))
}

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

var recordsGroup = Group{Name: "records", Help: "View personal records.", Children: []Node{
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
}}
