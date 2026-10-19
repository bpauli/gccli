package cmd

var notificationsGroup = Group{Name: "notifications", Help: "View notifications.", Children: []Node{
	Leaf{Name: "list", Help: "List notifications.", Default: true, Endpoint: "notifications.list", Verb: "get notifications", Output: JSON{}},
}}
