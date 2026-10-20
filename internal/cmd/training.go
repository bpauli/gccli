package cmd

var trainingGroup = Group{Name: "training", Help: "View training plans and adaptive coaching workouts.", Children: []Node{
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
			ID string `arg:"" help:"Training plan ID."`
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
	Leaf{Name: "coach-workouts", Help: "List adaptive coaching workout types.", Endpoint: "training.adaptive-coaching-workouts", Verb: "get coach workouts", Output: JSON{List: true}},
}}
