package cmd

var gearGroup = Group{Name: "gear", Help: "Manage gear.", Children: []Node{
	Leaf{
		Name: "list", Help: "List all gear.", Default: true,
		Endpoint: "gear.list",
		Params:   Params{"userProfilePK": ProfilePK},
		Verb:     "get gear", Output: JSON{},
	},
	Leaf{
		Name: "stats", Help: "Show usage statistics for a gear item.",
		Spec: argsOf(&struct {
			UUID string `arg:"" help:"Gear UUID."`
		}{}),
		Endpoint: "gear.stats",
		Params:   Params{"gearUUID": In("UUID")},
		Verb:     "get gear stats", Output: JSON{},
	},
	Leaf{
		Name: "activities", Help: "Show activities linked to a gear item.",
		Spec: argsOf(&struct {
			UUID  string `arg:"" help:"Gear UUID."`
			Limit int    `help:"Maximum number of activities to return." default:"20"`
		}{}),
		Endpoint: "gear.activities",
		Params:   Params{"gearUUID": In("UUID"), "limit": In("Limit")},
		Verb:     "get gear activities", Output: JSON{},
	},
	Leaf{
		Name: "defaults", Help: "Show default gear per activity type.",
		Endpoint: "gear.defaults",
		Params:   Params{"userProfileNumber": ProfilePK},
		Verb:     "get gear defaults", Output: JSON{},
	},
	Leaf{
		Name: "link", Help: "Link gear to an activity.",
		Spec: argsOf(&struct {
			UUID       string `arg:"" help:"Gear UUID."`
			ActivityID string `arg:"" help:"Activity ID." name:"activity-id"`
		}{}),
		Endpoint: "gear.link",
		Params:   Params{"gearUUID": In("UUID"), "activityID": In("ActivityID")},
		Verb:     "link gear",
		Output:   Write{Done: "Linked gear {UUID} to activity {ActivityID}"},
	},
	Leaf{
		Name: "unlink", Help: "Unlink gear from an activity.",
		Spec: argsOf(&struct {
			UUID       string `arg:"" help:"Gear UUID."`
			ActivityID string `arg:"" help:"Activity ID." name:"activity-id"`
		}{}),
		Endpoint: "gear.unlink",
		Params:   Params{"gearUUID": In("UUID"), "activityID": In("ActivityID")},
		Verb:     "unlink gear",
		Output:   Write{Done: "Unlinked gear {UUID} from activity {ActivityID}"},
	},
}}
