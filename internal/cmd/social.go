package cmd

var socialGroup = Group{Name: "social", Help: "View connections, followers, following and the newsfeed.", Children: []Node{
	Leaf{
		Name: "connections", Help: "List your connections.",
		Spec: argsOf(&struct {
			Pagination `embed:"" set:"items=connections"`
		}{}),
		Endpoint: "connections.list",
		Params:   Params{"start": In("Offset"), "limit": In("Limit")},
		Verb:     "get connections", Output: JSON{List: true},
	},
	Leaf{
		Name: "followers", Help: "List the followers of a user.",
		Spec: argsOf(&struct {
			DisplayName string `arg:"" optional:"" help:"Display name of the user. Defaults to your display name."`
		}{}),
		Endpoint: "followers.list",
		Params:   Params{"displayName": In("DisplayName").Or(DisplayName)},
		Verb:     "get followers", Output: JSON{List: true},
	},
	Leaf{
		Name: "following", Help: "List the users that a user follows.",
		Spec: argsOf(&struct {
			DisplayName string `arg:"" optional:"" help:"Display name of the user. Defaults to your display name."`
		}{}),
		Endpoint: "following.list",
		Params:   Params{"displayName": In("DisplayName").Or(DisplayName)},
		Verb:     "get following", Output: JSON{List: true},
	},
	Leaf{Name: "requests", Help: "List pending connection requests.", Endpoint: "connections.pending", Verb: "get connection requests", Output: JSON{List: true}},
	Leaf{Name: "feed", Help: "Show your newsfeed.", Endpoint: "newsfeed.list", Verb: "get newsfeed", Output: JSON{}},
}}
