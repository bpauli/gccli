package cmd

var activityCommentsGroup = Group{Name: "comments", Help: "List the comments on an activity.", Children: []Node{
	Leaf{
		Name: "list", Help: "List the comments on an activity.", Default: true,
		Spec: argsOf(&struct {
			ID ID `arg:"" help:"Activity ID."`
		}{}),
		Endpoint: "activities.comments",
		Params:   Params{"activityID": In("ID")},
		Verb:     "get comments", Output: JSON{List: true},
	},
}}

var activityLikesGroup = Group{Name: "likes", Help: "List the likes of an activity.", Children: []Node{
	Leaf{
		Name: "list", Help: "List the likes of an activity.", Default: true,
		Spec: argsOf(&struct {
			ID ID `arg:"" help:"Activity ID."`
		}{}),
		Endpoint: "activities.likes",
		Params:   Params{"activityID": In("ID")},
		Verb:     "get likes", Output: JSON{List: true},
	},
}}
