package cmd

var profileGroup = Group{Name: "profile", Help: "View user profile.", Children: []Node{
	Leaf{Name: "view", Help: "Show user profile.", Default: true, Endpoint: "profile.get", Verb: "get profile", Output: JSON{}},
	Leaf{Name: "settings", Help: "Show user settings.", Endpoint: "profile.user-settings", Verb: "get user settings", Output: JSON{}},
	profilePrivacyGroup,
	profilePrivacyZonesGroup,
	Leaf{Name: "hr-zones", Help: "Show heart rate zones.", Endpoint: "profile.hr-zones", Verb: "get heart rate zones", Output: JSON{}},
	Leaf{Name: "power-zones", Help: "Show power zones for all sports.", Endpoint: "profile.power-zones", Verb: "get power zones", Output: JSON{}},
}}

var profilePrivacyGroup = Group{Name: "privacy", Help: "View privacy settings.", Children: []Node{
	Leaf{Name: "view", Help: "Show privacy settings.", Default: true, Endpoint: "profile.privacy-settings", Verb: "get privacy settings", Output: JSON{}},
}}

var profilePrivacyZonesGroup = Group{Name: "privacy-zones", Help: "View privacy zones.", Children: []Node{
	Leaf{Name: "list", Help: "List privacy zones.", Default: true, Endpoint: "profile.privacy-zones", Verb: "get privacy zones", Output: JSON{}},
}}
