package cmd

var healthSnapshotsGroup = Group{Name: "snapshots", Help: "List and show Health Snapshot sessions.", Children: []Node{
	Leaf{
		Name: "list", Help: "List Health Snapshot sessions up to a date.", Default: true,
		Spec: argsOf(&struct {
			Pagination `embed:"" set:"items=sessions"`
			Until      Date `help:"Last date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
		}{}),
		Endpoint: "health.snapshots",
		Params:   Params{"limit": In("Limit"), "start": StartIndex, "until": In("Until").Or(Today)},
		Verb:     "get health snapshots", Output: JSON{List: true},
	},
	Leaf{
		Name: "show", Help: "Show the summary of a Health Snapshot session.",
		Spec: argsOf(&struct {
			Date Date `arg:"" help:"Date of the session (YYYY-MM-DD, today, yesterday, 3d)."`
			UUID ID   `arg:"" help:"Session UUID (activityUuid.uuid in 'health snapshots list')."`
		}{}),
		Endpoint: "health.snapshot",
		Params:   Params{"date": In("Date"), "uuid": In("UUID")},
		Verb:     "get health snapshot", Output: JSON{},
	},
	Leaf{
		Name: "epochs", Help: "Show the measurements of a Health Snapshot session.",
		Spec: argsOf(&struct {
			UUID ID `arg:"" help:"Session UUID (activityUuid.uuid in 'health snapshots list')."`
		}{}),
		Endpoint: "health.snapshot-epochs",
		Params:   Params{"uuid": In("UUID")},
		Verb:     "get health snapshot epochs", Output: JSON{},
	},
}}
