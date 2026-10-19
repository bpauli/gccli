package cmd

var socialGroup = Group{Name: "social", Help: "View connections and followers.", Children: []Node{
	Leaf{
		Name: "connections", Help: "List your connections.",
		Spec: argsOf(&struct {
			Pagination `embed:"" set:"items=connections"`
		}{}),
		Endpoint: "connections.list",
		Params:   Params{"start": In("Offset"), "limit": In("Limit")},
		Verb:     "get connections", Output: JSON{},
	},
	Leaf{
		Name: "followers", Help: "List the followers of a user.",
		Spec: argsOf(&struct {
			DisplayName string `arg:"" optional:"" help:"Display name of the user. Defaults to your display name."`
		}{}),
		Endpoint: "followers.list",
		Params:   Params{"displayName": In("DisplayName").Or(DisplayName)},
		Verb:     "get followers", Output: JSON{},
	},
}}
