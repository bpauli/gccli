package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/bpauli/gccli/internal/garminapi"
)

var trainingGroup = Group{Name: "training", Help: "Find, join and change training plans, and view adaptive coaching workouts.", Children: []Node{
	Leaf{
		Name: "plans", Help: "List training plans.", Default: true,
		Spec: argsOf(&struct {
			Locale string `help:"Locale for training plans (e.g. en, de, fr)." default:"en"`
		}{}),
		Endpoint: "training.plans",
		Params:   Params{"locale": In("Locale")},
		Verb:     "get training plans", Output: JSON{},
	},
	Leaf{
		Name: "plan", Help: "Show a specific training plan.",
		Spec: argsOf(&struct {
			ID ID `arg:"" help:"Training plan ID."`
		}{}),
		Endpoint: "training.plan",
		Params:   Params{"planID": In("ID")},
		Verb:     "get training plan", Output: JSON{},
	},
	Leaf{
		Name: "active", Help: "Show active training plans.",
		Spec: argsOf(&struct {
			StartDate Date `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today." name:"start"`
		}{}),
		Endpoint: "training.active-plans",
		Params:   Params{"startDate": In("StartDate").Or(Today)},
		Verb:     "get active training plans", Output: JSON{},
	},
	Custom{Name: "search", Help: "Search the Garmin catalog of training plans.", Cmd: &TrainingSearchCmd{}},
	Leaf{
		Name: "schedule", Help: "List the workouts and notes of a training plan by week and day.",
		Spec: argsOf(&struct {
			ID     ID     `arg:"" help:"Training plan ID."`
			Locale string `help:"Locale for workout and note names (e.g. en, de, fr)." default:"en"`
		}{}),
		Endpoint: "training.schedule",
		Params:   Params{"planID": In("ID"), "locale": In("Locale")},
		Verb:     "get training plan schedule",
		Output: Rows{Noun: "training plan schedule", Columns: []Column{
			{Label: "WEEK", Path: "weekId"},
			{Label: "DAY", Path: "dayOfWeekId"},
			{Label: "DATE", Path: "calendarDate"},
			{Label: "TASK ID", Path: "taskWorkout.workoutScheduleId"},
			{Label: "WORKOUT", Path: "taskWorkout.workoutName"},
			{Label: "NOTE", Path: "taskNote.note"},
		}},
	},
	Custom{Name: "join", Help: "Join a catalog training plan and schedule its workouts in the calendar.", Cmd: &TrainingJoinCmd{}},
	Leaf{
		Name: "leave", Help: "Leave a joined training plan and remove its workouts from the calendar.",
		Spec: argsOf(&struct {
			ID ID `arg:"" help:"ID of the joined training plan, as training join or training active prints it."`
		}{}),
		Endpoint: "training.leave",
		Params:   Params{"planID": In("ID")},
		Verb:     "leave training plan",
		Output:   Write{Done: "Left training plan {ID}", Confirm: "Leave training plan {ID} and remove its workouts from the calendar?"},
	},
	trainingTaskGroup,
	Leaf{Name: "coach-workouts", Help: "List adaptive coaching workout types.", Endpoint: "training.adaptive-coaching-workouts", Verb: "get coach workouts", Output: JSON{List: true}},
}}

var trainingTaskGroup = Group{Name: "task", Help: "Move and delete the workouts of a joined training plan.", Children: []Node{
	Custom{Name: "move", Help: "Move a workout of a joined training plan to another day.", Cmd: &TrainingTaskMoveCmd{}},
	Leaf{
		Name: "delete", Help: "Delete a workout of a joined training plan. Garmin puts a rest day in its place.",
		Spec: argsOf(&struct {
			PlanID ID `arg:"" help:"ID of the joined training plan."`
			TaskID ID `arg:"" help:"Task ID of the workout, as training schedule prints it."`
		}{}),
		Endpoint: "training.delete-task",
		Params:   Params{"planID": In("PlanID"), "taskID": In("TaskID")},
		Verb:     "delete training plan task",
		Output:   Write{Done: "Deleted task {TaskID} of training plan {PlanID}", Confirm: "Delete task {TaskID} of training plan {PlanID}?"},
	},
}}

