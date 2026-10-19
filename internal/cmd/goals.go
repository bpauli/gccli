package cmd

var goalsGroup = Group{Name: "goals", Help: "View goals.", Children: []Node{
	Leaf{
		Name: "list", Help: "List goals.", Default: true,
		Spec: argsOf(&struct {
			Status string `help:"Goal status: active, future, past." default:"active" enum:"active,future,past"`
		}{}),
		Endpoint: "goals.list",
		Params:   Params{"status": In("Status")},
		Verb:     "get goals", Output: JSON{},
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
}}

var badgesGroup = Group{Name: "badges", Help: "View badges.", Children: []Node{
	Leaf{Name: "earned", Help: "Show earned badges.", Default: true, Endpoint: "badges.earned", Verb: "get earned badges", Output: JSON{}},
	Leaf{Name: "available", Help: "Show available badges.", Endpoint: "badges.available", Verb: "get available badges", Output: JSON{}},
	Leaf{Name: "in-progress", Help: "Show badges in progress.", Endpoint: "badges.in-progress", Verb: "get in-progress badges", Output: JSON{}},
	Leaf{
		Name: "detail", Help: "Show details of a badge.",
		Spec: argsOf(&struct {
			ID string `arg:"" help:"Badge ID."`
		}{}),
		Endpoint: "badges.detail",
		Params:   Params{"badgeID": In("ID")},
		Verb:     "get badge detail", Output: JSON{},
	},
}}

var challengesGroup = Group{Name: "challenges", Help: "View challenges.", Children: []Node{
	Leaf{Name: "list", Help: "Show joined challenges.", Default: true, Endpoint: "challenges.joined", Verb: "get challenges", Output: JSON{}},
	Leaf{Name: "badge", Help: "Show badge challenges.", Endpoint: "badges.challenges", Verb: "get badge challenges", Output: JSON{}},
	Leaf{Name: "available", Help: "Show available badge challenges.", Endpoint: "challenges.available", Verb: "get available challenges", Output: JSON{}},
	Leaf{Name: "completed", Help: "Show completed badge challenges.", Endpoint: "challenges.completed", Verb: "get completed challenges", Output: JSON{}},
}}

var recordsGroup = Group{Name: "records", Help: "View personal records.", Children: []Node{
	Leaf{
		Name: "list", Help: "View personal records.", Default: true,
		Endpoint: "records.list",
		Params:   Params{"displayName": DisplayName},
		Verb:     "get personal records", Output: JSON{},
	},
	Leaf{
		Name: "types", Help: "List personal record types.",
		Endpoint: "records.types",
		Params:   Params{"displayName": DisplayName},
		Verb:     "get personal record types", Output: JSON{},
	},
	Leaf{
		Name: "activity", Help: "List the personal records set in an activity.",
		Spec: argsOf(&struct {
			ID string `arg:"" help:"Activity ID."`
		}{}),
		Endpoint: "records.activity",
		Params:   Params{"activityID": In("ID")},
		Verb:     "get activity records", Output: JSON{},
	},
}}
