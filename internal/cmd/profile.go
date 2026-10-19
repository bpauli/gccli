package cmd

var profileGroup = Group{Name: "profile", Help: "View user profile.", Children: []Node{
	Leaf{Name: "view", Help: "Show user profile.", Default: true, Endpoint: "profile.get", Verb: "get profile", Output: JSON{}},
	Leaf{Name: "settings", Help: "Show user settings.", Endpoint: "profile.user-settings", Verb: "get user settings", Output: JSON{}},
}}