// trainingSearchRows prints the plans of a catalog search.
var trainingSearchRows = Rows{Noun: "training plans", Items: "trainingPlanList", Columns: []Column{
	{Label: "ID", Path: "trainingPlanId"},
	{Label: "NAME", Path: "name"},
	{Label: "SPORT", Path: "trainingType.typeKey"},
	{Label: "GOAL", Path: "trainingSubType.subTypeKey"},
	{Label: "LEVEL", Path: "trainingLevel.levelKey"},
	{Label: "TARGET", Path: "trainingVersion.versionName"},
	{Label: "WEEKS", Path: "durationInWeeks"},
}}

// TrainingSearchCmd searches the Garmin catalog of training plans.
type TrainingSearchCmd struct {
	Sport  string `help:"Sport as Garmin prints it in the SPORT column, such as Running."`
	Goal   string `help:"Goal as Garmin prints it in the GOAL column, such as 5k, 10k, HalfMarathon or Marathon."`
	Level  string `help:"Level as Garmin prints it in the LEVEL column, such as Beginner, Intermediate or Advanced."`
	Target string `help:"Workout target as Garmin prints it in the TARGET column, such as HeartRate or Time."`
	Locale string `help:"Locale for plan names (e.g. en, de, fr)." default:"en"`
}

func (c *TrainingSearchCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.SearchTrainingPlans(g.Context, garminapi.TrainingPlanFilter{
		Sport: c.Sport, Goal: c.Goal, Level: c.Level, Target: c.Target, Locale: c.Locale,
	})
	if err != nil {
		return fmt.Errorf("search training plans: %w", err)
	}
	return trainingSearchRows.print(g, client, data)
}

// TrainingJoinCmd joins a catalog plan. Garmin copies the plan into the
// account under a new ID, which the other plan commands take.
type TrainingJoinCmd struct {
	ID    ID   `arg:"" help:"Catalog training plan ID, as training search prints it."`
	Start Date `help:"Day of the first workout (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today."`
}

func (c *TrainingJoinCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	start := c.Start.orToday()
	data, err := client.Call(g.Context, "training.join", map[string]string{"planID": string(c.ID), "startDate": start}, nil)
	if err != nil {
		return fmt.Errorf("join training plan: %w", err)
	}
	var plan struct {
		ID json.Number `json:"trainingPlanId"`
	}
	if unmarshalAnswer(data, &plan) == nil && plan.ID != "" {
		return reportWrite(g, data, "Joined training plan %s from %s as plan %s", c.ID, start, plan.ID)
	}
	return reportWrite(g, data, "Joined training plan %s from %s", c.ID, start)
}

// TrainingTaskMoveCmd moves a workout of a joined plan to another day.
type TrainingTaskMoveCmd struct {
	PlanID ID   `arg:"" help:"ID of the joined training plan."`
	TaskID ID   `arg:"" help:"Task ID of the workout, as training schedule prints it."`
	Date   Date `arg:"" help:"New date (YYYY-MM-DD, today, yesterday, 3d, +30d)."`
}

func (c *TrainingTaskMoveCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.MoveTrainingPlanTask(g.Context, string(c.PlanID), string(c.TaskID), string(c.Date))
	if err != nil {
		return fmt.Errorf("move training plan task: %w", err)
	}
	var task struct {
		Workout struct {
			ID json.Number `json:"workoutScheduleId"`
		} `json:"taskWorkout"`
	}
	if unmarshalAnswer(data, &task) == nil && task.Workout.ID != "" {
		return reportWrite(g, data, "Moved task %s to %s (new task ID: %s)", c.TaskID, c.Date, task.Workout.ID)
	}
	return reportWrite(g, data, "Moved task %s to %s", c.TaskID, c.Date)
}
