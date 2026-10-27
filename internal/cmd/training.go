package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
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
		Name: "coach-plan", Help: "Show a Garmin Coach plan, a plan of category FBT_ADAPTIVE.",
		Spec: argsOf(&struct {
			ID ID `arg:"" help:"Training plan ID, as training search or training active prints it."`
		}{}),
		Endpoint: "training.coach-plan",
		Params:   Params{"planID": In("ID")},
		Verb:     "get Garmin Coach plan", Output: JSON{},
	},
	Leaf{
		Name: "phased-plan", Help: "Show a joined phased training plan, an ITP plan with peak weeks. Garmin answers 400 for a plan that is not phased.",
		Spec: argsOf(&struct {
			ID ID `arg:"" help:"ID of the joined training plan, as training active prints it."`
		}{}),
		Endpoint: "training.phased-plan",
		Params:   Params{"planID": In("ID")},
		Verb:     "get phased training plan", Output: JSON{},
	},
	Leaf{
		Name: "strength-plan", Help: "Show a joined strength training plan, a plan of category STRENGTH. Garmin answers 400 for a plan that is not a strength plan.",
		Spec: argsOf(&struct {
			ID ID `arg:"" help:"ID of the joined training plan, as training active prints it."`
		}{}),
		Endpoint: "training.strength-plan",
		Params:   Params{"planID": In("ID")},
		Verb:     "get strength training plan", Output: JSON{},
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
	Custom{Name: "coach-join", Help: "Start a Garmin Coach plan from a template and schedule its workouts in the calendar.", Cmd: &TrainingCoachJoinCmd{}},
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
	{Label: "CATEGORY", Path: "trainingPlanCategory"},
	{Label: "SPORT", Path: "trainingType.typeKey"},
	{Label: "GOAL", Path: "trainingSubType.subTypeKey"},
	{Label: "LEVEL", Path: "trainingLevel.levelKey"},
	{Label: "TARGET", Path: "trainingVersion.versionName"},
	{Label: "WEEKS", Path: "durationInWeeks"},
}}

// TrainingSearchCmd searches the Garmin catalog of training plans.
type TrainingSearchCmd struct {
	Sport    string   `help:"Sport as Garmin prints it in the SPORT column, such as Running."`
	Goal     string   `help:"Goal as Garmin prints it in the GOAL column, such as 5k, 10k, HalfMarathon or Marathon."`
	Level    string   `help:"Level as Garmin prints it in the LEVEL column, such as Beginner, Intermediate or Advanced."`
	Target   string   `help:"Workout target as Garmin prints it in the TARGET column, such as HeartRate or Time."`
	Category []string `help:"Plan category: FBT_ADAPTIVE (Garmin Coach), STRENGTH, ITP or STATIC. Repeat the flag or separate with commas. Default: all categories." enum:"FBT_ADAPTIVE,STRENGTH,ITP,STATIC" placeholder:"CATEGORY"`
	Locale   string   `help:"Locale for plan names (e.g. en, de, fr)." default:"en"`
}

func (c *TrainingSearchCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	params := map[string]string{"locale": c.Locale, "sport": c.Sport, "goal": c.Goal, "level": c.Level, "target": c.Target, "category": strings.Join(c.Category, ",")}
	data, err := client.Call(g.Context, "training.search", params, []byte{})
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

// TrainingCoachJoinCmd starts a Garmin Coach plan from a catalog template.
type TrainingCoachJoinCmd struct {
	ID      NumericID `arg:"" help:"Catalog ID of a Garmin Coach template, as training search --category FBT_ADAPTIVE prints it."`
	Name    string    `help:"Name of the new plan." required:""`
	End     Date      `help:"Last day of the plan (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:""`
	Day     []string  `help:"Training day: MONDAY to SUNDAY. Repeat the flag or separate with commas. Each coach template sets how many days it needs, and Garmin rejects another number." required:"" enum:"MONDAY,TUESDAY,WEDNESDAY,THURSDAY,FRIDAY,SATURDAY,SUNDAY" placeholder:"DAY"`
	LongDay []string  `help:"Day for the long workout: MONDAY to SUNDAY. Repeat the flag or separate with commas." name:"long-day" enum:"MONDAY,TUESDAY,WEDNESDAY,THURSDAY,FRIDAY,SATURDAY,SUNDAY" placeholder:"DAY"`
	Locale  string    `help:"Locale for workout names (e.g. en, de, fr)." default:"en"`
}

type coachPlanRequest struct {
	TemplatePlanID            NumericID       `json:"templatePlanId"`
	CustomPlanName            string          `json:"customPlanName"`
	PlanEndDate               string          `json:"planEndDate"`
	AvailableTrainingDays     []string        `json:"availableTrainingDays"`
	PreferredLongTrainingDays []string        `json:"preferredLongTrainingDays"`
	LocaleKey                 string          `json:"localeKey"`
	WorkoutTargetTypePrefs    json.RawMessage `json:"workoutTargetTypePrefs"`
}

func (c *TrainingCoachJoinCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	prefs, err := client.Call(g.Context, "training.adaptive-coaching-workouts", nil, nil)
	if err != nil {
		return fmt.Errorf("get coach workouts: %w", err)
	}
	if prefs == nil {
		prefs = json.RawMessage("[]")
	}
	body, err := json.Marshal(coachPlanRequest{
		TemplatePlanID:            c.ID,
		CustomPlanName:            c.Name,
		PlanEndDate:               string(c.End),
		AvailableTrainingDays:     c.Day,
		PreferredLongTrainingDays: append([]string{}, c.LongDay...),
		LocaleKey:                 c.Locale,
		WorkoutTargetTypePrefs:    prefs,
	})
	if err != nil {
		return fmt.Errorf("join Garmin Coach plan: %w", err)
	}
	data, err := client.Call(g.Context, "training.coach-join", nil, body)
	if err != nil {
		return fmt.Errorf("join Garmin Coach plan: %w", err)
	}
	var plan struct {
		ID json.Number `json:"trainingPlanId"`
	}
	if unmarshalAnswer(data, &plan) == nil && plan.ID != "" {
		return reportWrite(g, data, "Joined Garmin Coach template %s as plan %s", c.ID, plan.ID)
	}
	return reportWrite(g, data, "Joined Garmin Coach template %s", c.ID)
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
	form := url.Values{"workoutSchedulePk": {string(c.TaskID)}, "scheduleDate": {string(c.Date)}}
	data, err := client.Call(g.Context, "training.move-task", map[string]string{"planID": string(c.PlanID)}, []byte(form.Encode()))
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
