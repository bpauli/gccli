package cmd

var notificationsGroup = Group{Name: "notifications", Help: "View notifications and notification preferences.", Children: []Node{
	Leaf{Name: "list", Help: "List notifications.", Default: true, Endpoint: "notifications.list", Verb: "get notifications", Output: JSON{}},
	Leaf{Name: "preferences", Help: "Show notification preferences.", Endpoint: "notifications.preferences", Verb: "get notification preferences", Output: JSON{}},
}}
