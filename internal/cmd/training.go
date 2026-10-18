package cmd

var trainingGroup = Group{Name: "training", Help: "View training plans.", Children: []Node{
	Leaf{
		Name: "plans", Help: "List training plans.", Default: true,
		Spec: argsOf(&struct {
			Locale string `help:"Locale for training plans (e.g. en, de, fr)." default:"en"`
		}{}),
		Endpoint: "training.plans",
		Params:   Params{"locale": In("Locale")},
		Verb:     "get training plans", Output: JSON{},
	},
	Leaf{
		Name: "plan", Help: "Show a specific training plan.",
		Spec: argsOf(&struct {
			ID string `arg:"" help:"Training plan ID."`
		}{}),
		Endpoint: "training.plan",
		Params:   Params{"planID": In("ID")},
		Verb:     "get training plan", Output: JSON{},
	},
}}
